package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/dsp"
	"orcaton/internal/num"
)

type devIndex struct {
	by  map[string][]*Unit
	idx map[int]int
}

func newDevIndex(units []*Unit) *devIndex {
	_, by := byDevice(units)
	d := &devIndex{by: by, idx: map[int]int{}}
	for _, lst := range by {
		sortByOrder(lst)
		for i, u := range lst {
			d.idx[u.UID] = i
		}
	}
	return d
}

func (d *devIndex) neighbours(u *Unit, ci int) (*Unit, *Unit) {
	lst := d.by[u.Device]
	i := d.idx[u.UID]
	var prev, next *Unit
	for k := i - 1; k >= 0; k-- {
		if v := lst[k]; v.Component == ci && v.placed() {
			prev = v
			break
		}
	}
	for k := i + 1; k < len(lst); k++ {
		if v := lst[k]; v.Component == ci && v.placed() {
			next = v
			break
		}
	}
	return prev, next
}

func (d *devIndex) orderWindow(u *Unit, ci int) (float64, float64, bool) {
	prev, next := d.neighbours(u, ci)
	if prev == nil && next == nil {
		return 0, 0, false
	}
	lo, hi := math.Inf(-1), math.Inf(1)
	if prev != nil && prev.Position != nil {
		lo = num.Float(num.Add(prev.Position, prev.Duration))
	}
	if next != nil && next.Position != nil {
		hi = num.Float(num.Sub(next.Position, u.Duration))
	}
	if lo <= hi {
		return lo, hi, true
	}
	return 0, 0, false
}

func overlapsGroup(u *Unit, pos float64, comp []int, units []*Unit) bool {
	for _, j := range comp {
		v := units[j]
		if v.Position == nil || !u.SharesTrack(v) {
			continue
		}
		s := num.Float(v.Position)
		if math.Min(pos+u.dur, s+v.dur)-math.Max(pos, s) > overlapTolF {
			return true
		}
	}
	return false
}

type prepCache struct {
	m   map[int]*dsp.Prepared
	ln  bool
	fps float64
}

func (e *Engine) fullPrep() *prepCache {
	if e.prep == nil {
		e.prep = &prepCache{m: map[int]*dsp.Prepared{}}
	}
	return e.prep
}

func (e *Engine) clockPrep() *prepCache {
	if e.prepLN == nil {
		e.prepLN = &prepCache{m: map[int]*dsp.Prepared{}, ln: true,
			fps: float64(e.O.SampleRate) / float64(e.O.Hop)}
	}
	return e.prepLN
}

func (p *prepCache) get(u *Unit) *dsp.Prepared {
	if x, ok := p.m[u.UID]; ok {
		return x
	}
	f := u.Feature
	if p.ln {
		f = dsp.LocalNormalize(f, p.fps, localNormSec, localNormFloor)
	}
	x := dsp.NewPrepared(f)
	p.m[u.UID] = x
	return x
}

func (e *Engine) windowPass(units []*Unit, comps [][]int) {
	fps := float64(e.O.SampleRate) / float64(e.O.Hop)
	di := newDevIndex(units)
	prep := e.fullPrep()
	for round := 0; round < 5; round++ {
		found := false
		for _, u := range units {
			if u.Status == core.StatusSynced || u.Feature == nil {
				continue
			}
			type cand struct {
				ci, j  int
				lo, hi float64
			}
			var cands []cand
			for ci, comp := range comps {
				lo, hi, ok := di.orderWindow(u, ci)
				if !ok {
					continue
				}
				if clo, chi, has := e.clockWindow(u, comp, units); has {
					lo, hi = math.Max(lo, clo), math.Min(hi, chi)
					if lo > hi {
						continue
					}
				}
				for _, j := range comp {
					x := units[j]
					if x.Feature == nil || x.Position == nil || (x.Lane == u.Lane && !e.O.IntraLane) {
						continue
					}
					prep.get(x)
					cands = append(cands, cand{ci, j, lo, hi})
				}
			}
			pu := prep.get(u)
			results := make([]*dsp.OffsetResult, len(cands))
			cost := func(k int) int64 { return dsp.CorrBytes(prep.m[cands[k].j].Size, pu.Size) }
			parallelForMem(len(cands), e.workers(), e.gate, cost, func(k int) {
				cd := cands[k]
				px := num.Float(units[cd.j].Position)
				results[k] = dsp.XcorrWindow(prep.m[cd.j], pu, fps, cd.lo-px, cd.hi-px,
					e.O.MinOverlapSec, 0.25)
			})
			var best *dsp.OffsetResult
			bestCI, bestJ := -1, -1
			for k, res := range results {
				if res == nil || res.Confidence < e.O.WindowMinConf || res.Ratio < e.O.WindowRatio {
					continue
				}
				if best == nil || res.Confidence > best.Confidence {
					best, bestCI, bestJ = res, cands[k].ci, cands[k].j
				}
			}
			if best == nil {
				continue
			}
			pos := fOr0(units[bestJ].Position) + best.OffsetSec
			if overlapsGroup(u, pos, comps[bestCI], units) {
				continue
			}
			u.Position = num.FloatLimit(pos, 1_000_000)
			u.Status = core.StatusSynced
			u.Component = bestCI
			u.BestScore = best.Confidence
			u.clearError()
			comps[bestCI] = append(comps[bestCI], u.UID)
			e.Report.TreeEdges = append(e.Report.TreeEdges,
				&Edge{A: bestJ, B: u.UID, Offset: best.OffsetSec, Res: *best})
			e.Report.WindowSynced++
			found = true
		}
		if !found {
			break
		}
	}
}

const clockJumpSec = 60.0

func (e *Engine) clockWindow(u *Unit, comp []int, units []*Unit) (lo, hi float64, ok bool) {
	if !e.O.RecordTime || u.RecTime == nil {
		return 0, 0, false
	}
	var st, en []float64
	for _, j := range comp {
		x := units[j]
		if x.Device != u.Device || x.Status != core.StatusSynced || x.RecTime == nil || x.Position == nil {
			continue
		}
		p := num.Float(x.Position)
		st = append(st, p-*x.RecTime)
		en = append(en, p+x.dur-*x.RecTime)
	}
	if len(st) == 0 {
		return 0, 0, false
	}
	a := median(st) + *u.RecTime
	b := median(en) + *u.RecTime - u.dur
	tol := clockJumpSec
	return math.Min(a, b) - tol, math.Max(a, b) + tol, true
}

func (e *Engine) spanned(w, v *Unit) bool {
	if w.Device != v.Device || w.RecTime == nil || v.RecTime == nil {
		return false
	}
	if math.Abs(*v.RecTime-*w.RecTime-w.dur) > e.O.SpannedTolSec {
		return false
	}
	if w.TC != nil && v.TC != nil {
		return math.Abs(*v.TC-*w.TC-w.dur) <= 0.05
	}
	return true
}

type clockKey struct {
	ci  int
	dev string
}

type clockVal struct{ off, dur float64 }

func (e *Engine) clocks(units []*Unit) map[clockKey]clockVal {
	var order []clockKey
	calib := map[clockKey][]float64{}
	durs := map[clockKey][]float64{}
	for _, u := range units {
		if u.Status == core.StatusSynced && u.RecTime != nil && u.Position != nil {
			k := clockKey{u.Component, u.Device}
			if _, ok := calib[k]; !ok {
				order = append(order, k)
			}
			calib[k] = append(calib[k], num.Float(u.Position)-*u.RecTime)
			durs[k] = append(durs[k], u.dur)
		}
	}
	clock := map[clockKey]clockVal{}
	for _, k := range order {
		offs := calib[k]
		mn, mx := offs[0], offs[0]
		for _, v := range offs {
			mn, mx = math.Min(mn, v), math.Max(mx, v)
		}
		e.Report.ClockSpread = math.Max(e.Report.ClockSpread, mx-mn)
		if c, ok := robustClock(offs, e.O.ClockToleranceSec); ok {
			clock[k] = clockVal{c, median(durs[k])}
		}
	}
	return clock
}

func robustClock(offs []float64, tol float64) (float64, bool) {
	if len(offs) == 0 {
		return 0, false
	}
	med := median(offs)
	var in []float64
	for _, v := range offs {
		if math.Abs(v-med) <= tol {
			in = append(in, v)
		}
	}
	if len(offs) > 1 && len(in) < 2 || 3*len(in) < 2*len(offs) {
		return 0, false
	}
	return median(in), true
}

func (e *Engine) mergeByClock(units []*Unit, comps [][]int) [][]int {
	if len(comps) < 2 {
		return comps
	}
	clock := e.clocks(units)
	tol := e.O.ClockToleranceSec
	groups := make([][]int, len(comps))
	for i, c := range comps {
		groups[i] = append([]int(nil), c...)
	}
	alive := make([]bool, len(groups))
	for i := range alive {
		alive[i] = true
	}
	changed := true
	for changed {
		changed = false
		for a := 0; a < len(groups); a++ {
			for b := a + 1; b < len(groups); b++ {
				if !(alive[a] && alive[b]) {
					continue
				}
				var shifts []float64
				for k, ca := range clock {
					if k.ci != a {
						continue
					}
					if cb, ok := clock[clockKey{b, k.dev}]; ok && e.stampOK(k.dev, ca.dur, cb.dur) {
						shifts = append(shifts, ca.off-cb.off)
					}
				}
				if len(shifts) == 0 {
					continue
				}
				shift, ok := consensusShift(shifts, tol)
				if !ok {
					continue
				}
				shift = e.snapSpanned(units, groups[a], groups[b], shift)
				outlier := func(i int) bool { return e.clockOutlier(units[i], clock) }
				shift, drop, ok := e.fitJoin(units, groups[a], groups[b], shift, outlier)
				if !ok {
					continue
				}
				if len(drop) > 0 {
					groups[a], groups[b] = e.dropUnits(units, groups[a], drop), e.dropUnits(units, groups[b], drop)
				}
				sq := num.FloatLimit(shift, 1_000_000)
				for _, j := range groups[b] {
					units[j].Position = num.Add(units[j].Position, sq)
					units[j].Component = a
				}
				groups[a] = append(groups[a], groups[b]...)
				alive[b] = false
				for k, c := range clock {
					if k.ci == b {
						delete(clock, k)
						if _, ok := clock[clockKey{a, k.dev}]; !ok {
							clock[clockKey{a, k.dev}] = clockVal{c.off + shift, c.dur}
						}
					}
				}
				e.Report.ClockMerged++
				changed = true
			}
		}
	}
	var out [][]int
	for i, g := range groups {
		if alive[i] {
			out = append(out, g)
		}
	}
	sortComps(units, out)
	for ci, comp := range out {
		mn := qOr0(units[comp[0]].Position)
		for _, i := range comp[1:] {
			mn = num.Min(mn, qOr0(units[i].Position))
		}
		for _, i := range comp {
			units[i].Component = ci
			units[i].Position = num.Sub(qOr0(units[i].Position), mn)
		}
	}
	return out
}

func (e *Engine) fixEndStamps(units []*Unit) {
	tol := e.O.ClockToleranceSec
	type key struct {
		dev string
		ci  int
	}
	var keys []key
	offs := map[key][][2]float64{}
	for _, u := range units {
		if u.Status != core.StatusSynced || u.RecTime == nil || u.Position == nil {
			continue
		}
		k := key{u.Device, u.Component}
		if _, ok := offs[k]; !ok {
			keys = append(keys, k)
		}
		p := num.Float(u.Position) - *u.RecTime
		offs[k] = append(offs[k], [2]float64{p, p + u.dur})
	}
	type score struct{ start, end, n int }
	var devs []string
	byDev := map[string]*score{}
	for _, k := range keys {
		v := offs[k]
		if len(v) < 2 {
			continue
		}
		sc := byDev[k.dev]
		if sc == nil {
			sc = &score{}
			byDev[k.dev] = sc
			devs = append(devs, k.dev)
		}
		for side := 0; side < 2; side++ {
			xs := make([]float64, len(v))
			for i, o := range v {
				xs[i] = o[side]
			}
			n := inTol(xs, median(xs), tol)
			if side == 0 {
				sc.start += n
			} else {
				sc.end += n
			}
		}
		sc.n += len(v)
	}
	e.stampSure = map[string]bool{}
	for _, d := range devs {
		sc := byDev[d]
		if sc.start > sc.end && sc.start >= 2 && 3*sc.start >= 2*sc.n {
			e.stampSure[d] = true
		}
		if sc.end <= sc.start || sc.end < 2 || 3*sc.end < 2*sc.n {
			continue
		}
		e.stampSure[d] = true
		for _, u := range units {
			if u.Device != d || u.RecTime == nil {
				continue
			}
			t := *u.RecTime - u.dur
			if u.Order.T == *u.RecTime {
				u.Order.T = t
			}
			u.RecTime = &t
		}
		e.Report.EndStampCams = append(e.Report.EndStampCams, d)
	}
}

func (e *Engine) stampOK(dev string, calDur, dur float64) bool {
	return e.stampSure[dev] || math.Abs(calDur-dur) <= e.O.ClockToleranceSec
}

func (e *Engine) spannedNeighbour(u *Unit, ci int, di *devIndex) bool {
	prv, nxt := di.neighbours(u, ci)
	return prv != nil && prv.Position != nil && e.spanned(prv, u) ||
		nxt != nil && nxt.Position != nil && e.spanned(u, nxt)
}

func inTol(xs []float64, c, tol float64) int {
	n := 0
	for _, v := range xs {
		if math.Abs(v-c) <= tol {
			n++
		}
	}
	return n
}

func consensusShift(shifts []float64, tol float64) (float64, bool) {
	if len(shifts) == 1 {
		return shifts[0], true
	}
	var best []float64
	for _, c := range shifts {
		var m []float64
		for _, v := range shifts {
			if math.Abs(v-c) <= tol {
				m = append(m, v)
			}
		}
		if len(m) > len(best) {
			best = m
		}
	}
	if len(best) < 2 || 2*len(best) <= len(shifts) {
		return 0, false
	}
	return median(best), true
}

func (e *Engine) snapSpanned(units []*Unit, ga, gb []int, shift float64) float64 {
	tol := e.O.ClockToleranceSec
	for _, i := range ga {
		for _, j := range gb {
			ui, uj := units[i], units[j]
			pi, pj := fOr0(ui.Position), fOr0(uj.Position)
			var exact float64
			switch {
			case e.spanned(ui, uj):
				exact = pi + ui.dur - pj
			case e.spanned(uj, ui):
				exact = pi - pj - uj.dur
			default:
				continue
			}
			if math.Abs(exact-shift) <= tol {
				return exact
			}
		}
	}
	return shift
}

const maxClockDrops = 3

func (e *Engine) fitJoin(units []*Unit, ga, gb []int, shift float64, outlier func(int) bool) (float64, []int, bool) {
	if s, ok := e.nearestJoin(units, ga, gb, shift); ok {
		return s, nil, true
	}
	drop := map[int]bool{}
	for _, c := range joinOverlaps(units, ga, gb, shift) {
		oi, oj := outlier(c[0]), outlier(c[1])
		switch {
		case oi && !oj:
			drop[c[0]] = true
		case oj && !oi:
			drop[c[1]] = true
		default:
			return shift, nil, false
		}
	}
	if len(drop) == 0 || len(drop) > maxClockDrops {
		return shift, nil, false
	}
	keep := func(g []int) []int {
		var out []int
		for _, i := range g {
			if !drop[i] {
				out = append(out, i)
			}
		}
		return out
	}
	s, ok := e.nearestJoin(units, keep(ga), keep(gb), shift)
	if !ok {
		return shift, nil, false
	}
	var ids []int
	for i := range drop {
		ids = append(ids, i)
	}
	sort.Ints(ids)
	return s, ids, true
}

func (e *Engine) nearestJoin(units []*Unit, ga, gb []int, shift float64) (float64, bool) {
	tol := e.O.ClockToleranceSec
	cands := []float64{shift}
	for _, i := range ga {
		for _, j := range gb {
			ui, uj := units[i], units[j]
			if !ui.SharesTrack(uj) {
				continue
			}
			pi, pj := fOr0(ui.Position), fOr0(uj.Position)
			for _, c := range []float64{pi - pj - uj.dur, pi + ui.dur - pj} {
				if math.Abs(c-shift) <= tol {
					cands = append(cands, c)
				}
			}
		}
	}
	sort.SliceStable(cands, func(x, y int) bool { return math.Abs(cands[x]-shift) < math.Abs(cands[y]-shift) })
	for k, c := range cands {
		if k >= 64 {
			break
		}
		if e.canJoin(units, ga, gb, c) {
			return c, true
		}
	}
	return shift, false
}

func joinOverlaps(units []*Unit, ga, gb []int, shift float64) [][2]int {
	var out [][2]int
	for _, i := range ga {
		for _, j := range gb {
			ui, uj := units[i], units[j]
			if !ui.SharesTrack(uj) {
				continue
			}
			pi, pj := fOr0(ui.Position), fOr0(uj.Position)+shift
			if math.Min(pi+ui.dur, pj+uj.dur)-math.Max(pi, pj) > overlapTolF {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

func (e *Engine) clockOutlier(u *Unit, clock map[clockKey]clockVal) bool {
	if u.Status != core.StatusSynced || u.RecTime == nil || u.Position == nil {
		return false
	}
	c, ok := clock[clockKey{u.Component, u.Device}]
	return ok && math.Abs(num.Float(u.Position)-*u.RecTime-c.off) > e.O.ClockToleranceSec
}

func (e *Engine) dropUnits(units []*Unit, g []int, drop []int) []int {
	gone := map[int]bool{}
	for _, i := range drop {
		gone[i] = true
	}
	var out []int
	for _, i := range g {
		if !gone[i] {
			out = append(out, i)
			continue
		}
		u := units[i]
		u.Status, u.Position, u.Component = core.StatusUnsynced, nil, -1
		u.setError(ReasonNoMatch, "звук противоречит часам камеры (вероятна ложная пара)")
		e.Report.ClockDropped++
	}
	return out
}

func (e *Engine) canJoin(units []*Unit, ga, gb []int, shift float64) bool {
	pos := map[int]float64{}
	var order []int
	for _, i := range ga {
		pos[i] = fOr0(units[i].Position)
		order = append(order, i)
	}
	for _, j := range gb {
		if _, ok := pos[j]; !ok {
			order = append(order, j)
		}
		pos[j] = fOr0(units[j].Position) + shift
	}
	for _, i := range ga {
		for _, j := range gb {
			ui, uj := units[i], units[j]
			if !ui.SharesTrack(uj) {
				continue
			}
			if math.Min(pos[i]+ui.dur, pos[j]+uj.dur)-math.Max(pos[i], pos[j]) > overlapTolF {
				return false
			}
		}
	}
	buckets := map[string][]posKey{}
	for _, i := range order {
		buckets[units[i].Device] = append(buckets[units[i].Device], posKey{pos[i], units[i].Order})
	}
	return !orderBrokenIn(buckets)
}

func (e *Engine) timePass(units []*Unit, comps [][]int) {
	fps := float64(e.O.SampleRate) / float64(e.O.Hop)
	di := newDevIndex(units)
	for round := 0; round < 5; round++ {
		clock := e.clocks(units)
		found := false
		for _, u := range units {
			if u.placed() || u.RecTime == nil || u.errorIs("нет пути к файлу") || u.DupOf != nil {
				continue
			}
			ci, lo, hi, ok := e.clockGroup(u, len(comps), clock, di)
			if !ok {
				continue
			}
			est, exact := e.clockEstimate(u, ci, clock, lo, hi, di)
			pos, res, partner := est, (*dsp.OffsetResult)(nil), -1
			if u.Feature != nil {
				pos, res, partner = e.audioNearClock(u, units, comps[ci], est, lo, hi, fps)
			}
			if overlapsGroup(u, pos, comps[ci], units) {
				continue
			}
			e.placeByClock(u, ci, comps, pos, exact, res, partner)
			found = true
		}
		if !found {
			break
		}
	}
}

func (e *Engine) clockGroup(u *Unit, nComps int, clock map[clockKey]clockVal, di *devIndex) (int, float64, float64, bool) {
	bestCI := -1
	var bestViol, lo, hi float64
	for ci := 0; ci < nComps; ci++ {
		c, ok := clock[clockKey{ci, u.Device}]
		if !ok || !e.stampOK(u.Device, c.dur, u.dur) && !e.spannedNeighbour(u, ci, di) {
			continue
		}
		wlo, whi, ok := di.orderWindow(u, ci)
		if !ok {
			continue
		}
		est := *u.RecTime + c.off
		viol := math.Max(math.Max(wlo-est, est-whi), 0.0)
		if viol > e.O.ClockToleranceSec {
			continue
		}
		if bestCI < 0 || viol < bestViol {
			bestCI, bestViol, lo, hi = ci, viol, wlo, whi
		}
	}
	return bestCI, lo, hi, bestCI >= 0
}

func (e *Engine) clockEstimate(u *Unit, ci int, clock map[clockKey]clockVal, lo, hi float64, di *devIndex) (float64, num.Q) {
	est := math.Min(math.Max(*u.RecTime+clock[clockKey{ci, u.Device}].off, lo), hi)
	prv, nxt := di.neighbours(u, ci)
	var exact num.Q
	if prv != nil && prv.Position != nil && e.spanned(prv, u) {
		exact = num.Add(prv.Position, prv.Duration)
	} else if nxt != nil && nxt.Position != nil && e.spanned(u, nxt) {
		exact = num.Sub(nxt.Position, u.Duration)
	}
	if exact != nil {
		est = num.Float(exact)
	}
	return est, exact
}

func (e *Engine) audioNearClock(u *Unit, units []*Unit, comp []int, est, lo, hi, fps float64) (
	float64, *dsp.OffsetResult, int) {
	preps := []*prepCache{e.fullPrep()}
	preps = append(preps, e.clockPrep())
	r := e.O.TimeSearchSec
	var js []int
	for _, j := range comp {
		x := units[j]
		if x.Feature == nil || x.Position == nil || x.Status != core.StatusSynced ||
			(x.Lane == u.Lane && !e.O.IntraLane) {
			continue
		}
		js = append(js, j)
	}
	for _, prep := range preps {
		for _, j := range js {
			prep.get(units[j])
		}
		pu := prep.get(u)
		results := make([]*dsp.OffsetResult, len(js))
		cost := func(k int) int64 { return dsp.CorrBytes(prep.m[js[k]].Size, pu.Size) }
		parallelForMem(len(js), e.workers(), e.gate, cost, func(k int) {
			px := num.Float(units[js[k]].Position)
			results[k] = dsp.XcorrWindow(prep.m[js[k]], pu, fps, math.Max(lo, est-r)-px,
				math.Min(hi, est+r)-px, e.O.MinOverlapSec, e.O.TimeMinActivity)
		})
		pos, partner := est, -1
		var res *dsp.OffsetResult
		for k, cand := range results {
			if cand == nil || cand.Confidence < e.O.TimeMinConf || cand.Ratio < e.O.TimeRatio {
				continue
			}
			if res == nil || cand.Confidence > res.Confidence {
				px := num.Float(units[js[k]].Position)
				res, partner, pos = cand, js[k], px+cand.OffsetSec
			}
		}
		if res != nil {
			return pos, res, partner
		}
	}
	return e.nearClockShiftTest(u, units, js, est, lo, hi, fps)
}

const (
	nearClockSec = 1.0
	shiftWindows = 16
	shiftMargin  = 1.15
	localRatio   = 1.15
	localExclSec = 0.25
)

func (e *Engine) nearClockShiftTest(u *Unit, units []*Unit, js []int, est, lo, hi, fps float64) (
	float64, *dsp.OffsetResult, int) {
	prep := e.fullPrep()
	pu := prep.get(u)
	r := e.O.TimeSearchSec
	type cand struct {
		j           int
		z, off, rat float64
	}
	var best *cand
	for _, j := range js {
		px := num.Float(units[j].Position)
		lo0, z := dsp.WindowScores(prep.get(units[j]), pu, fps, math.Max(lo, est-r)-px, math.Min(hi, est+r)-px,
			e.O.MinOverlapSec, e.O.TimeMinActivity)
		i, zi := dsp.ArgMax(z)
		if zi < e.O.TimeMinConf {
			continue
		}
		off := float64(lo0+i) / fps
		rat := dsp.RatioAround(z, i, int(localExclSec*fps))
		if math.Abs(px+off-est) > nearClockSec || rat < localRatio {
			continue
		}
		if best == nil || zi > best.z {
			best = &cand{j: j, z: zi, off: off, rat: rat}
		}
	}
	if best == nil {
		return est, nil, -1
	}
	px := num.Float(units[best.j].Position)
	pa := prep.m[best.j]
	c := est - px
	noise, n := math.Inf(-1), 0
	for k := 0; k < 4*shiftWindows && n < shiftWindows; k++ {
		d := 15 + 37*float64(k/2)
		if d > 900 {
			break
		}
		if k%2 == 1 {
			d = -d
		}
		_, z := dsp.WindowScores(pa, pu, fps, c+d-r, c+d+r, e.O.MinOverlapSec, e.O.TimeMinActivity)
		if _, zi := dsp.ArgMax(z); !math.IsInf(zi, -1) {
			noise = math.Max(noise, zi)
			n++
		}
	}
	if n < shiftWindows/2 || best.z < shiftMargin*noise {
		return est, nil, -1
	}
	res := dsp.XcorrWindow(pa, pu, fps, best.off-localExclSec, best.off+localExclSec,
		e.O.MinOverlapSec, e.O.TimeMinActivity)
	if res == nil {
		return est, nil, -1
	}
	res.Ratio = best.rat
	e.Report.TimeShiftTested++
	return px + res.OffsetSec, res, best.j
}

func (e *Engine) placeByClock(u *Unit, ci int, comps [][]int, pos float64, exact num.Q,
	res *dsp.OffsetResult, partner int) {
	if exact != nil && res == nil {
		u.Position = exact
	} else {
		u.Position = num.FloatLimit(pos, 1_000_000)
	}
	u.Status = core.StatusByTimecode
	if res != nil {
		u.Status = core.StatusSynced
	}
	u.Component = ci
	u.clearError()
	comps[ci] = append(comps[ci], u.UID)
	if res != nil {
		u.BestScore = res.Confidence
		e.Report.TreeEdges = append(e.Report.TreeEdges,
			&Edge{A: partner, B: u.UID, Offset: res.OffsetSec, Res: *res})
		e.Report.TimeAudio++
	} else {
		e.Report.TimePlaced++
	}
}
