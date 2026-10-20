package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

func (e *Engine) serializeSingles(units []*Unit) {
	var off num.Q
	for _, v := range units {
		if v.Offline && v.Position != nil && (off == nil || num.Lt(v.Position, off)) {
			off = v.Position
		}
	}
	devs, by := byDevice(units)
	for _, d := range devs {
		lst := append([]*Unit(nil), by[d]...)
		sortByOrder(lst)
		var order []int
		for i := len(lst) - 1; i >= 0; i-- {
			if _, hi, _ := camWindow(lst, i); hi != nil {
				order = append(order, i)
			}
		}
		for i := range lst {
			if _, hi, _ := camWindow(lst, i); hi == nil {
				order = append(order, i)
			}
		}
		for _, i := range order {
			e.placeSingle(units, lst, i, off)
		}
	}
}

func (e *Engine) placeSingle(units, lst []*Unit, i int, off num.Q) {
	u := lst[i]
	if u.Status != core.StatusUnsynced || u.Offline || u.Position == nil || !e.singleClash(units, u) {
		return
	}
	lo, hi, hasPrev := camWindow(lst, i)
	if off != nil && (hi == nil || num.Gt(hi, off)) {
		hi = off
	}
	insertLo := lo
	if !hasPrev {
		insertLo = nil
	}
	if u.RecTime == nil && hi != nil && e.insertAt(units, u, nil, hi) {
		return
	}
	if pos, ok := freeSlot(units, u, lo, hi, u.Position); ok {
		u.Position = pos
		return
	}
	e.insertAt(units, u, insertLo, hi)
}

func (e *Engine) insertAt(units []*Unit, u *Unit, lo, hi num.Q) bool {
	t, ok := e.idleInstant(units, u, lo, hi)
	if !ok {
		return false
	}
	gap := e.wholeFrames(u.Duration)
	for _, v := range units {
		if v != u && v.Position != nil && !num.Lt(v.Position, t) {
			v.Position = num.Add(v.Position, gap)
		}
	}
	u.Position = t
	e.Report.TimeInserted++
	return true
}

func (e *Engine) wholeFrames(d num.Q) num.Q {
	if e.seqFPS == nil || !num.Gt(e.seqFPS, num.I(0)) {
		return d
	}
	f := math.Ceil(num.Float(num.Mul(d, e.seqFPS)) - 1e-9)
	return num.Div(num.I(int64(f)), e.seqFPS)
}

func lone(v *Unit) bool {
	return v.Position != nil && !v.Offline &&
		(v.Status == core.StatusByTimecode || v.Status == core.StatusUnsynced)
}

func overlaps(a, b *Unit) bool {
	return num.Lt(a.Position, unitEnd(b)) && num.Lt(b.Position, unitEnd(a))
}

func (e *Engine) singleClash(units []*Unit, u *Unit) bool {
	for _, v := range units {
		if v != u && lone(v) && overlaps(u, v) {
			return true
		}
	}
	return false
}

func camWindow(lst []*Unit, i int) (num.Q, num.Q, bool) {
	lo, hi, hasPrev := num.I(0), num.Q(nil), false
	for j := i - 1; j >= 0; j-- {
		if lst[j].Position != nil {
			lo, hasPrev = unitEnd(lst[j]), true
			break
		}
	}
	for j := i + 1; j < len(lst); j++ {
		if lst[j].Position != nil {
			hi = lst[j].Position
			break
		}
	}
	return lo, hi, hasPrev
}

func freeSlot(units []*Unit, u *Unit, lo, hi, ref num.Q) (num.Q, bool) {
	var busy []*Unit
	for _, v := range units {
		if v != u && v.Position != nil && !v.Offline && (lone(v) || v.SharesTrack(u)) {
			busy = append(busy, v)
		}
	}
	cands := []num.Q{ref, lo}
	for _, v := range busy {
		cands = append(cands, unitEnd(v), num.Sub(v.Position, u.Duration))
	}
	var best num.Q
	for _, c := range cands {
		end := num.Add(c, u.Duration)
		if num.Lt(c, lo) || hi != nil && num.Gt(end, hi) {
			continue
		}
		free := true
		for _, v := range busy {
			if num.Lt(c, unitEnd(v)) && num.Lt(v.Position, end) {
				free = false
				break
			}
		}
		if free && (best == nil || num.Lt(num.Abs(num.Sub(c, ref)), num.Abs(num.Sub(best, ref)))) {
			best = c
		}
	}
	return best, best != nil
}

func (e *Engine) idleInstant(units []*Unit, u *Unit, lo, hi num.Q) (num.Q, bool) {
	idle := func(t num.Q) bool {
		for _, v := range units {
			if v != u && v.Position != nil && !v.Offline && num.Lt(v.Position, t) && num.Lt(t, unitEnd(v)) {
				return false
			}
		}
		return true
	}
	for _, t := range []num.Q{hi, lo} {
		if t != nil && idle(t) {
			return t, true
		}
	}
	return nil, false
}

type span struct{ s, e num.Q }

func mergeSpans(iv []span) []span {
	sort.Slice(iv, func(i, j int) bool { return num.Lt(iv[i].s, iv[j].s) })
	var out []span
	for _, x := range iv {
		if n := len(out); n > 0 && !num.Gt(x.s, out[n-1].e) {
			out[n-1].e = num.Max(out[n-1].e, x.e)
			continue
		}
		out = append(out, x)
	}
	return out
}
