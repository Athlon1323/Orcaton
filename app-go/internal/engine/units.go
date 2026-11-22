package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"orcaton/internal/core"
	"orcaton/internal/dsp"
	"orcaton/internal/formats"
	"orcaton/internal/media"
	"orcaton/internal/num"
)

type member struct {
	label string
	kind  core.TrackKind
	clip  *core.Clip
}

func (e *Engine) buildUnits(seq *core.Sequence) []*Unit {
	order, groups := groupClips(seq)
	var units []*Unit
	for _, k := range order {
		units = append(units, newUnit(len(units), k.kind == "media", groups[k]))
	}
	mergeTakeFolders(units)
	splitMixedFolders(units)
	return units
}

func mergeTakeFolders(units []*Unit) {
	type key struct{ lane, parent, ext, shape string }
	count := map[string]int{}
	for _, u := range units {
		count[u.Device]++
	}
	groups := map[key][]*Unit{}
	var order []key
	for _, u := range units {
		folder := strings.TrimPrefix(u.Device, u.Lane+"|")
		ext := unitExt(u)
		if count[u.Device] != 1 || ext == "" || folder == "" || folder == u.Device {
			continue
		}
		parent := formats.ParentPathOS(folder)
		if parent == folder || parent == "." || parent == "" {
			continue
		}
		k := key{u.Lane, parent, ext, takeShape(u.Name())}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], u)
	}
	for _, k := range order {
		if len(groups[k]) < 2 {
			continue
		}
		for _, u := range groups[k] {
			u.Device = k.lane + "|" + k.parent
		}
	}
}

func takeShape(name string) string {
	stem := formats.Casefold(strings.TrimSuffix(name, filepath.Ext(name)))
	if i := strings.LastIndexAny(stem, "_-"); i > 0 {
		stem = stem[:i]
	}
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return '#'
		}
		return r
	}, stem)
}

func splitMixedFolders(units []*Unit) {
	exts := map[string]map[string]bool{}
	for _, u := range units {
		if e := unitExt(u); e != "" {
			if exts[u.Device] == nil {
				exts[u.Device] = map[string]bool{}
			}
			exts[u.Device][e] = true
		}
	}
	for _, u := range units {
		if e := unitExt(u); e != "" && len(exts[u.Device]) > 1 {
			u.Device += "|" + e
		}
	}
}

func unitExt(u *Unit) string {
	if u.Clips[0].Media == nil {
		return ""
	}
	return strings.ToLower(filepath.Ext(u.Name()))
}

func PlanUnits(seq *core.Sequence) []*Unit { return (&Engine{}).buildUnits(seq) }

type unitKey struct {
	kind string
	id   string
	clip *core.Clip
}

func groupClips(seq *core.Sequence) ([]unitKey, map[unitKey][]member) {
	var order []unitKey
	groups := map[unitKey][]member{}
	for _, tc := range seq.Clips() {
		var k unitKey
		if tc.Clip.Media != nil {
			k = unitKey{"media", tc.Clip.Media.ID, nil}
		} else {
			k = unitKey{"clip", "", tc.Clip}
		}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], member{tc.Track.Label(), tc.Track.Kind, tc.Clip})
	}
	return order, groups
}

func newUnit(uid int, isMedia bool, members []member) *Unit {
	clips := make([]*core.Clip, len(members))
	var audioLabels, allLabels []string
	trackSet := map[string]bool{}
	var main *core.Clip
	for i, m := range members {
		clips[i] = m.clip
		allLabels = append(allLabels, m.label)
		trackSet[m.label] = true
		if m.kind == core.Audio {
			audioLabels = append(audioLabels, m.label)
			if main == nil {
				main = m.clip
			}
		}
	}
	if main == nil {
		main = clips[0]
	}
	sort.Strings(audioLabels)
	sort.Strings(allLabels)
	lane := allLabels[0]
	if len(audioLabels) > 0 {
		lane = audioLabels[0]
	}
	m := main.Media
	dur, srcIn, anchor := unitSpan(isMedia, main, clips)
	tc := math.Inf(1)
	var tcp *float64
	if m != nil && m.TCStart != nil {
		tc = num.Float(m.TCStart)
		v := tc
		tcp = &v
	}
	folder := ""
	name := main.Name
	if m != nil {
		folder = formats.ParentPathOS(m.PathOr())
		name = m.Name
	}
	var tracks []string
	for t := range trackSet {
		tracks = append(tracks, t)
	}
	sort.Strings(tracks)
	u := &Unit{
		UID: uid, Lane: lane, Clips: clips, Tracks: tracks, trackSet: trackSet,
		SourceIn: srcIn, Duration: dur, Anchor: anchor, Start0: inputStart(isMedia, clips, anchor),
		Order:    OrderKey{T: tc, Name: formats.Casefold(name)},
		TC:       tcp,
		HasAudio: m != nil && m.HasAudio,
		Device:   lane + "|" + folder,
		Status:   core.StatusUnknown, Component: -1,
		dur: num.Float(dur),
	}
	if m != nil && m.Path != nil {
		p := *m.Path
		u.MediaPath = &p
	}
	return u
}

func unitSpan(isMedia bool, main *core.Clip, clips []*core.Clip) (dur, srcIn, anchor num.Q) {
	m := main.Media
	if !isMedia || m == nil {
		return main.Duration, main.SourceIn, main.Start
	}
	if m.Duration != nil && !num.IsZero(m.Duration) {
		dur = m.Duration
	} else {
		dur = clips[0].SourceOut()
		for _, c := range clips[1:] {
			dur = num.Max(dur, c.SourceOut())
		}
	}
	return dur, num.I(0), num.I(0)
}

func inputStart(isMedia bool, clips []*core.Clip, anchor num.Q) num.Q {
	var start num.Q
	for _, c := range clips {
		if !isMedia || c.Media == nil {
			continue
		}
		if s := num.Sub(c.Start, c.SourceIn); start == nil || num.Lt(s, start) {
			start = s
		}
	}
	if start == nil {
		return anchor
	}
	return start
}

func byDevice(units []*Unit) ([]string, map[string][]*Unit) {
	var devs []string
	m := map[string][]*Unit{}
	for _, u := range units {
		if _, ok := m[u.Device]; !ok {
			devs = append(devs, u.Device)
		}
		m[u.Device] = append(m[u.Device], u)
	}
	return devs, m
}

func sortByOrder(lst []*Unit) {
	sort.SliceStable(lst, func(i, j int) bool { return lst[i].Order.Less(lst[j].Order) })
}

func (e *Engine) loadRecordTimes(ctx context.Context, units []*Unit) error {
	if !e.O.RecordTime {
		return nil
	}
	src := e.Times
	if src == nil {
		var err error
		if src, err = media.NewRecordTimes(nil, nil, true); err != nil {
			return err
		}
	}
	var todo []*Unit
	for _, u := range units {
		if u.Clips[0].Media != nil && u.MediaPath != nil && *u.MediaPath != "" {
			todo = append(todo, u)
		}
	}
	e.emit(StageRecordTime, 0, len(todo), "")
	var mu sync.Mutex
	done := 0
	parallelFor(len(todo), e.probeWorkers(), func(i int) {
		if ctx.Err() != nil {
			return
		}
		u := todo[i]
		if rt := src.Get(u.Clips[0].Media.Name, *u.MediaPath); rt != nil {
			v := rt.Start
			u.RecTime = &v
			u.recAlt = rt.Alt
		}
		mu.Lock()
		done++
		e.emit(StageRecordTime, done, len(todo), u.Name())
		mu.Unlock()
	})
	if err := ctx.Err(); err != nil {
		return err
	}
	resolveNameDates(units)
	n := 0
	for _, u := range units {
		if u.RecTime != nil {
			n++
		}
	}
	e.Report.WithRecTime = n
	devs, by := byDevice(units)
	for _, d := range devs {
		all := true
		for _, u := range by[d] {
			if u.RecTime == nil {
				all = false
				break
			}
		}
		if all {
			for _, u := range by[d] {
				u.Order.T = *u.RecTime
			}
		}
	}
	return nil
}

func (e *Engine) loadFeatures(ctx context.Context, units []*Unit) error {
	prm := media.FeatureParams{SampleRate: e.O.SampleRate, Hop: e.O.Hop, AudioMode: e.O.AudioMode}
	src := e.Features
	if src == nil {
		st, err := media.NewFeatureStore(e.O.CacheDir, &prm)
		if err != nil {
			return err
		}
		src = st
	}
	fps := src.Params().FPS()
	var paths []string
	for _, u := range units {
		m := u.Clips[0].Media
		switch {
		case m != nil && m.Offline:
			u.setError(ReasonOffline, "the file is not on disk")
			u.Offline = true
		case !u.HasAudio:
			u.setError(ReasonNoAudio, "no audio")
		case u.MediaPath == nil || *u.MediaPath == "":
			u.setError(ReasonNoPath, "no file path")
			u.Offline = true
		case !src.Has(*u.MediaPath):
			u.setError(ReasonNotFound, "file not found (check --map)")
			u.Offline = true
		default:
			paths = append(paths, *u.MediaPath)
		}
	}
	total := len(uniq(paths))
	done := 0
	e.emit(StageFeatures, 0, total, "")
	errs := src.Prefetch(ctx, paths, e.jobs(), func(p string, _ error) {
		done++
		e.emit(StageFeatures, done, total, formats.BaseName(p))
	})
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, p := range uniq(paths) {
		var nf *media.FFmpegNotFound
		if errors.As(errs[p], &nf) {
			return nf
		}
	}
	for _, u := range units {
		if u.Error != nil || u.MediaPath == nil || *u.MediaPath == "" {
			continue
		}
		if err, ok := errs[*u.MediaPath]; ok {
			u.setError(ReasonDecodeError, fmt.Sprintf("decode error: %v", err))
			continue
		}
		raw, err := src.Get(*u.MediaPath)
		if err != nil {
			u.setError(ReasonDecodeError, fmt.Sprintf("decode error: %v", err))
			continue
		}
		full := dsp.MaskOnsetArtifacts(raw, fps, 0.15, 0.05)
		i0 := int(math.RoundToEven(num.Float(u.SourceIn) * fps))
		if i0 < 0 {
			i0 = 0
		}
		i1 := i0 + int(math.RoundToEven(num.Float(u.Duration)*fps))
		if i1 > len(full) {
			i1 = len(full)
		}
		u.Feature = nil
		if i1 > i0 {
			u.Feature = full[i0:i1]
		}
		if u.Feature == nil || float64(len(u.Feature)) < e.O.MinOverlapSec*fps {
			u.setError(ReasonShortAudio, "audio is too short")
			u.Feature = nil
		}
	}
	return nil
}

func uniq(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func (e *Engine) jobs() int {
	if e.res.Jobs > 0 {
		return e.res.Jobs
	}
	return max(e.O.Jobs, 1)
}

func (e *Engine) probeWorkers() int { return min(max(2*e.jobs(), 8), 16) }
