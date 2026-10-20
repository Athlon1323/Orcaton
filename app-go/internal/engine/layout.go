package engine

import (
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

func (e *Engine) layout(units []*Unit, comps [][]int) {
	offline := markUnplaced(units)
	e.Report.Offline = len(offline)
	switch e.O.Unsynced {
	case PlaceRemove:
		e.placeBlocksPlain(units, comps)
		return
	case PlaceEnd:
		e.placeBlocksPlain(units, comps)
		var rest []*Unit
		for _, u := range units {
			if !u.Offline {
				rest = append(rest, u)
			}
		}
		e.placeRestAtEnd(rest)
		e.placeOffline(units, offline)
		return
	}
	st := e.newLayoutState(units, offline)
	for _, comp := range comps {
		st.placeBlock(units, comp)
	}
	st.placeRest(units)
	e.fixSharedTracks(units)
	e.placeOffline(units, offline)
	e.compressIdle(units)
	if e.O.Singles == SinglesSequential {
		e.serializeSingles(units)
		e.compressIdleBy(units, true)
	}
}

func markUnplaced(units []*Unit) []*Unit {
	for _, u := range units {
		if !u.placed() {
			u.Status = core.StatusUnsynced
			if u.Error == nil {
				u.setError(ReasonNoMatch, "нет надёжного совпадения по звуку")
			}
			u.Position = nil
		}
	}
	var offline []*Unit
	for _, u := range units {
		if u.Offline && !u.placed() {
			offline = append(offline, u)
		}
	}
	return offline
}

type layoutState struct {
	e        *Engine
	edges    bool
	devs     []string
	byDev    map[string][]*Unit
	idxInDev map[int]int
	placed   map[int]bool
	cursor   map[string]num.Q
	last     map[string]*Unit
	prevEnd  num.Q
	gap      num.Q
	maxGap   num.Q
}

func (e *Engine) newLayoutState(units, offline []*Unit) *layoutState {
	st := &layoutState{e: e, edges: e.O.Unsynced == PlaceEdges, idxInDev: map[int]int{},
		placed: map[int]bool{}, cursor: map[string]num.Q{}, last: map[string]*Unit{},
		gap: num.FromFloat(e.O.GroupGapSec), maxGap: num.FloatLimit(e.O.MaxTimeGapSec, 1000)}
	st.devs, st.byDev = byDevice(units)
	for _, d := range st.devs {
		sortByOrder(st.byDev[d])
		for i, u := range st.byDev[d] {
			st.idxInDev[u.UID] = i
		}
	}
	for _, u := range offline {
		st.placed[u.UID] = true
	}
	return st
}

func (st *layoutState) cur(d string) num.Q {
	if v, ok := st.cursor[d]; ok {
		return v
	}
	return num.I(0)
}

func unitEnd(u *Unit) num.Q { return num.Add(qOr0(u.Position), u.Duration) }

func (st *layoutState) byClock(w, v *Unit, floor, ceil num.Q) num.Q {
	if w == nil || w.Position == nil || w.RecTime == nil || v.RecTime == nil || !st.e.O.RecordTime {
		return floor
	}
	t := num.Add(w.Position, num.FloatLimit(*v.RecTime-*w.RecTime, 1000))
	if num.Lt(t, floor) {
		return floor
	}
	if num.Gt(num.Sub(t, floor), st.maxGap) {
		t = num.Add(floor, st.gap)
	}
	if ceil != nil && num.Gt(num.Add(t, v.Duration), ceil) {
		return floor
	}
	return t
}

func (st *layoutState) placeSeq(dev string, upto int) {
	for _, v := range st.byDev[dev][:upto] {
		if st.placed[v.UID] || v.placed() {
			continue
		}
		v.Position = st.byClock(st.last[dev], v, st.cur(dev), nil)
		st.cursor[dev] = num.Add(v.Position, v.Duration)
		st.last[dev] = v
		st.placed[v.UID] = true
	}
}

func (st *layoutState) startOf(dev string, f *Unit) (num.Q, num.Q) {
	c, w := st.cur(dev), st.last[dev]
	for _, v := range st.byDev[dev][:st.idxInDev[f.UID]] {
		if st.placed[v.UID] || v.placed() {
			continue
		}
		pos := st.byClock(w, v, c, nil)
		tmp := *v
		tmp.Position = pos
		c, w = num.Add(pos, v.Duration), &tmp
	}
	return st.byClock(w, f, c, nil), c
}

func (st *layoutState) placeBlock(units []*Unit, comp []int) {
	var block []*Unit
	for _, i := range comp {
		if units[i].placed() {
			block = append(block, units[i])
		}
	}
	if len(block) == 0 {
		return
	}
	var base num.Q
	for _, u := range block {
		if u.Position != nil && (base == nil || num.Lt(u.Position, base)) {
			base = u.Position
		}
	}
	rel := map[int]num.Q{}
	for _, u := range block {
		rel[u.UID] = num.Sub(qOr0(u.Position), base)
	}
	firstOrder, first := firstPerDevice(block)
	if st.edges && st.prevEnd != nil {
		for _, d := range firstOrder {
			st.cursor[d] = num.Max(st.cur(d), num.Add(st.prevEnd, st.gap))
		}
	}
	need := num.I(0)
	if st.prevEnd != nil {
		need = num.Add(st.prevEnd, st.gap)
	}
	for _, d := range firstOrder {
		f := first[d]
		fPos, fillersEnd := st.startOf(d, f)
		need = num.Max(need, num.Sub(fPos, rel[f.UID]))
		if st.edges {
			need = num.Max(need, fillersEnd)
		}
	}
	for _, d := range firstOrder {
		st.placeSeq(d, st.idxInDev[first[d].UID])
	}
	for _, u := range block {
		u.Position = num.Add(need, rel[u.UID])
		st.placed[u.UID] = true
	}
	for _, d := range firstOrder {
		st.fillBlock(d, block)
	}
	pe := unitEnd(block[0])
	for _, u := range block[1:] {
		pe = num.Max(pe, unitEnd(u))
	}
	st.prevEnd = pe
}

func firstPerDevice(block []*Unit) ([]string, map[string]*Unit) {
	var order []string
	first := map[string]*Unit{}
	for _, u := range block {
		f, ok := first[u.Device]
		if !ok {
			order = append(order, u.Device)
			first[u.Device] = u
		} else if u.Order.Less(f.Order) {
			first[u.Device] = u
		}
	}
	return order, first
}

func (st *layoutState) fillBlock(d string, block []*Unit) {
	var members []*Unit
	for _, u := range block {
		if u.Device == d {
			members = append(members, u)
		}
	}
	sortByOrder(members)
	maxEnd := unitEnd(members[0])
	for _, u := range members[1:] {
		maxEnd = num.Max(maxEnd, unitEnd(u))
	}
	if st.edges {
		lst := members[0]
		for _, v := range members[1:] {
			if num.Gt(unitEnd(v), unitEnd(lst)) {
				lst = v
			}
		}
		st.last[d] = lst
		st.cursor[d] = num.Max(st.cur(d), maxEnd)
		return
	}
	byDev := st.byDev[d]
	lo, hi := st.idxInDev[members[0].UID], st.idxInDev[members[len(members)-1].UID]
	c := num.I(0)
	var prev *Unit
	span := byDev[lo : hi+1]
	for _, v := range span {
		if v.placed() && v.Position != nil {
			c = num.Add(v.Position, v.Duration)
			prev = v
		} else if !st.placed[v.UID] {
			var nxt num.Q
			for _, w := range byDev[st.idxInDev[v.UID]:] {
				if w.placed() && w.Position != nil {
					nxt = w.Position
					break
				}
			}
			if nxt != nil && num.Gt(num.Add(c, v.Duration), nxt) {
				continue
			}
			v.Position = st.byClock(prev, v, c, nxt)
			c = num.Add(v.Position, v.Duration)
			prev = v
			st.placed[v.UID] = true
		}
	}
	var lst *Unit
	for _, v := range span {
		if st.placed[v.UID] && (lst == nil || num.Gt(unitEnd(v), unitEnd(lst))) {
			lst = v
		}
	}
	st.last[d] = lst
	st.cursor[d] = num.Max(st.cur(d), maxEnd)
}

func (st *layoutState) placeRest(units []*Unit) {
	trackEnd := map[string]num.Q{}
	te := func(tr string) num.Q {
		if v, ok := trackEnd[tr]; ok {
			return v
		}
		return num.I(0)
	}
	var tEnd num.Q
	if st.edges {
		for _, u := range units {
			if st.placed[u.UID] && u.Position != nil {
				if tEnd == nil {
					tEnd = unitEnd(u)
				} else {
					tEnd = num.Max(tEnd, unitEnd(u))
				}
				for _, tr := range u.Tracks {
					trackEnd[tr] = num.Max(te(tr), num.Add(u.Position, u.Duration))
				}
			}
		}
	}
	if tEnd == nil {
		tEnd = num.I(0)
	}
	var rest []string
	restKey := map[string]OrderKey{}
	for _, d := range st.devs {
		var k *OrderKey
		for _, v := range st.byDev[d] {
			if !st.placed[v.UID] && (k == nil || v.Order.Less(*k)) {
				o := v.Order
				k = &o
			}
		}
		if k != nil {
			rest = append(rest, d)
			restKey[d] = *k
		}
	}
	sort.SliceStable(rest, func(a, b int) bool { return restKey[rest[a]].Before(restKey[rest[b]]) })
	for _, d := range rest {
		lst := st.byDev[d]
		if st.edges {
			c := num.Max(st.cur(d), tEnd)
			seen := map[string]bool{}
			for _, v := range lst {
				for _, tr := range v.Tracks {
					if !seen[tr] {
						seen[tr] = true
						c = num.Max(c, te(tr))
					}
				}
			}
			st.cursor[d] = c
		}
		for _, v := range lst {
			if st.placed[v.UID] {
				continue
			}
			v.Position = st.byClock(st.last[d], v, st.cur(d), nil)
			st.cursor[d] = num.Add(v.Position, v.Duration)
			st.last[d] = v
			st.placed[v.UID] = true
			for _, tr := range v.Tracks {
				trackEnd[tr] = num.Max(te(tr), st.cursor[d])
			}
		}
	}
}
