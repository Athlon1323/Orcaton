package engine

import (
	"crypto/sha1"
	"encoding/binary"
	"math"
	"os"
	"sort"
	"strconv"

	"orcaton/internal/core"
	"orcaton/internal/num"
)

func dupKey(u *Unit) string {
	if u.Feature == nil || u.Offline {
		return ""
	}
	var lo, hi float32 = math.MaxFloat32, -math.MaxFloat32
	h := sha1.New()
	var b [4]byte
	for _, v := range u.Feature {
		lo, hi = min(lo, v), max(hi, v)
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(v))
		h.Write(b[:])
	}
	if !(hi > lo) {
		return ""
	}
	return string(h.Sum(nil)) + "|" + num.String(u.Duration) + "|" + num.String(u.SourceIn) +
		"|" + strconv.Itoa(len(u.Feature))
}

func fileSize(u *Unit) int64 {
	if u.MediaPath == nil {
		return -1
	}
	st, err := os.Stat(*u.MediaPath)
	if err != nil {
		return -1
	}
	return st.Size()
}

func markDuplicates(units []*Unit) int {
	groups := map[string][]*Unit{}
	var keys []string
	for _, u := range units {
		k := dupKey(u)
		if k == "" {
			continue
		}
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], u)
	}
	n := 0
	for _, k := range keys {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		sort.SliceStable(g, func(i, j int) bool {
			if a, b := len([]rune(g[i].Name())), len([]rune(g[j].Name())); a != b {
				return a < b
			}
			return g[i].UID < g[j].UID
		})
		orig := g[0]
		size := fileSize(orig)
		for _, u := range g[1:] {
			if s := fileSize(u); s >= 0 && size >= 0 && s != size {
				continue
			}
			u.DupOf = orig
			u.Feature = nil
			n++
		}
	}
	return n
}

func splitDuplicates(units []*Unit) (main, dups []*Unit) {
	for _, u := range units {
		if u.DupOf != nil && !u.inLane {
			dups = append(dups, u)
			continue
		}
		u.UID = len(main)
		main = append(main, u)
	}
	return main, dups
}

func attachDuplicates(main, dups []*Unit) []*Unit {
	all := append([]*Unit(nil), main...)
	for _, u := range dups {
		o := u.DupOf
		u.UID = len(all)
		u.Position, u.Status, u.Component, u.BestScore = o.Position, o.Status, o.Component, o.BestScore
		u.Error, u.Reason = o.Error, o.Reason
		all = append(all, u)
	}
	return all
}

func keepDuplicatesInLane(units []*Unit) {
	for _, u := range units {
		if u.DupOf != nil && u.SharesTrack(u.DupOf) {
			u.inLane = true
			u.setError(ReasonDuplicate, dupReasonText)
		}
	}
}

const dupReasonText = "копия другого файла проекта (в DRT новые дорожки не добавляются)"

func (e *Engine) fitDuplicatesInPlace(units []*Unit) {
	busy := map[string][]interval{}
	for _, u := range units {
		if u.Position == nil || (u.DupOf != nil && !u.inLane) {
			continue
		}
		for _, t := range u.Tracks {
			busy[t] = append(busy[t], interval{u.Position, num.Add(u.Position, u.Duration)})
		}
	}
	for _, u := range units {
		if u.DupOf == nil || u.inLane || u.Position == nil {
			continue
		}
		var lists [][]interval
		for _, t := range u.Tracks {
			lists = append(lists, busy[t])
		}
		if pos := firstFit(u.Position, u.Duration, lists); !num.Eq(pos, u.Position) {
			u.Position, u.Status, u.Component = pos, core.StatusUnsynced, -1
			u.setError(ReasonDuplicate, dupReasonText)
		}
		for _, t := range u.Tracks {
			busy[t] = append(busy[t], interval{u.Position, num.Add(u.Position, u.Duration)})
		}
	}
}

func moveDuplicateClips(seq *core.Sequence, units []*Unit) int {
	trackOf := map[*core.Clip]*core.Track{}
	for _, t := range seq.Tracks {
		for _, c := range t.Clips {
			trackOf[c] = t
		}
	}
	busy := func(t *core.Track, c *core.Clip) bool {
		for _, o := range t.Clips {
			if o == c {
				continue
			}
			ov := num.Float(num.Min(o.End(), c.End())) - num.Float(num.Max(o.Start, c.Start))
			if ov > overlapTolF {
				return true
			}
		}
		return false
	}
	extra := map[*core.Track][]*core.Track{}
	nextIndex := func(kind core.TrackKind) int {
		n := 0
		for _, t := range seq.Tracks {
			if t.Kind == kind {
				n = max(n, t.Index)
			}
		}
		return n + 1
	}
	moved := 0
	for _, u := range units {
		if u.DupOf == nil || u.Position == nil {
			continue
		}
		newTracks := map[string]bool{}
		for _, c := range u.Clips {
			src := trackOf[c]
			if src == nil || !busy(src, c) {
				if src != nil {
					newTracks[src.Label()] = true
				}
				continue
			}
			var dst *core.Track
			for _, t := range extra[src] {
				if !busy(t, c) {
					dst = t
					break
				}
			}
			if dst == nil {
				dst = &core.Track{Kind: src.Kind, Index: nextIndex(src.Kind), Enabled: src.Enabled}
				extra[src] = append(extra[src], dst)
				seq.Tracks = append(seq.Tracks, dst)
			}
			removeClip(src, c)
			dst.Clips = append(dst.Clips, c)
			trackOf[c] = dst
			newTracks[dst.Label()] = true
			moved++
		}
		u.Tracks = u.Tracks[:0]
		for t := range newTracks {
			u.Tracks = append(u.Tracks, t)
		}
		sort.Strings(u.Tracks)
		u.trackSet = newTracks
	}
	return moved
}

func removeClip(t *core.Track, c *core.Clip) {
	for i, o := range t.Clips {
		if o == c {
			t.Clips = append(t.Clips[:i], t.Clips[i+1:]...)
			return
		}
	}
}
