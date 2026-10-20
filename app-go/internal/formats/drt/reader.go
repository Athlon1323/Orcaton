package drt

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"orcaton/internal/core"
	"orcaton/internal/formats"
	"orcaton/internal/num"
	"orcaton/internal/xmldom"
)

var nsFix = regexp.MustCompile(`<(/?)([A-Za-z_]\w*)::(\w+)`)

var DefaultTCStartSec = num.I(3600)

var ClipTags = []string{"Sm2TiVideoClip", "Sm2TiAudioClip"}

func LoadLenientXML(data []byte) (*xmldom.Node, error) {
	fixed := nsFix.ReplaceAll(data, []byte("<${1}${2}__${3}"))
	root, err := xmldom.Parse(fixed, false)
	if err != nil {
		return nil, formats.Errorf("Не удалось разобрать XML внутри DRT")
	}
	return root, nil
}

func t(el *xmldom.Node, path string) *string {
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

func tOr(el *xmldom.Node, path string) string {
	if v := t(el, path); v != nil {
		return *v
	}
	return ""
}

type Importer struct{}

func (Importer) Name() string         { return "drt" }
func (Importer) Title() string        { return "DRT timeline (.drt)" }
func (Importer) Extensions() []string { return []string{".drt"} }

func (Importer) Sniff(path string) (bool, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return false, nil
	}
	defer z.Close()
	return hasDrtEntries(z.File), nil
}

func hasDrtEntries(files []*zip.File) bool {
	hasProj, hasSeq := false, false
	for _, f := range files {
		if f.Name == "project.xml" {
			hasProj = true
		}
		if strings.HasPrefix(f.Name, "SeqContainer/") {
			hasSeq = true
		}
	}
	return hasProj && hasSeq
}

func readEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func absPath(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if r, err := filepath.EvalSymlinks(a); err == nil {
		return r
	}
	return a
}

type seqXML struct {
	entry string
	root  *xmldom.Node
}

func (imp Importer) Read(path string) (*core.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return imp.ReadData(data, path)
}

func (imp Importer) ReadData(data []byte, path string) (*core.Project, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || !hasDrtEntries(z.File) {
		return nil, formats.Errorf("%s: не похоже на .drt (нет project.xml / SeqContainer/)", path)
	}
	byName := map[string]*zip.File{}
	for _, f := range z.File {
		byName[f.Name] = f
	}
	projData, err := readEntry(byName["project.xml"])
	if err != nil {
		return nil, err
	}
	projX, err := LoadLenientXML(projData)
	if err != nil {
		return nil, err
	}
	var pools []*xmldom.Node
	var seqNames []string
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "MediaPool/") && strings.HasSuffix(f.Name, ".xml") {
			d, err := readEntry(f)
			if err != nil {
				return nil, err
			}
			x, err := LoadLenientXML(d)
			if err != nil {
				return nil, err
			}
			pools = append(pools, x)
		}
		if strings.HasPrefix(f.Name, "SeqContainer/") {
			seqNames = append(seqNames, f.Name)
		}
	}
	sort.Strings(seqNames)
	var seqs []seqXML
	for _, n := range seqNames {
		d, err := readEntry(byName[n])
		if err != nil {
			return nil, err
		}
		x, err := LoadLenientXML(d)
		if err != nil {
			return nil, err
		}
		seqs = append(seqs, seqXML{n, x})
	}

	name := tOr(projX, "ProjectName")
	if name == "" {
		name = formats.Stem(path)
	}
	p := core.NewProject(name)
	p.SourceFormat = imp.Name()
	p.SourcePath = absPath(path)
	p.DBVersions = dbVersions(projData)

	poolNames := map[string]string{}
	var seqMeta []*xmldom.Node
	for _, pool := range pools {
		for _, el := range pool.Iter() {
			if id := el.Attr("DbId"); strings.HasPrefix(el.Tag, "Sm2Mp") && id != "" {
				if nm := tOr(el, "Name"); nm != "" {
					poolNames[id] = nm
				}
			}
			if el.Tag == "Sm2Sequence" {
				seqMeta = append(seqMeta, el)
			}
		}
	}
	byRef := map[string]*xmldom.Node{}
	for _, el := range seqMeta {
		if ref := seqRef(el); ref != "" {
			byRef[ref] = el
		}
	}
	for i, sx := range seqs {
		meta := byRef[sx.root.Attr("DbId")]
		if meta == nil && i < len(seqMeta) {
			meta = seqMeta[i]
		}
		tl := timelineOf(meta)
		nm := ""
		if tl != nil {
			nm = tOr(tl, "Name")
		}
		if nm == "" {
			nm = "Timeline " + strconv.Itoa(i+1)
		}
		seq := sequence(sx.root, meta, nm, p, poolNames)
		seq.DrtEntry = sx.entry
		if meta != nil {
			if v, ok := meta.Get("DbId"); ok {
				seq.DrtSeqID = &v
			}
		}
		if tl != nil {
			if v, ok := tl.Get("DbId"); ok {
				seq.DrtTimelineID = &v
			}
		}
		p.Sequences = append(p.Sequences, seq)
	}
	if len(p.Sequences) == 0 {
		return nil, formats.Errorf("%s: в DRT нет ни одной секвенции (SeqContainer/*.xml)", path)
	}
	return p, nil
}

func seqRef(meta *xmldom.Node) string {
	strs := UTF16BEStrings(tOr(meta, "FieldsBlob"), 3)
	for i := 0; i+1 < len(strs); i++ {
		if strs[i] == "SeqRef" {
			return strs[i+1]
		}
	}
	return ""
}

func timelineOf(meta *xmldom.Node) *xmldom.Node {
	if meta == nil {
		return nil
	}
	for _, a := range meta.Ancestors() {
		if a.Tag == "Sm2MpTimelineClip" {
			return a
		}
	}
	return nil
}

var dbVerRe = regexp.MustCompile(`(\w+)="([^"]+)"`)

func dbVersions(projData []byte) [][2]string {
	head := projData
	if len(head) > 300 {
		head = head[:300]
	}
	s := strings.ToValidUTF8(string(head), "�")
	s, _, _ = strings.Cut(s, "-->")
	var out [][2]string
	idx := map[string]int{}
	for _, m := range dbVerRe.FindAllStringSubmatch(s, -1) {
		if i, ok := idx[m[1]]; ok {
			out[i][1] = m[2]
			continue
		}
		idx[m[1]] = len(out)
		out = append(out, [2]string{m[1], m[2]})
	}
	return out
}

func sequence(sx, meta *xmldom.Node, name string, p *core.Project,
	poolNames map[string]string) *core.Sequence {
	rate := core.RateInt(25)
	if meta != nil {
		if fps, ok := DecodeDoubleLE(t(meta, "FrameRate")); ok && core.ValidFloat(fps) {
			rate = core.RateFloat(fps)
		}
	}
	tcStart := DefaultTCStartSec
	if meta != nil {
		if st, _, ok := DecodeExtents(t(meta, "MediaExtents")); ok {
			fr := math.RoundToEven(st * rate.Float())
			tcStart = rate.FramesToSeconds(num.FromFloat(fr))
		}
	}
	seq := &core.Sequence{Name: name, Rate: rate, TCStart: tcStart}
	if meta != nil {
		if w, h, ok := DecodeResolution(t(meta, "Resolution")); ok {
			seq.Width, seq.Height = &w, &h
		}
	}
	if v, ok := sx.Get("DbId"); ok {
		seq.DrtContainerID = &v
	}
	tcFrames := rate.SecondsToFrames(seq.TCStart)
	for _, kv := range []struct {
		kind core.TrackKind
		vec  string
	}{{core.Video, "VideoTrackVec"}, {core.Audio, "AudioTrackVec"}} {
		v := sx.Find(kv.vec)
		if v == nil {
			continue
		}
		for i, tr := range v.Iter("Sm2TiTrack") {
			track := &core.Track{Kind: kv.kind, Index: i + 1, Enabled: true}
			if nm := tOr(tr, "UserDefinedName"); nm != "" {
				track.Name = &nm
			}
			track.DrtType = t(tr, "Type")
			if id, ok := tr.Get("DbId"); ok {
				track.DrtTrackID = &id
			}
			items := tr.Find("Items")
			if items == nil {
				seq.Tracks = append(seq.Tracks, track)
				continue
			}
			for _, el := range items.Iter(ClipTags...) {
				c := clip(el, rate, tcFrames, p, poolNames)
				if kv.kind == core.Video && c.Media != nil {
					c.Media.DrtVideo = true
				}
				track.Clips = append(track.Clips, c)
			}
			seq.Tracks = append(seq.Tracks, track)
		}
	}
	return seq
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func clip(el *xmldom.Node, rate core.Rate, tcFrames num.Q, p *core.Project,
	poolNames map[string]string) *core.Clip {
	ref := tOr(el, "MediaRef")
	fpath := t(el, "MediaFilePath")
	offline := false
	if ref == "" && fpath != nil && *fpath != "" {
		ref = "offline:" + *fpath
		offline = true
	}
	m := p.Media[ref]
	if m == nil && ref != "" {
		name := poolNames[ref]
		if name == "" {
			if fpath != nil && *fpath != "" {
				name = formats.BaseName(*fpath)
			} else {
				name = ref
			}
		}
		m = &core.MediaRef{ID: ref, Name: name, Path: fpath, HasVideo: true, HasAudio: true,
			Offline: offline}
		if offline {
			m.FileName = name
		}
		if mfps, ok := DecodeDoubleLE(t(el, "MediaFrameRate")); ok && core.ValidFloat(mfps) {
			r := core.RateFloat(mfps)
			m.Rate = &r
		}
		p.AddMedia(m)
	}
	zero := num.I(0)
	start := DecodeTime(t(el, "Start"))
	if start == nil {
		start = zero
	}
	dur := DecodeTime(t(el, "Duration"))
	if dur == nil {
		dur = zero
	}
	srcIn := DecodeTime(t(el, "In"))
	if srcIn == nil {
		srcIn = zero
	}
	c := &core.Clip{
		Name:       tOr(el, "Name"),
		Media:      m,
		Start:      rate.FramesToSeconds(num.Sub(start, tcFrames)),
		Duration:   rate.FramesToSeconds(dur),
		SourceIn:   rate.FramesToSeconds(srcIn),
		Enabled:    true,
		SyncStatus: core.StatusUnknown,
		DrtTag:     el.Tag,
	}
	if trk := tOr(el, "MediaTrackIdx"); isDigits(trk) {
		n, _ := strconv.Atoi(trk)
		n++
		c.SourceTrack = &n
	}
	if id, ok := el.Get("DbId"); ok {
		c.ID = &id
	}
	return c
}
