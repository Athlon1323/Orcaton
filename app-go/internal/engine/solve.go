package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

type posKey struct {
	pos float64
	key OrderKey
}

func posKeyLess(a, b posKey) bool {
	if a.pos != b.pos {
		return a.pos < b.pos
	}
	return a.key.Less(b.key)
}

func orderBrokenIn(buckets map[string][]posKey) bool {
	for _, lst := range buckets {
		if len(lst) < 2 {
			continue
		}
		s := append([]posKey(nil), lst...)
		sort.SliceStable(s, func(i, j int) bool { return posKeyLess(s[i], s[j]) })
		for i := 1; i < len(s); i++ {
			if s[i].key.Less(s[i-1].key) {
				return true
			}
		}
	}
	return false
}

func (e *Engine) solve(units []*Unit, edges []*Edge) [][]int {
	g := newGroups(units, e.O.FileOrder)
	sorted := append([]*Edge(nil), edges...)
	sort.SliceStable(sorted, func(i, j int) bool { return -sorted[i].Score() < -sorted[j].Score() })
	var tree []*Edge
	for _, ed := range sorted {
		ra, rb := g.find(ed.A), g.find(ed.B)
		if ra == rb {
			continue
		}
		shift := g.rel[ed.A] + ed.Offset - g.rel[ed.B]
		if g.overlaps(g.members[ra], g.members[rb], shift) {
			e.Report.RejectedOverlap++
			continue
		}
		if g.orderBroken(g.members[ra], g.members[rb], shift) {
			e.Report.RejectedOrder++
			continue
		}
		g.join(ra, rb, shift)
		tree = append(tree, ed)
		for _, uid := range []int{ed.A, ed.B} {
			units[uid].BestScore = math.Max(units[uid].BestScore, ed.Score())
		}
	}
	e.Report.TreeEdges = tree
	comps := g.components()
	sortComps(units, comps)
	g.place(comps)
	e.Report.InconsistentEdges = e.inconsistentEdges(units, edges, tree)
	return comps
}

type groups struct {
	units     []*Unit
	fileOrder bool
	root      []int
	members   map[int][]int
	rel       []float64
}

func newGroups(units []*Unit, fileOrder bool) *groups {
	n := len(units)
	g := &groups{units: units, fileOrder: fileOrder, root: make([]int, n),
		members: make(map[int][]int, n), rel: make([]float64, n)}
	for i := 0; i < n; i++ {
		g.root[i] = i
		g.members[i] = []int{i}
	}
	return g
}

func (g *groups) find(x int) int {
	for g.root[x] != x {
		x = g.root[x]
	}
	return x
}

func (g *groups) join(ra, rb int, shift float64) {
	for _, j := range g.members[rb] {
		g.rel[j] += shift
	}
	g.members[ra] = append(g.members[ra], g.members[rb]...)
	delete(g.members, rb)
	g.root[rb] = ra
}

func (g *groups) overlaps(ga, gb []int, shift float64) bool {
	for _, i := range ga {
		ui := g.units[i]
		si, ei := g.rel[i], g.rel[i]+ui.dur
		for _, j := range gb {
			uj := g.units[j]
			if !ui.SharesTrack(uj) {
				continue
			}
			sj := g.rel[j] + shift
			ej := sj + uj.dur
			if math.Min(ei, ej)-math.Max(si, sj) > overlapTolF {
				return true
			}
		}
	}
	return false
}

func (g *groups) orderBroken(ga, gb []int, shift float64) bool {
	if !g.fileOrder {
		return false
	}
	buckets := map[string][]posKey{}
	for _, i := range ga {
		u := g.units[i]
		buckets[u.Device] = append(buckets[u.Device], posKey{g.rel[i], u.Order})
	}
	for _, j := range gb {
		u := g.units[j]
		buckets[u.Device] = append(buckets[u.Device], posKey{g.rel[j] + shift, u.Order})
	}
	return orderBrokenIn(buckets)
}

func (g *groups) components() [][]int {
	var comps [][]int
	for r := 0; r < len(g.root); r++ {
		m, ok := g.members[r]
		if ok && len(m) > 1 && g.find(r) == r {
			comps = append(comps, m)
		}
	}
	return comps
}

func (g *groups) place(comps [][]int) {
	for ci, comp := range comps {
		mn := math.Inf(1)
		for _, i := range comp {
			mn = math.Min(mn, g.rel[i])
		}
		for _, i := range comp {
			g.units[i].Position = num.FloatLimit(g.rel[i]-mn, 1_000_000)
			g.units[i].Status = core.StatusSynced
			g.units[i].Component = ci
		}
	}
}

func (e *Engine) inconsistentEdges(units []*Unit, edges, tree []*Edge) int {
	inTree := map[*Edge]bool{}
	for _, t := range tree {
		inTree[t] = true
	}
	tol := 2.0 / (float64(e.O.SampleRate) / float64(e.O.Hop))
	bad := 0
	for _, ed := range edges {
		pa, pb := units[ed.A].Position, units[ed.B].Position
		if inTree[ed] || pa == nil || pb == nil || units[ed.A].Component != units[ed.B].Component {
			continue
		}
		if math.Abs(num.Float(num.Sub(pb, pa))-ed.Offset) > tol {
			bad++
		}
	}
	return bad
}

func minOrder(units []*Unit, comp []int) OrderKey {
	best := units[comp[0]].Order
	for _, i := range comp[1:] {
		if units[i].Order.Before(best) {
			best = units[i].Order
		}
	}
	return best
}

func sortComps(units []*Unit, comps [][]int) {
	keys := make([]OrderKey, len(comps))
	for i, c := range comps {
		keys[i] = minOrder(units, c)
	}
	idx := make([]int, len(comps))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return keys[idx[a]].Before(keys[idx[b]]) })
	tmp := make([][]int, len(comps))
	for i, k := range idx {
		tmp[i] = comps[k]
	}
	copy(comps, tmp)
}

func (e *Engine) checkConflicts(units []*Unit, comps [][]int) {
	for _, comp := range comps {
		var trOrder []string
		byTrack := map[string][]*Unit{}
		for _, i := range comp {
			for _, tr := range units[i].Tracks {
				if _, ok := byTrack[tr]; !ok {
					trOrder = append(trOrder, tr)
				}
				byTrack[tr] = append(byTrack[tr], units[i])
			}
		}
		for _, tr := range trOrder {
			lst := byTrack[tr]
			sort.SliceStable(lst, func(a, b int) bool {
				return num.Lt(qOr0(lst[a].Position), qOr0(lst[b].Position))
			})
			for k := 1; k < len(lst); k++ {
				a, b := lst[k-1], lst[k]
				if a.Status != core.StatusSynced || b.Status != core.StatusSynced {
					continue
				}
				if num.Gt(num.Sub(num.Add(a.Position, a.Duration), b.Position), overlapTol) {
					loser := b
					if a.BestScore < b.BestScore {
						loser = a
					}
					loser.Status = core.StatusUnsynced
					loser.Position = nil
					loser.setError(ReasonConflict, "конфликт: наложение на своей дорожке")
					e.Report.Conflicts++
				}
			}
		}
	}
}
