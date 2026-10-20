package xmeml

import (
	"os"
	"path/filepath"
	"strconv"

	"orcaton/internal/core"
	"orcaton/internal/formats"
	"orcaton/internal/num"
	"orcaton/internal/xmldom"
)

type Exporter struct{}

func (Exporter) Name() string         { return "xmeml" }
func (Exporter) Title() string        { return "FCP7 XML (xmeml v5)" }
func (Exporter) Extensions() []string { return []string{".xml"} }

func frames(sec num.Q, r core.Rate) int64 { return num.RoundInt(r.SecondsToFrames(sec)) }

func i64(n int64) string { return strconv.FormatInt(n, 10) }

func rateEl(parent *xmldom.W, r core.Rate) {
	re := parent.Sub("rate")
	re.SubText("timebase", i64(r.Timebase()))
	re.SubText("ntsc", boolS(r.IsNTSC()))
}

func boolS(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}

type clipRef struct {
	id     string
	kind   core.TrackKind
	tindex int
	cindex int
}

func (Exporter) Write(p *core.Project, path string) error {
	root := xmldom.NewW("xmeml", xmldom.Attr{Name: "version", Value: "5"})
	for i, seq := range p.Sequences {
		writeSequence(root, seq, i+1)
	}
	data := xmldom.Serialize(root, "<!DOCTYPE xmeml>")
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func writeSequence(root *xmldom.W, seq *core.Sequence, n int) {
	rate := seq.Rate
	s := root.Sub("sequence", xmldom.Attr{Name: "id", Value: "sequence-" + strconv.Itoa(n)})
	s.SubText("name", seq.Name)
	s.SubText("duration", i64(frames(seq.Duration(), rate)))
	rateEl(s, rate)
	s.SubText("in", "-1")
	s.SubText("out", "-1")
	tc := s.Sub("timecode")
	tcFrames := frames(seq.TCStart, rate)
	tc.SubText("string", core.FramesToTC(tcFrames, rate, false))
	tc.SubText("frame", i64(tcFrames))
	tc.SubText("displayformat", "NDF")
	rateEl(tc, rate)

	media := s.Sub("media")
	video, audio := media.Sub("video"), media.Sub("audio")
	fmtEl := video.Sub("format").Sub("samplecharacteristics")
	rateEl(fmtEl, rate)
	w, h := 1920, 1080
	if seq.Width != nil && *seq.Width != 0 {
		w = *seq.Width
	}
	if seq.Height != nil && *seq.Height != 0 {
		h = *seq.Height
	}
	fmtEl.SubText("width", strconv.Itoa(w))
	fmtEl.SubText("height", strconv.Itoa(h))
	fmtEl.SubText("pixelaspectratio", "square")

	ids := map[*core.Clip]clipRef{}
	groups := map[string][]*core.Clip{}
	for _, t := range seq.Tracks {
		for ci, c := range t.SortedClips() {
			ids[c] = clipRef{"clipitem-" + t.Label() + "-" + strconv.Itoa(ci+1), t.Kind, t.Index, ci + 1}
			if c.LinkGroup != nil && *c.LinkGroup != "" {
				groups[*c.LinkGroup] = append(groups[*c.LinkGroup], c)
			}
		}
	}
	written := map[string]bool{}
	for _, t := range seq.Tracks {
		parent := audio
		if t.Kind == core.Video {
			parent = video
		}
		tr := parent.Sub("track")
		for _, c := range t.SortedClips() {
			writeClip(tr, c, t, rate, ids, groups, written)
		}
		tr.SubText("enabled", boolS(t.Enabled))
		tr.SubText("locked", boolS(t.Locked))
	}
}

func writeClip(tr *xmldom.W, c *core.Clip, t *core.Track, rate core.Rate,
	ids map[*core.Clip]clipRef, groups map[string][]*core.Clip, written map[string]bool) {
	ci := tr.Sub("clipitem", xmldom.Attr{Name: "id", Value: ids[c].id})
	if c.Media != nil {
		ci.SubText("masterclipid", "masterclip-"+c.Media.ID)
	}
	ci.SubText("name", c.Name)
	ci.SubText("enabled", boolS(c.Enabled))
	mediaDur := c.SourceOut()
	if c.Media != nil && c.Media.Duration != nil && !num.IsZero(c.Media.Duration) {
		mediaDur = c.Media.Duration
	}
	ci.SubText("duration", i64(frames(mediaDur, rate)))
	rateEl(ci, rate)
	start := frames(c.Start, rate)
	dur := frames(c.Duration, rate)
	srcIn := frames(c.SourceIn, rate)
	ci.SubText("start", i64(start))
	ci.SubText("end", i64(start+dur))
	ci.SubText("in", i64(srcIn))
	ci.SubText("out", i64(srcIn+dur))
	if c.Media != nil {
		writeFile(ci, c.Media, written)
	}
	if t.Kind == core.Audio {
		st := ci.Sub("sourcetrack")
		st.SubText("mediatype", "audio")
		idx := 1
		if c.SourceTrack != nil && *c.SourceTrack != 0 {
			idx = *c.SourceTrack
		}
		st.SubText("trackindex", strconv.Itoa(idx))
	}
	if c.LinkGroup != nil && *c.LinkGroup != "" && len(groups[*c.LinkGroup]) > 1 {
		for _, member := range groups[*c.LinkGroup] {
			r := ids[member]
			lk := ci.Sub("link")
			lk.SubText("linkclipref", r.id)
			lk.SubText("mediatype", string(r.kind))
			lk.SubText("trackindex", strconv.Itoa(r.tindex))
			lk.SubText("clipindex", strconv.Itoa(r.cindex))
		}
	}
	if c.Label != nil && *c.Label != "" {
		ci.Sub("labels").SubText("label2", *c.Label)
	}
}

func writeFile(ci *xmldom.W, m *core.MediaRef, written map[string]bool) {
	f := ci.Sub("file", xmldom.Attr{Name: "id", Value: m.ID})
	if written[m.ID] {
		return
	}
	written[m.ID] = true
	frate := core.RateInt(25)
	if m.Rate != nil {
		frate = *m.Rate
	}
	if m.Offline {
		name := m.FileName
		if name == "" {
			name = "Slug"
		}
		f.SubText("name", name)
		rateEl(f, frate)
		if m.Duration != nil {
			f.SubText("duration", i64(frames(m.Duration, frate)))
		}
		f.SubText("mediaSource", "Slug")
		sc := f.Sub("media").Sub("video").Sub("samplecharacteristics")
		if m.Width != nil && *m.Width != 0 {
			sc.SubText("width", strconv.Itoa(*m.Width))
		}
		if m.Height != nil && *m.Height != 0 {
			sc.SubText("height", strconv.Itoa(*m.Height))
		}
		return
	}
	f.SubText("name", m.Name)
	var url string
	switch {
	case m.Path != nil && *m.Path != "" && m.OriginalPath != nil:
		url = formats.PathToURL(*m.Path)
	case m.URL != nil && *m.URL != "":
		url = *m.URL
	case m.Path != nil && *m.Path != "":
		url = formats.PathToURL(*m.Path)
	}
	if url != "" {
		f.SubText("pathurl", url)
	}
	rateEl(f, frate)
	if m.Duration != nil {
		f.SubText("duration", i64(frames(m.Duration, frate)))
	}
	if m.TCStart != nil {
		tc := f.Sub("timecode")
		fr := frames(m.TCStart, frate)
		rateEl(tc, frate)
		s := core.FramesToTC(fr, frate, false)
		if m.TCString != nil && *m.TCString != "" {
			s = *m.TCString
		}
		tc.SubText("string", s)
		tc.SubText("frame", i64(fr))
		tc.SubText("displayformat", "NDF")
	}
	media := f.Sub("media")
	if m.HasVideo {
		sc := media.Sub("video").Sub("samplecharacteristics")
		rateEl(sc, frate)
		if m.Width != nil && *m.Width != 0 {
			sc.SubText("width", strconv.Itoa(*m.Width))
		}
		if m.Height != nil && *m.Height != 0 {
			sc.SubText("height", strconv.Itoa(*m.Height))
		}
	}
	if m.HasAudio {
		a := media.Sub("audio")
		sc := a.Sub("samplecharacteristics")
		sc.SubText("depth", "16")
		sr := 48000
		if m.SampleRate != nil && *m.SampleRate != 0 {
			sr = *m.SampleRate
		}
		sc.SubText("samplerate", strconv.Itoa(sr))
		ch := 2
		if m.AudioChannels != nil && *m.AudioChannels != 0 {
			ch = *m.AudioChannels
		}
		a.SubText("channelcount", strconv.Itoa(ch))
	}
}
