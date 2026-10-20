package api

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"orcaton/internal/engine"
	"orcaton/internal/media"
	"orcaton/internal/settings"
)

type Settings struct {
	Options Options       `json:"options"`
	Media   MediaSettings `json:"media"`
	Files   []string      `json:"files"`
}

type MediaSettings struct {
	Map         string   `json:"map"`
	Search      []string `json:"search"`
	SearchDepth int      `json:"searchDepth"`
	KeepPaths   bool     `json:"keepPaths"`
	Originals   []string `json:"originals"`
}

func (m MediaSettings) SearchRoots() []string {
	var out []string
	for _, p := range strings.Split(m.Map, ";") {
		if p = strings.TrimSpace(p); p != "" && !strings.Contains(p, "=") {
			out = append(out, p)
		}
	}
	for _, p := range m.Search {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func DefaultSettings() Settings {
	return Settings{Options: DefaultOptions(), Media: MediaSettings{SearchDepth: media.DefaultSearchDepth,
		Search: []string{}, Originals: []string{}}, Files: []string{}}
}

func UserSettingsPath() string { return settings.UserPath() }

func ProjectSettingsPath(project string) string { return settings.ProjectPath(project) }

type field struct {
	key string
	ptr func(*Settings) any
}

var fields = []field{
	{"map", func(s *Settings) any { return &s.Media.Map }},
	{"search", func(s *Settings) any { return &s.Media.Search }},
	{"search-depth", func(s *Settings) any { return &s.Media.SearchDepth }},
	{"keep-paths", func(s *Settings) any { return &s.Media.KeepPaths }},
	{"originals", func(s *Settings) any { return &s.Media.Originals }},
	{"unsynced", func(s *Settings) any { return &s.Options.Unsynced }},
	{"order-by", func(s *Settings) any { return &s.Options.OrderBy }},
	{"singles", func(s *Settings) any { return &s.Options.Singles }},
	{"label", func(s *Settings) any { return &s.Options.UnsyncedLabel }},
	{"time-label", func(s *Settings) any { return &s.Options.TimeLabel }},
	{"min-conf", func(s *Settings) any { return &s.Options.MinConfidence }},
	{"min-ratio", func(s *Settings) any { return &s.Options.MinRatio }},
	{"auto-threshold", func(s *Settings) any { return &s.Options.AutoThreshold }},
	{"min-overlap", func(s *Settings) any { return &s.Options.MinOverlapSec }},
	{"max-offset", func(s *Settings) any { return &s.Options.MaxOffsetSec }},
	{"audio-mode", func(s *Settings) any { return &s.Options.AudioMode }},
	{"file-order", func(s *Settings) any { return &s.Options.FileOrder }},
	{"intra-lane", func(s *Settings) any { return &s.Options.IntraLane }},
	{"record-time", func(s *Settings) any { return &s.Options.RecordTime }},
	{"max-gap", func(s *Settings) any { return &s.Options.MaxGapSec }},
	{"keep-gap", func(s *Settings) any { return &s.Options.KeepGapSec }},
	{"jobs", func(s *Settings) any { return &s.Options.Jobs }},
	{"threads", func(s *Settings) any { return &s.Options.Threads }},
	{"mem-limit", func(s *Settings) any { return &s.Options.MemLimitMB }},
	{"cache", func(s *Settings) any { return &s.Options.CacheDir }},
}

func (f field) get(s *Settings) any {
	if f.key == "mem-limit" {
		return float64(s.Options.MemLimitMB) / 1024
	}
	v := reflect.ValueOf(f.ptr(s)).Elem().Interface()
	if l, ok := v.([]string); ok && l == nil {
		return []string{}
	}
	return v
}

func (f field) set(s *Settings, v any) {
	if f.key == "mem-limit" {
		s.Options.MemLimitMB = int(math.Round(v.(float64) * 1024))
		return
	}
	reflect.ValueOf(f.ptr(s)).Elem().Set(reflect.ValueOf(v))
}

func LoadSettings(project string) (Settings, error) {
	s := DefaultSettings()
	ls, err := settings.Load(project, "")
	if err != nil {
		return s, err
	}
	s.Files = append(s.Files, ls.Files()...)
	for _, f := range fields {
		if v, _ := ls.Get(f.key); v != nil {
			f.set(&s, v)
		}
	}
	if err := s.validate(); err != nil {
		return DefaultSettings(), fmt.Errorf("%s: %w", strings.Join(s.Files, "; "), err)
	}
	return s, nil
}

func (s Settings) validate() error {
	if d := s.Media.SearchDepth; d < 0 || d > 64 {
		return &engine.OptionError{Name: "SearchDepth", Msg: fmt.Sprintf("нужно 0…64, получено %d", d)}
	}
	return s.Options.Validate()
}

func SaveSettings(path string, s Settings) error {
	if err := s.validate(); err != nil {
		return err
	}
	cur, err := settings.Read(path)
	if err != nil {
		return err
	}
	d := DefaultSettings()
	var kvs []settings.KV
	for _, f := range fields {
		v := f.get(&s)
		present := false
		if cur != nil {
			_, present = cur.Vals[f.key]
		}
		if present || !reflect.DeepEqual(v, f.get(&d)) {
			kvs = append(kvs, settings.KV{Name: f.key, Val: v})
		}
	}
	return settings.Update(path, kvs)
}
