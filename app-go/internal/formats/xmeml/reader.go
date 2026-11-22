package xmeml

import (
	"bytes"
	"math/big"
	"os"
	"sort"
	"strings"

	"orcaton/internal/core"
	"orcaton/internal/formats"
	"orcaton/internal/num"
	"orcaton/internal/xmldom"
)

type Importer struct{}

func (Importer) Name() string         { return "xmeml" }
func (Importer) Title() string        { return "FCP7 XML (xmeml)" }
func (Importer) Extensions() []string { return []string{".xml"} }

func (Importer) Sniff(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, 4096)
	n, _ := f.Read(head)
	return bytes.Contains(head[:n], []byte("<xmeml")), nil
}

func text(el *xmldom.Node, path string) *string {
	if el == nil {
		return nil
	}
	v, ok := el.FindText(path)
	if !ok {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

func textOr(el *xmldom.Node, path, def string) string {
	if v := text(el, path); v != nil {
		return *v
	}
	return def
}

func intOf(el *xmldom.Node, path string) *int64 {
	v := text(el, path)
	if v == nil || *v == "" {
		return nil
	}
	q, ok := num.ParseFraction(*v)
	if !ok {
		return nil
	}
	n := new(big.Int).Quo(q.Num(), q.Denom())
	i := n.Int64()
	return &i
}

func intOr(el *xmldom.Node, path string, def int64) int64 {
	if v := intOf(el, path); v != nil {
		return *v
	}
	return def
}

func toIntPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func rateOf(el *xmldom.Node, path string) *core.Rate {
	if el == nil {
		return nil
	}
	r := el.Find(path)
	if r == nil {
		return nil
	}
	tb := text(r, "timebase")
	if tb == nil {
		return nil
	}
	rate, err := core.RateFromXmeml(*tb, text(r, "ntsc"))
	if err != nil || !rate.Valid() {
		return nil
	}
	return &rate
}

func (imp Importer) Read(path string) (*core.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return imp.ReadData(data, path)
}

func (imp Importer) ReadData(data []byte, path string) (*core.Project, error) {
	root, err := xmldom.Parse(data, true)
	if err != nil {
		return nil, formats.Errorf("%s: invalid XML: %v", path, err)
	}
	if root.Tag != "xmeml" {
		return nil, formats.Errorf("%s: root <%s>, expected <xmeml>", path, root.Tag)
	}
	p := core.NewProject(formats.Stem(path))
	p.SourceFormat = imp.Name()
	p.SourcePath = path
	if v, ok := root.Get("version"); ok {
		p.XmemlVersion = &v
	}
	for _, f := range root.Iter("file") {
		fid := f.Attr("id")
		if fid != "" && f.Len() > 0 {
			if _, ok := p.Media[fid]; !ok {
				p.AddMedia(media(f))
			}
		}
	}
	for _, s := range root.Iter("sequence") {
		seq, err := sequence(s, p)
		if err != nil {
			return nil, err
		}
		p.Sequences = append(p.Sequences, seq)
	}
	if len(p.Sequences) == 0 {
		return nil, formats.Errorf("%s: no <sequence>", path)
	}
	return p, nil
}

func media(f *xmldom.Node) *core.MediaRef {
	fid := f.Attr("id")
	rate := rateOf(f, "rate")
	url := text(f, "pathurl")
	name := textOr(f, "name", fid)
	if name == "" {
		name = fid
	}
	m := &core.MediaRef{ID: fid, Name: name, Rate: rate}
	if url != nil {
		u := *url
		m.URL = &u
		if u != "" {
			pth := formats.URLToPath(u)
			m.Path = &pth
		}
	}
	if dur := intOf(f, "duration"); dur != nil && rate != nil {
		m.Duration = rate.FramesToSeconds(num.I(*dur))
	}
	if tc := f.Find("timecode"); tc != nil {
		m.TCString = text(tc, "string")
		frame := intOf(tc, "frame")
		tcRate := rateOf(tc, "rate")
		if tcRate == nil {
			tcRate = rate
		}
		dfmt := strings.ToUpper(textOr(tc, "displayformat", ""))
		if dfmt == "" {
			dfmt = "NDF"
		}
		if frame == nil && m.TCString != nil && *m.TCString != "" && tcRate != nil {
			df := dfmt == "DF"
			if fr, err := core.TCToFrames(*m.TCString, *tcRate, &df); err == nil {
				frame = &fr
			}
		}
		if frame != nil && tcRate != nil {
			m.TCStart = tcRate.FramesToSeconds(num.I(*frame))
		}
		m.TCDisplayFormat = dfmt
	}
	if strings.ToLower(strings.TrimSpace(textOr(f, "mediaSource", ""))) == "slug" &&
		(url == nil || *url == "") {
		m.Offline = true
		m.FileName = m.Name
	}
	md := f.Find("media")
	m.HasVideo = md != nil && md.Find("video") != nil
	m.HasAudio = md != nil && md.Find("audio") != nil
	if md != nil {
		m.Width = toIntPtr(intOf(md, "video/samplecharacteristics/width"))
		m.Height = toIntPtr(intOf(md, "video/samplecharacteristics/height"))
		m.AudioChannels = toIntPtr(intOf(md, "audio/channelcount"))
		m.SampleRate = toIntPtr(intOf(md, "audio/samplecharacteristics/samplerate"))
	}
	return m
}

func sequence(s *xmldom.Node, p *core.Project) (*core.Sequence, error) {
	rate := rateOf(s, "rate")
	if rate == nil {
		return nil, formats.Errorf("<sequence> has no valid <rate>")
	}
	name := textOr(s, "name", "Sequence")
	if name == "" {
		name = "Sequence"
	}
	seq := &core.Sequence{Name: name, Rate: *rate, TCStart: num.I(0)}
	if v, ok := s.Get("id"); ok {
		seq.XmemlID = &v
	}
	if tc := s.Find("timecode"); tc != nil {
		if fr := intOf(tc, "frame"); fr != nil {
			r := rateOf(tc, "rate")
			if r == nil {
				r = rate
			}
			seq.TCStart = r.FramesToSeconds(num.I(*fr))
		}
		seq.TCStringSrc = text(tc, "string")
	}
	fmtEl := s.Find("media/video/format/samplecharacteristics")
	if fmtEl != nil {
		seq.Width = toIntPtr(intOf(fmtEl, "width"))
		seq.Height = toIntPtr(intOf(fmtEl, "height"))
	}
	seq.SampleRate = toIntPtr(intOf(s, "media/audio/format/samplecharacteristics/samplerate"))
	for _, kind := range []core.TrackKind{core.Video, core.Audio} {
		for i, t := range s.FindAll("media/" + string(kind) + "/track") {
			seq.Tracks = append(seq.Tracks, track(t, kind, i+1, *rate, p))
		}
	}
	return seq, nil
}

func boolText(el *xmldom.Node, path, def string) bool {
	v := textOr(el, path, def)
	if v == "" {
		v = def
	}
	return strings.ToUpper(v) == "TRUE"
}

func track(t *xmldom.Node, kind core.TrackKind, index int, seqRate core.Rate,
	p *core.Project) *core.Track {
	tr := &core.Track{Kind: kind, Index: index,
		Enabled: boolText(t, "enabled", "TRUE"), Locked: boolText(t, "locked", "FALSE")}
	prevEnd := num.I(0)
	for _, c := range t.FindAll("clipitem") {
		if c.Len() == 0 {
			continue
		}
		cl := clip(c, seqRate, p, prevEnd)
		tr.Clips = append(tr.Clips, cl)
		prevEnd = cl.End()
	}
	return tr
}

func clip(c *xmldom.Node, seqRate core.Rate, p *core.Project, prevEnd num.Q) *core.Clip {
	rate := seqRate
	if r := rateOf(c, "rate"); r != nil {
		rate = *r
	}
	startF, endF := intOr(c, "start", 0), intOr(c, "end", 0)
	inF, outF := intOr(c, "in", 0), intOr(c, "out", 0)
	start := prevEnd
	if startF >= 0 {
		start = rate.FramesToSeconds(num.I(startF))
	}
	var dur num.Q
	if endF >= 0 && startF >= 0 {
		dur = rate.FramesToSeconds(num.I(endF - startF))
	} else {
		dur = rate.FramesToSeconds(num.I(outF - inF))
	}
	var m *core.MediaRef
	if fel := c.Find("file"); fel != nil {
		m = p.Media[fel.Attr("id")]
	}
	name := textOr(c, "name", "")
	if m != nil && m.Offline && name != "" {
		m.Name = name
	}
	set := map[string]bool{}
	for _, lk := range c.FindAll("link") {
		v, _ := lk.FindText("linkclipref")
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	var links []string
	for k := range set {
		links = append(links, k)
	}
	sort.Strings(links)
	var group *string
	cid, hasID := c.Get("id")
	if len(links) > 0 && hasID && cid != "" {
		g := cid
		for _, l := range links {
			if l < g {
				g = l
			}
		}
		group = &g
	}
	srcIn := inF
	if srcIn < 0 {
		srcIn = 0
	}
	cl := &core.Clip{
		Name:         name,
		Media:        m,
		Start:        start,
		Duration:     dur,
		SourceIn:     rate.FramesToSeconds(num.I(srcIn)),
		Enabled:      boolText(c, "enabled", "TRUE"),
		Label:        text(c, "labels/label2"),
		SyncStatus:   core.StatusUnknown,
		LinkGroup:    group,
		SourceTrack:  toIntPtr(intOf(c, "sourcetrack/trackindex")),
		MasterClipID: text(c, "masterclipid"),
	}
	if hasID {
		cl.ID = &cid
	}
	return cl
}
