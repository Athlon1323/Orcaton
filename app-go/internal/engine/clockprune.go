package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

func (e *Engine) pruneClockOutliers(units []*Unit, comps [][]int) [][]int {
	if !e.O.RecordTime || !e.O.FileOrder {
		return comps
	}
	out := make([][]int, 0, len(comps))
	var extra [][]int
	for ci, comp := range comps {
		drop := e.clockOutliersIn(units, comp)
		if len(drop) == 0 {
			out = append(out, comp)
			continue
		}
		gone := map[int]bool{}
		for _, i := range drop {
			gone[i] = true
			u := units[i]
			u.Status, u.Position, u.Component, u.BestScore = core.StatusUnsynced, nil, -1, 0
			u.setError(ReasonNoMatch, "звук противоречит часам камеры (вероятна ложная пара)")
			e.Report.ClockDropped++
		}
		parts := e.connectedParts(units, comp, gone)
		out = append(out, parts[0])
		extra = append(extra, parts[1:]...)
		_ = ci
	}
	out = append(out, extra...)
	var res [][]int
	for _, c := range out {
		if len(c) < 2 {
			for _, i := range c {
				u := units[i]
				u.Status, u.Position, u.Component, u.BestScore = core.StatusUnsynced, nil, -1, 0
				u.setError(ReasonNoMatch, "пара снята: звук партнёра противоречил часам камеры")
			}
			continue
		}
		res = append(res, c)
	}
	for ci, c := range res {
		mn := math.Inf(1)
		for _, i := range c {
			mn = math.Min(mn, num.Float(units[i].Position))
		}
		sh := num.FloatLimit(-mn, 1_000_000)
		for _, i := range c {
			units[i].Component = ci
			if mn != 0 {
				units[i].Position = num.Add(units[i].Position, sh)
			}
		}
	}
	var tree []*Edge
	for _, ed := range e.Report.TreeEdges {
		a, b := units[ed.A], units[ed.B]
		if a.Status == core.StatusSynced && b.Status == core.StatusSynced && a.Component == b.Component {
			tree = append(tree, ed)
		}
	}
	e.Report.TreeEdges = tree
	return res
}

func (e *Engine) clockOutliersIn(units []*Unit, comp []int) []int {
	byDev := map[string][]int{}
	var devs []string
	for _, i := range comp {
		u := units[i]
		if u.Status != core.StatusSynced || u.RecTime == nil || u.Position == nil {
			continue
		}
		if _, ok := byDev[u.Device]; !ok {
			devs = append(devs, u.Device)
		}
		byDev[u.Device] = append(byDev[u.Device], i)
	}
	var drop []int
	for _, d := range devs {
		m := byDev[d]
		if len(m) < 3 {
			continue
		}
		start := func(i int) float64 { return num.Float(units[i].Position) - *units[i].RecTime }
		agrees := func(i int, c float64) bool {
			s := start(i)
			return math.Abs(s-c) <= clockJumpSec || math.Abs(s+units[i].dur-c) <= clockJumpSec
		}
		best, bestN, bestZ := 0.0, 0, 0.0
		for _, i := range m {
			c := start(i)
			n, z := 0, 0.0
			for _, j := range m {
				if agrees(j, c) {
					n++
					z += units[j].BestScore
				}
			}
			if n > bestN || n == bestN && z > bestZ {
				best, bestN, bestZ = c, n, z
			}
		}
		var in []float64
		var out []int
		for _, j := range m {
			if agrees(j, best) {
				in = append(in, start(j))
			} else {
				out = append(out, j)
			}
		}
		if len(out) == 0 || bestN < 2 {
			continue
		}
		second := 0
		for _, i := range out {
			n := 0
			for _, j := range out {
				if agrees(j, start(i)) {
					n++
				}
			}
			second = max(second, n)
		}
		if bestN < 2*second {
			continue
		}
		center := median(in)
		for _, j := range out {
			if !agrees(j, center) {
				drop = append(drop, j)
			}
		}
	}
	sort.Ints(drop)
	return drop
}

func (e *Engine) connectedParts(units []*Unit, comp []int, gone map[int]bool) [][]int {
	in := map[int]bool{}
	for _, i := range comp {
		if !gone[i] {
			in[i] = true
		}
	}
	root := map[int]int{}
	var find func(int) int
	find = func(x int) int {
		if r, ok := root[x]; ok && r != x {
			r2 := find(r)
			root[x] = r2
			return r2
		}
		return x
	}
	tol := 2.0 / (float64(e.O.SampleRate) / float64(e.O.Hop))
	link := func(ed *Edge) {
		if !in[ed.A] || !in[ed.B] {
			return
		}
		pa, pb := units[ed.A].Position, units[ed.B].Position
		if pa == nil || pb == nil || math.Abs(num.Float(num.Sub(pb, pa))-ed.Offset) > tol {
			return
		}
		ra, rb := find(ed.A), find(ed.B)
		if ra != rb {
			root[rb] = ra
		}
	}
	for _, ed := range e.Report.TreeEdges {
		link(ed)
	}
	for _, ed := range e.Report.Edges {
		link(ed)
	}
	parts := map[int][]int{}
	var order []int
	for _, i := range comp {
		if !in[i] {
			continue
		}
		r := find(i)
		if _, ok := parts[r]; !ok {
			order = append(order, r)
		}
		parts[r] = append(parts[r], i)
	}
	res := make([][]int, 0, len(order))
	for _, r := range order {
		res = append(res, parts[r])
	}
	sort.SliceStable(res, func(a, b int) bool { return len(res[a]) > len(res[b]) })
	if len(res) == 0 {
		res = append(res, nil)
	}
	return res
}
