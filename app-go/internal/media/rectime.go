package media

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"orcaton/internal/formats"
	"orcaton/internal/pyio"
)

var renderers = []string{"davinci", "resolve", "blackmagic", "premiere", "adobe", "final cut",
	"lavf", "handbrake", "media encoder", "shutter encoder", "compressor"}

var videoExt = map[string]bool{".mp4": true, ".mov": true, ".mxf": true, ".mts": true,
	".m2ts": true, ".avi": true, ".mkv": true, ".insv": true}

var creationRe = regexp.MustCompile(`(?i)<CreationDate\s+value="([^"]+)"`)

type RecordTime struct {
	Start  float64
	Source string
	Origin string
}

func rtKey(name string) string { return formats.Casefold(formats.NFC(formats.Stem(name))) }

func ParseTime(text string) (float64, error) { return pyio.ParseISO(text) }

func ReadSidecar(path string) (float64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	buf := make([]byte, 65536)
	n, _ := f.Read(buf)
	head := strings.ToValidUTF8(string(buf[:n]), "�")
	m := creationRe.FindStringSubmatch(head)
	if m == nil {
		return 0, false
	}
	t, err := ParseTime(m[1])
	if err != nil {
		return 0, false
	}
	return t, true
}

func ContainerTime(path string) (float64, bool) {
	info, err := Probe(path)
	if err != nil {
		return 0, false
	}
	tags := map[string]string{}
	if fm, ok := info["format"].(map[string]any); ok {
		if tg, ok := fm["tags"].(map[string]any); ok {
			for k, v := range tg {
				tags[strings.ToLower(k)] = fmt.Sprint(v)
			}
		}
	}
	parts := []string{tags["encoder"]}
	if sts, ok := info["streams"].([]any); ok {
		for _, s := range sts {
			enc := ""
			if sm, ok := s.(map[string]any); ok {
				if tg, ok := sm["tags"].(map[string]any); ok {
					if e, ok := tg["encoder"]; ok {
						enc = fmt.Sprint(e)
					}
				}
			}
			parts = append(parts, enc)
		}
	}
	enc := strings.ToLower(strings.Join(parts, " "))
	for _, r := range renderers {
		if strings.Contains(enc, r) {
			return 0, false
		}
	}
	ct := tags["creation_time"]
	if ct == "" {
		return 0, false
	}
	t, err := ParseTime(ct)
	if err != nil {
		return 0, false
	}
	return t, true
}

type RecordTimes struct {
	ProbeMedia   bool
	NoMediaFiles bool
	table        map[string]float64
	sidecars     map[string]string
	videos       map[string]string
	mu           sync.Mutex
	cache        map[[2]string]*RecordTime
}

func NewRecordTimes(originals []string, tables []string, probeMedia bool) (*RecordTimes, error) {
	rt := &RecordTimes{ProbeMedia: probeMedia, table: map[string]float64{},
		sidecars: map[string]string{}, videos: map[string]string{},
		cache: map[[2]string]*RecordTime{}}
	for _, t := range tables {
		tb, err := LoadTable(t)
		if err != nil {
			return nil, err
		}
		for k, v := range tb {
			rt.table[k] = v
		}
	}
	for _, root := range originals {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			ext := strings.ToLower(filepath.Ext(name))
			stem := strings.TrimSuffix(name, filepath.Ext(name))
			if ext == ".xml" && strings.HasSuffix(strings.ToUpper(stem), "M01") {
				k := rtKey(stem[:len(stem)-3])
				if _, ok := rt.sidecars[k]; !ok {
					rt.sidecars[k] = p
				}
			} else if videoExt[ext] {
				k := rtKey(name)
				if _, ok := rt.videos[k]; !ok {
					rt.videos[k] = p
				}
			}
			return nil
		})
	}
	return rt, nil
}

func (rt *RecordTimes) Get(name, path string) *RecordTime {
	ck := [2]string{name, path}
	rt.mu.Lock()
	if v, ok := rt.cache[ck]; ok {
		rt.mu.Unlock()
		return v
	}
	rt.mu.Unlock()
	v := rt.find(name, path)
	rt.mu.Lock()
	rt.cache[ck] = v
	rt.mu.Unlock()
	return v
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func (rt *RecordTimes) find(name, path string) *RecordTime {
	k := rtKey(name)
	if t, ok := rt.table[k]; ok {
		return &RecordTime{t, "table", "table"}
	}
	if path != "" && !rt.NoMediaFiles && isDir(filepath.Dir(path)) {
		base := filepath.Base(path)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		if strings.HasPrefix(base, ".") && filepath.Ext(base) == base {
			stem = base
		}
		for _, cand := range []string{filepath.Join(filepath.Dir(path), stem+"M01.XML"),
			filepath.Join(filepath.Dir(path), stem+"M01.xml")} {
			if isFile(cand) {
				if t, ok := ReadSidecar(cand); ok {
					return &RecordTime{t, "sidecar", cand}
				}
			}
		}
	}
	if sc, ok := rt.sidecars[k]; ok {
		if t, ok := ReadSidecar(sc); ok {
			return &RecordTime{t, "sidecar", sc}
		}
	}
	if rt.ProbeMedia {
		if v, ok := rt.videos[k]; ok {
			if t, ok := ContainerTime(v); ok {
				return &RecordTime{t, "container", v}
			}
		}
		if path != "" && !rt.NoMediaFiles && isFile(path) {
			if t, ok := ContainerTime(path); ok {
				return &RecordTime{t, "container", path}
			}
		}
	}
	return nil
}

func LoadTable(path string) (map[string]float64, error) {
	rows, err := pyio.ReadCSVDicts(path)
	if err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for _, r := range rows {
		if r["name"] != "" && r["record_start"] != "" {
			t, err := ParseTime(r["record_start"])
			if err != nil {
				return nil, err
			}
			out[rtKey(r["name"])] = t
		}
	}
	return out, nil
}

type NamedTime struct {
	Name string
	RT   *RecordTime
}

func SaveTable(rows []NamedTime, path string) (int, error) {
	var out [][]string
	for _, r := range rows {
		out = append(out, []string{r.Name, pyio.ISOUTC(r.RT.Start), r.RT.Source, r.RT.Origin})
	}
	if err := pyio.WriteCSVFile(path, []string{"name", "record_start", "source", "origin"}, out); err != nil {
		return 0, err
	}
	return len(rows), nil
}
