package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

type interval struct{ s, e num.Q }

func sortIntervals(iv []interval) {
	sort.SliceStable(iv, func(a, b int) bool {
		if c := iv[a].s.Cmp(iv[b].s); c != 0 {
			return c < 0
		}
		return iv[a].e.Cmp(iv[b].e) < 0
	})
}

func firstFit(start, dur num.Q, busyLists [][]interval) num.Q {
	var busy []interval
	for _, l := range busyLists {
		busy = append(busy, l...)
	}
	sortIntervals(busy)
	pos := start
	moved := true
	for moved {
		moved = false
		for _, b := range busy {
			if num.Lt(b.s, num.Add(pos, dur)) && num.Lt(pos, b.e) {
				pos = b.e
				moved = true
			}
		}
	}
	return pos
}

func (e *Engine) placeOffline(units []*Unit, offline []*Unit) {
	if len(offline) == 0 {
		return
	}
	off := map[int]bool{}
	for _, u := range offline {
		off[u.UID] = true
	}
	var g num.Q
	for _, u := range units {
		if u.Position != nil && !off[u.UID] {
			if v := num.Add(u.Position, u.Duration); g == nil || num.Gt(v, g) {
				g = v
			}
		}
	}
	if g == nil {
		g = num.I(0)
	}
	ends := map[string]num.Q{}
	endOf := func(tr string) num.Q {
		if v, ok := ends[tr]; ok {
			return v
		}
		return g
	}
	lst := append([]*Unit(nil), offline...)
	sort.SliceStable(lst, func(a, b int) bool {
		if lst[a].Device != lst[b].Device {
			return lst[a].Device < lst[b].Device
		}
		return lst[a].Order.Less(lst[b].Order)
	})
	for _, u := range lst {
		var pos num.Q
		for _, tr := range u.Tracks {
			if pos == nil {
				pos = endOf(tr)
			} else {
				pos = num.Max(pos, endOf(tr))
			}
		}
		u.Position = pos
		for _, tr := range u.Tracks {
			ends[tr] = num.Add(pos, u.Duration)
		}
	}
}

func (e *Engine) fixSharedTracks(units []*Unit) {
	occupied := map[string][]interval{}
	anchored := map[string]bool{}
	for _, u := range units {
		if u.placed() && u.Position != nil {
			anchored[u.Device] = true
			for _, tr := range u.Tracks {
				occupied[tr] = append(occupied[tr], interval{u.Position, num.Add(u.Position, u.Duration)})
			}
		}
	}
	var lst []*Unit
	for _, u := range units {
		if !u.placed() && u.Position != nil {
			lst = append(lst, u)
		}
	}
	sort.SliceStable(lst, func(a, b int) bool {
		if x, y := anchored[lst[a].Device], anchored[lst[b].Device]; x != y {
			return x
		}
		if c := lst[a].Position.Cmp(lst[b].Position); c != 0 {
			return c < 0
		}
		if lst[a].Device == lst[b].Device {
			return lst[a].Order.Less(lst[b].Order)
		}
		return lst[a].Order.Before(lst[b].Order)
	})
	devEnd := map[string]num.Q{}
	for _, u := range lst {
		var busy [][]interval
		for _, t := range u.Tracks {
			busy = append(busy, occupied[t])
		}
		start := u.Position
		if v, ok := devEnd[u.Device]; ok {
			start = num.Max(start, v)
		}
		pos := firstFit(start, u.Duration, busy)
		u.Position = pos
		devEnd[u.Device] = num.Add(pos, u.Duration)
		for _, tr := range u.Tracks {
			occupied[tr] = append(occupied[tr], interval{pos, num.Add(pos, u.Duration)})
		}
	}
}

func (e *Engine) placeBlocksPlain(units []*Unit, comps [][]int) {
	c := num.I(0)
	for _, comp := range comps {
		var block []*Unit
		for _, i := range comp {
			if units[i].placed() {
				block = append(block, units[i])
			}
		}
		if len(block) == 0 {
			continue
		}
		var base num.Q
		for _, u := range block {
			if u.Position != nil && (base == nil || num.Lt(u.Position, base)) {
				base = u.Position
			}
		}
		for _, u := range block {
			u.Position = num.Add(num.Sub(u.Position, base), c)
		}
		var mx num.Q
		for _, u := range block {
			if u.Position != nil {
				v := num.Add(u.Position, u.Duration)
				if mx == nil || num.Gt(v, mx) {
					mx = v
				}
			}
		}
		c = num.Add(mx, num.FromFloat(e.O.GroupGapSec))
	}
}

func (e *Engine) placeRestAtEnd(units []*Unit) {
	occupied := map[string][]interval{}
	var endAll num.Q
	for _, u := range units {
		if u.Position != nil {
			for _, tr := range u.Tracks {
				iv := interval{u.Position, num.Add(u.Position, u.Duration)}
				occupied[tr] = append(occupied[tr], iv)
				if endAll == nil || num.Gt(iv.e, endAll) {
					endAll = iv.e
				}
			}
		}
	}
	if endAll == nil {
		endAll = num.I(0)
	}
	curs := map[string]num.Q{}
	var lst []*Unit
	for _, u := range units {
		if u.Position == nil {
			lst = append(lst, u)
		}
	}
	sort.SliceStable(lst, func(a, b int) bool {
		if lst[a].Device != lst[b].Device {
			return lst[a].Device < lst[b].Device
		}
		return lst[a].Order.Less(lst[b].Order)
	})
	for _, u := range lst {
		start, ok := curs[u.Device]
		if !ok {
			start = endAll
		}
		var busy [][]interval
		for _, t := range u.Tracks {
			busy = append(busy, occupied[t])
		}
		pos := firstFit(start, u.Duration, busy)
		u.Position = pos
		for _, tr := range u.Tracks {
			occupied[tr] = append(occupied[tr], interval{pos, num.Add(pos, u.Duration)})
		}
		curs[u.Device] = num.Add(pos, u.Duration)
	}
}

func (e *Engine) compressIdle(units []*Unit) { e.compressIdleBy(units, false) }

func (e *Engine) compressIdleBy(units []*Unit, wholeFrames bool) {
	var iv []interval
	for _, u := range units {
		if u.Position != nil {
			iv = append(iv, interval{u.Position, num.Add(u.Position, u.Duration)})
		}
	}
	if len(iv) == 0 {
		return
	}
	sortIntervals(iv)
	maxGap := num.FloatLimit(e.O.IdleGapSec, 1000)
	keep := num.FloatLimit(e.O.IdleKeepSec, 1000)
	type cut struct{ at, d num.Q }
	var cuts []cut
	end := iv[0].e
	for _, x := range iv[1:] {
		g := num.Sub(x.s, end)
		if num.Gt(g, maxGap) && num.Gt(g, keep) {
			d := num.Sub(g, keep)
			if wholeFrames && e.seqFPS != nil && num.Gt(e.seqFPS, num.I(0)) {
				d = num.Div(num.I(int64(math.Floor(num.Float(num.Mul(d, e.seqFPS))+1e-9))), e.seqFPS)
			}
			if num.Gt(d, num.I(0)) {
				cuts = append(cuts, cut{x.s, d})
			}
		}
		end = num.Max(end, x.e)
	}
	if len(cuts) == 0 {
		return
	}
	for _, u := range units {
		if u.Position == nil {
			continue
		}
		s := num.I(0)
		for _, c := range cuts {
			if num.Ge(u.Position, c.at) {
				s = num.Add(s, c.d)
			}
		}
		u.Position = num.Sub(u.Position, s)
	}
	e.Report.GapsCompressed += len(cuts)
}

func (e *Engine) apply(seq *core.Sequence, units []*Unit) {
	removed := map[*core.Clip]bool{}
	for _, u := range units {
		if u.Position == nil {
			for _, c := range u.Clips {
				removed[c] = true
			}
			continue
		}
		for _, c := range u.Clips {
			if c.Media != nil {
				c.Start = num.Add(u.Position, c.SourceIn)
			} else {
				c.Start = num.Add(u.Position, num.Sub(c.Start, u.Anchor))
			}
			c.SyncStatus = u.Status
			switch {
			case u.Status == core.StatusUnsynced:
				l := e.O.UnsyncedLabel
				c.Label = &l
			case u.Status == core.StatusByTimecode:
				l := e.O.TimeLabel
				c.Label = &l
			case c.Label != nil && (*c.Label == e.O.UnsyncedLabel || *c.Label == e.O.TimeLabel):
				c.Label = nil
			}
		}
	}
	for _, t := range seq.Tracks {
		var keep []*core.Clip
		for _, c := range t.Clips {
			if !removed[c] {
				keep = append(keep, c)
			}
		}
		sort.SliceStable(keep, func(a, b int) bool { return num.Lt(keep[a].Start, keep[b].Start) })
		t.Clips = keep
	}
	var base num.Q
	for _, t := range seq.Tracks {
		for _, c := range t.Clips {
			if base == nil || num.Lt(c.Start, base) {
				base = c.Start
			}
		}
	}
	if base != nil && !num.IsZero(base) {
		for _, t := range seq.Tracks {
			for _, c := range t.Clips {
				c.Start = num.Sub(c.Start, base)
			}
		}
	}
}
