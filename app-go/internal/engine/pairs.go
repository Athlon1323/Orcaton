package engine

import (
	"context"
	"math"
	"sync"

	"orcaton/internal/dsp"
)

func (e *Engine) correlate(ctx context.Context, units []*Unit) []*Edge {
	fps := float64(e.O.SampleRate) / float64(e.O.Hop)
	k := e.O.CoarseFactor
	if k < 1 {
		k = 1
	}
	pl := e.pairPlan(units, k)
	results := e.coarse(ctx, pl, fps/float64(k))
	if ctx.Err() != nil {
		return nil
	}
	thr, noise := e.noiseThreshold(results)
	floor := thr * e.O.AmbiguityFloor
	var above []*Edge
	for _, r := range results {
		if r.Score() >= floor {
			above = append(above, r)
		}
	}
	cands := e.refine(ctx, units, above, fps)
	var edges []*Edge
	for _, c := range cands {
		if c.CoarseZ >= thr && c.Res.Ratio >= e.O.MinRatio {
			edges = append(edges, c)
		}
	}
	edges = e.dropAmbiguous(units, cands, edges)
	e.Report.TriedPairs = pl.nPairs
	e.Report.Threshold = thr
	e.Report.NoiseLevel = noise
	e.Report.Edges = edges
	return edges
}

type pairs struct {
	cand     []*Unit
	prep     map[int]*dsp.Prepared
	anchors  []int
	byAnchor map[int][]*Unit
	nPairs   int
}

func (e *Engine) pairPlan(units []*Unit, k int) *pairs {
	pl := &pairs{prep: map[int]*dsp.Prepared{}, byAnchor: map[int][]*Unit{}}
	for _, u := range units {
		if u.Feature != nil {
			pl.cand = append(pl.cand, u)
		}
	}
	for _, u := range pl.cand {
		pl.prep[u.UID] = dsp.NewPrepared(dsp.PoolFeature(u.Feature, k))
	}
	for i, a := range pl.cand {
		for _, b := range pl.cand[i+1:] {
			if !e.O.IntraLane && a.Lane == b.Lane {
				continue
			}
			x, y := a, b
			if pl.prep[a.UID].Size < pl.prep[b.UID].Size {
				x, y = b, a
			}
			if _, ok := pl.byAnchor[x.UID]; !ok {
				pl.anchors = append(pl.anchors, x.UID)
			}
			pl.byAnchor[x.UID] = append(pl.byAnchor[x.UID], y)
			pl.nPairs++
		}
	}
	return pl
}

func (e *Engine) coarse(ctx context.Context, pl *pairs, fpsC float64) []*Edge {
	perAnchor := make([][]*Edge, len(pl.anchors))
	var mu sync.Mutex
	done := 0
	step := max(1, (pl.nPairs+99)/100)
	e.emit(StagePairs, 0, pl.nPairs, "")
	longest := func(ai int) int {
		mx := 0
		for _, p := range pl.byAnchor[pl.anchors[ai]] {
			mx = max(mx, pl.prep[p.UID].Size)
		}
		return mx
	}
	cost := func(ai int) int64 { return dsp.CorrBytes(pl.prep[pl.anchors[ai]].Size, longest(ai)) }
	parallelForMem(len(pl.anchors), e.workers(), e.gate, cost, func(ai int) {
		if ctx.Err() != nil {
			return
		}
		ax := pl.anchors[ai]
		partners := pl.byAnchor[ax]
		pa := pl.prep[ax]
		nfft := dsp.NextFastLen(pa.Size + longest(ai) - 1)
		var res []*Edge
		for _, y := range partners {
			r := dsp.XcorrPrepared(pa, pl.prep[y.UID], fpsC, e.O.MinOverlapSec, e.O.MaxOffsetSec,
				0.25, nfft)
			if r != nil {
				res = append(res, &Edge{A: ax, B: y.UID, Offset: r.OffsetSec, Res: *r})
			}
			mu.Lock()
			done++
			if done%step == 0 || done == pl.nPairs {
				e.emit(StagePairs, done, pl.nPairs, "")
			}
			mu.Unlock()
		}
		pa.DropCache()
		perAnchor[ai] = res
	})
	var results []*Edge
	for _, r := range perAnchor {
		results = append(results, r...)
	}
	for _, u := range pl.cand {
		pl.prep[u.UID].DropCache()
	}
	return results
}

func (e *Engine) noiseThreshold(results []*Edge) (thr, noise float64) {
	thr = e.O.MinConfidence
	if !e.O.AutoThreshold || len(results) < 30 {
		return thr, 0
	}
	zs := make([]float64, len(results))
	for i, r := range results {
		zs[i] = r.Score()
	}
	med := median(zs)
	dev := make([]float64, len(zs))
	for i, z := range zs {
		dev[i] = math.Abs(z - med)
	}
	mad := median(dev) * 1.4826
	noise = med + e.O.NoiseK*mad
	return math.Max(thr, noise), noise
}

func (e *Engine) refine(ctx context.Context, units []*Unit, edges []*Edge, fps float64) []*Edge {
	var out []*Edge
	e.emit(StageRefine, 0, len(edges), "")
	if e.O.CoarseFactor <= 1 {
		for _, x := range edges {
			out = append(out, &Edge{x.A, x.B, x.Offset, x.Res, x.Score()})
		}
		e.emit(StageRefine, len(edges), len(edges), "")
		return out
	}
	full := e.fullPrep()
	fp := func(uid int) *dsp.Prepared { return full.get(units[uid]) }
	for _, x := range edges {
		if ctx.Err() != nil {
			return out
		}
		r := dsp.RefinePrepared(fp(x.A), fp(x.B), fps, x.Offset, e.O.RefineRadiusSec,
			e.O.MinOverlapSec, 0.25)
		if r == nil {
			out = append(out, &Edge{x.A, x.B, x.Offset, x.Res, x.Score()})
			continue
		}
		out = append(out, &Edge{x.A, x.B, r.OffsetSec, dsp.OffsetResult{
			OffsetSec: r.OffsetSec, Confidence: r.Confidence, Peak: r.Peak, Ratio: x.Res.Ratio,
			OverlapSec: r.OverlapSec}, x.Score()})
	}
	e.emit(StageRefine, len(edges), len(edges), "")
	return out
}

type claim struct {
	y   int
	pos float64
	z   float64
}

func (e *Engine) dropAmbiguous(units []*Unit, claimsSrc, edges []*Edge) []*Edge {
	claims := map[int]map[string][]claim{}
	add := func(x int, lane string, c claim) {
		if claims[x] == nil {
			claims[x] = map[string][]claim{}
		}
		claims[x][lane] = append(claims[x][lane], c)
	}
	for _, x := range claimsSrc {
		if x.Res.Ratio < e.O.CompetitorRatio {
			continue
		}
		add(x.A, units[x.B].Lane, claim{x.B, x.Offset, x.Score()})
		add(x.B, units[x.A].Lane, claim{x.A, -x.Offset, x.Score()})
	}
	contested := func(x, y int, posY, z float64) bool {
		uy := units[y]
		for _, c := range claims[x][uy.Lane] {
			if c.y == y || !uy.SharesTrack(units[c.y]) {
				continue
			}
			ov := math.Min(posY+uy.dur, c.pos+units[c.y].dur) - math.Max(posY, c.pos)
			if ov > overlapTolF && z < e.O.Dominance*c.z {
				return true
			}
		}
		return false
	}
	var kept []*Edge
	for _, x := range edges {
		if contested(x.A, x.B, x.Offset, x.Score()) || contested(x.B, x.A, -x.Offset, x.Score()) {
			e.Report.Ambiguous++
			continue
		}
		kept = append(kept, x)
	}
	return kept
}

func parallelFor(n, workers int, f func(i int)) {
	if workers <= 1 || n <= 1 {
		for i := 0; i < n; i++ {
			f(i)
		}
		return
	}
	ch := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(workers, n); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				f(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()
}
