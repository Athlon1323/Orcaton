package drt

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"orcaton/internal/core"
	"orcaton/internal/formats"
	"orcaton/internal/num"
)

var (
	itemOpenRe  = regexp.MustCompile(`<(Sm2TiVideoClip|Sm2TiAudioClip|Sm2TiTransition) DbId="([^"]+)">`)
	trackOpenRe = regexp.MustCompile(`<Sm2TiTrack DbId="([^"]+)">`)
	startRe     = regexp.MustCompile(`<Start>([^<]*)</Start>`)
	markedRe    = regexp.MustCompile(`[ \t\n\r\f\v]*<Element>[ \t\n\r\f\v]*<!--orcaton:removed-->[ \t\n\r\f\v]*</Element>`)
	subframe    = num.F(1, 1_000_000)
	removedMark = []byte("<!--orcaton:removed-->")
)

type PatchStats struct {
	Moved              int
	Removed            int
	TransitionsRemoved int
	Renamed            bool
	Extents            bool
	ChangedEntries     []string
}

func (s *PatchStats) Changed() bool { return len(s.ChangedEntries) > 0 }

type Exporter struct{ Last *PatchStats }

var DefaultExporter = &Exporter{Last: &PatchStats{}}

func (*Exporter) Name() string { return "drt" }
func (*Exporter) Title() string {
	return "DRT timeline (.drt) — патч исходного .drt"
}
func (*Exporter) Extensions() []string { return []string{".drt"} }

type zentry struct {
	hdr  zip.FileHeader
	data []byte
}

func (e *Exporter) Write(p *core.Project, path string) error {
	src, err := source(p)
	if err != nil {
		return err
	}
	z, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	var entries []*zentry
	data := map[string][]byte{}
	var order []string
	for _, f := range z.File {
		d, err := readEntry(f)
		if err != nil {
			z.Close()
			return err
		}
		entries = append(entries, &zentry{hdr: f.FileHeader})
		if _, ok := data[f.Name]; !ok {
			order = append(order, f.Name)
		}
		data[f.Name] = d
	}
	z.Close()
	stats := &PatchStats{}
	for _, seq := range p.Sequences {
		if err := patchSequence(seq, data, order, stats); err != nil {
			return err
		}
	}
	e.Last = stats
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if !stats.Changed() {
		if absPath(src) != absPath(path) {
			return copyFile(src, path)
		}
		return nil
	}
	tmp := path + ".tmp"
	if err := writeZip(tmp, entries, data); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func writeZip(path string, entries []*zentry, data map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	for _, en := range entries {
		d := data[en.hdr.Name]
		h := &zip.FileHeader{
			Name:           en.hdr.Name,
			Method:         en.hdr.Method,
			ModifiedTime:   en.hdr.ModifiedTime,
			ModifiedDate:   en.hdr.ModifiedDate,
			ExternalAttrs:  en.hdr.ExternalAttrs,
			CreatorVersion: en.hdr.CreatorVersion&0xff00 | 20,
			ReaderVersion:  20,
		}
		if !isASCII(h.Name) {
			h.Flags |= 0x800
		}
		h.CRC32 = crc32.ChecksumIEEE(d)
		h.UncompressedSize64 = uint64(len(d))
		var payload []byte
		if h.Method == zip.Deflate {
			var buf bytes.Buffer
			fw, _ := flate.NewWriter(&buf, 6)
			fw.Write(d)
			fw.Close()
			payload = buf.Bytes()
		} else {
			h.Method = zip.Store
			payload = d
		}
		h.CompressedSize64 = uint64(len(payload))
		ww, err := w.CreateRaw(h)
		if err != nil {
			f.Close()
			return err
		}
		if _, err := ww.Write(payload); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func source(p *core.Project) (string, error) {
	if p.SourceFormat != "drt" || p.SourcePath == "" {
		sf := p.SourceFormat
		if sf == "" {
			sf = "неизвестно"
		}
		return "", formats.Errorf("DRT записывается только патчем исходного .drt: вход тоже должен "+
			"быть .drt (сейчас — %s). Для входа XML выберите выход .xml", sf)
	}
	if st, err := os.Stat(p.SourcePath); err != nil || st.IsDir() {
		return "", formats.Errorf("Исходный .drt не найден (нужен для записи патчем): %s", p.SourcePath)
	}
	return p.SourcePath, nil
}

type match struct {
	start, end         int
	tag, id            string
	bodyStart, bodyEnd int
}

func findItems(buf []byte) []match {
	var out []match
	pos := 0
	for pos < len(buf) {
		loc := itemOpenRe.FindSubmatchIndex(buf[pos:])
		if loc == nil {
			break
		}
		s := pos + loc[0]
		tag := string(buf[pos+loc[2] : pos+loc[3]])
		id := string(buf[pos+loc[4] : pos+loc[5]])
		bodyStart := pos + loc[1]
		closeTag := []byte("</" + tag + ">")
		k := bytes.Index(buf[bodyStart:], closeTag)
		if k < 0 {
			pos = s + 1
			continue
		}
		bodyEnd := bodyStart + k
		out = append(out, match{s, bodyEnd + len(closeTag), tag, id, bodyStart, bodyEnd})
		pos = bodyEnd + len(closeTag)
	}
	return out
}

func findTracks(buf []byte) []match {
	var out []match
	closeTag := []byte("</Sm2TiTrack>")
	pos := 0
	for pos < len(buf) {
		loc := trackOpenRe.FindSubmatchIndex(buf[pos:])
		if loc == nil {
			break
		}
		s := pos + loc[0]
		bodyStart := pos + loc[1]
		k := bytes.Index(buf[bodyStart:], closeTag)
		if k < 0 {
			pos = s + 1
			continue
		}
		bodyEnd := bodyStart + k
		out = append(out, match{s, bodyEnd + len(closeTag), "Sm2TiTrack",
			string(buf[pos+loc[2] : pos+loc[3]]), bodyStart, bodyEnd})
		pos = bodyEnd + len(closeTag)
	}
	return out
}

func isClipTag(tag string) bool { return tag == ClipTags[0] || tag == ClipTags[1] }

func patchSequence(seq *core.Sequence, data map[string][]byte, order []string,
	stats *PatchStats) error {
	entry := seq.DrtEntry
	if _, ok := data[entry]; entry == "" || !ok {
		return formats.Errorf("Таймлайн '%s': не найден SeqContainer в исходном .drt", seq.Name)
	}
	rate := seq.Rate
	tc := rate.SecondsToFrames(seq.TCStart)
	clips := map[string]*core.Clip{}
	for _, tcl := range seq.Clips() {
		if tcl.Clip.ID != nil && *tcl.Clip.ID != "" {
			clips[*tcl.Clip.ID] = tcl.Clip
		}
	}
	newStart := func(c *core.Clip, old num.Q) num.Q {
		fr := num.Add(rate.SecondsToFrames(c.Start), tc)
		if num.Le(num.Abs(num.Sub(fr, old)), subframe) {
			return nil
		}
		if c.Media == nil || c.Media.DrtVideo {
			return num.I(num.RoundInt(fr))
		}
		return num.LimitDenominator(fr, 1_000_000)
	}

	raw := data[entry]
	moved, removed := 0, 0
	var ferr error
	var out bytes.Buffer
	last := 0
	for _, tm := range findTracks(raw) {
		out.Write(raw[last:tm.bodyStart])
		body := raw[tm.bodyStart:tm.bodyEnd]
		dirty := false
		var nb bytes.Buffer
		lb := 0
		for _, im := range findItems(body) {
			nb.Write(body[lb:im.start])
			whole := body[im.start:im.end]
			lb = im.end
			if !isClipTag(im.tag) {
				nb.Write(whole)
				continue
			}
			c := clips[im.id]
			if c == nil {
				removed++
				dirty = true
				nb.Write(removedMark)
				continue
			}
			inner := body[im.bodyStart:im.bodyEnd]
			loc := startRe.FindSubmatchIndex(inner)
			if loc == nil {
				if ferr == nil {
					ferr = formats.Errorf("Клип %s: нет поля Start", im.id)
				}
				nb.Write(whole)
				continue
			}
			oldS := string(inner[loc[2]:loc[3]])
			old := DecodeTime(&oldS)
			if old == nil {
				old = num.I(0)
			}
			fr := newStart(c, old)
			if fr == nil || num.Eq(fr, old) {
				nb.Write(whole)
				continue
			}
			moved++
			dirty = true
			off := im.bodyStart - im.start
			s, e := off+loc[2], off+loc[3]
			nb.Write(whole[:s])
			nb.WriteString(EncodeTime(fr))
			nb.Write(whole[e:])
		}
		nb.Write(body[lb:])
		nbody := nb.Bytes()
		if dirty {
			var n int
			nbody, n = dropTransitions(nbody)
			stats.TransitionsRemoved += n
		}
		nbody = markedRe.ReplaceAll(nbody, nil)
		out.Write(nbody)
		out.WriteString("</Sm2TiTrack>")
		last = tm.end
	}
	out.Write(raw[last:])
	if ferr != nil {
		return ferr
	}
	patched := out.Bytes()
	stats.Moved += moved
	stats.Removed += removed
	if !bytes.Equal(patched, raw) {
		data[entry] = patched
		stats.ChangedEntries = append(stats.ChangedEntries, entry)
		patchPool(seq, data, order, stats)
	} else if renameNeeded(seq, data, order) {
		patchPool(seq, data, order, stats)
	}
	return nil
}

func dropTransitions(body []byte) ([]byte, int) {
	n := 0
	var b bytes.Buffer
	last := 0
	for _, m := range findItems(body) {
		b.Write(body[last:m.start])
		if m.tag == "Sm2TiTransition" {
			n++
			b.Write(removedMark)
		} else {
			b.Write(body[m.start:m.end])
		}
		last = m.end
	}
	b.Write(body[last:])
	return b.Bytes(), n
}

func poolEntries(order []string) []string {
	var out []string
	for _, n := range order {
		if strings.HasPrefix(n, "MediaPool/") && strings.HasSuffix(n, ".xml") {
			out = append(out, n)
		}
	}
	return out
}

func renameNeeded(seq *core.Sequence, data map[string][]byte, order []string) bool {
	if seq.DrtTimelineID == nil || *seq.DrtTimelineID == "" {
		return false
	}
	for _, n := range poolEntries(order) {
		if s, e, ok := timelineName(data[n], *seq.DrtTimelineID); ok {
			return unescape(data[n][s:e]) != seq.Name
		}
	}
	return false
}

func patchPool(seq *core.Sequence, data map[string][]byte, order []string, stats *PatchStats) {
	endSec := num.I(0)
	for _, tc := range seq.Clips() {
		endSec = num.Max(endSec, tc.Clip.End())
	}
	for _, n := range poolEntries(order) {
		raw := data[n]
		buf := raw
		if seq.DrtSeqID != nil && *seq.DrtSeqID != "" {
			buf = patchExtents(buf, *seq.DrtSeqID, num.Float(endSec), stats)
		}
		if seq.DrtTimelineID != nil && *seq.DrtTimelineID != "" {
			if s, e, ok := timelineName(buf, *seq.DrtTimelineID); ok && unescape(buf[s:e]) != seq.Name {
				nb := append(append(append([]byte(nil), buf[:s]...), []byte(escape(seq.Name))...), buf[e:]...)
				buf = nb
				stats.Renamed = true
			}
		}
		if !bytes.Equal(buf, raw) {
			data[n] = buf
			stats.ChangedEntries = append(stats.ChangedEntries, n)
		}
	}
}

func timelineName(buf []byte, tlID string) (int, int, bool) {
	open := []byte(`<Sm2MpTimelineClip DbId="` + tlID + `">`)
	stop1, stop2 := []byte("<Sm2"), []byte("</Sm2MpTimelineClip>")
	nameOpen, nameClose := []byte("<Name>"), []byte("</Name>")
	from := 0
	for {
		k := bytes.Index(buf[from:], open)
		if k < 0 {
			return 0, 0, false
		}
		s := from + k + len(open)
		for p := s; p <= len(buf); p++ {
			if bytes.HasPrefix(buf[p:], nameOpen) {
				q := p + len(nameOpen)
				r := q
				for r < len(buf) && buf[r] != '<' {
					r++
				}
				if bytes.HasPrefix(buf[r:], nameClose) {
					return q, r, true
				}
			}
			if p == len(buf) || bytes.HasPrefix(buf[p:], stop1) || bytes.HasPrefix(buf[p:], stop2) {
				break
			}
		}
		from = from + k + 1
	}
}

func patchExtents(buf []byte, seqID string, endSec float64, stats *PatchStats) []byte {
	open := []byte(`<Sm2Sequence DbId="` + seqID + `">`)
	stop := []byte("</Sm2Sequence>")
	eo, ec := []byte("<MediaExtents>"), []byte("</MediaExtents>")
	from := 0
	for {
		k := bytes.Index(buf[from:], open)
		if k < 0 {
			return buf
		}
		s := from + k + len(open)
		for p := s; p <= len(buf); p++ {
			if bytes.HasPrefix(buf[p:], eo) {
				q := p + len(eo)
				r := q
				for r < len(buf) && isHexByte(buf[r]) {
					r++
				}
				if bytes.HasPrefix(buf[r:], ec) {
					val := string(buf[q:r])
					st, dur, ok := DecodeExtents(&val)
					if !ok || absf(dur-endSec) < 1e-6 {
						return buf
					}
					stats.Extents = true
					nv := EncodeExtents(st, endSec)
					return append(append(append([]byte(nil), buf[:q]...), []byte(nv)...), buf[r:]...)
				}
			}
			if p == len(buf) || bytes.HasPrefix(buf[p:], stop) {
				break
			}
		}
		from = from + k + 1
	}
}

func isHexByte(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

var escR = strings.NewReplacer("&", "&amp;", ">", "&gt;", "<", "&lt;")
var unescR = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")

func escape(s string) string { return escR.Replace(s) }

func unescape(b []byte) string { return unescR.Replace(string(b)) }
