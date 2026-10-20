package settings

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"orcaton/internal/toml"
)

const (
	FileName = "orcaton.toml"
	Section  = "sync"
	EnvVar   = "ORCATON_CONFIG"
)

type Kind int

const (
	Bool Kind = iota
	Int
	Float
	String
	List
)

type Key struct {
	Name string
	Kind Kind
	Path bool
}

var Keys = []Key{
	{"map", String, true}, {"search", List, true}, {"search-depth", Int, false},
	{"keep-paths", Bool, false}, {"unsynced", String, false}, {"order-by", String, false},
	{"singles", String, false}, {"label", String, false},
	{"min-conf", Float, false}, {"min-ratio", Float, false}, {"auto-threshold", Bool, false},
	{"min-overlap", Float, false}, {"max-offset", Float, false}, {"audio-mode", String, false},
	{"file-order", Bool, false}, {"intra-lane", Bool, false}, {"jobs", Int, false},
	{"threads", Int, false}, {"mem-limit", Float, false}, {"cache", String, true},
	{"originals", List, true}, {"record-time", Bool, false}, {"time-label", String, false},
	{"max-gap", Float, false}, {"keep-gap", Float, false}, {"progress", String, false},
}

func Lookup(name string) (Key, bool) {
	name = Norm(name)
	for _, k := range Keys {
		if k.Name == name {
			return k, true
		}
	}
	return Key{}, false
}

func Norm(name string) string { return strings.ReplaceAll(strings.TrimSpace(name), "_", "-") }

type Error struct {
	Path string
	Key  string
	Msg  string
}

func (e *Error) Error() string {
	if e.Key == "" {
		return e.Path + ": " + e.Msg
	}
	return fmt.Sprintf("%s: [%s] %s: %s", e.Path, Section, e.Key, e.Msg)
}

type Layer struct {
	Path string
	Keys []string
	Vals map[string]any
}

func UserPath() string {
	if v, ok := os.LookupEnv(EnvVar); ok {
		if v = strings.TrimSpace(v); v == "" || strings.EqualFold(v, "none") || v == "-" {
			return ""
		}
		return v
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "orcaton", FileName)
}

func ProjectPath(project string) string {
	if project == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(project), FileName)
}

func Read(path string) (*Layer, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc, err := toml.Parse(string(data))
	if err != nil {
		return nil, &Error{Path: path, Msg: err.Error()}
	}
	l := &Layer{Path: path, Vals: map[string]any{}}
	for _, k := range doc.Keys {
		if _, ok := doc.M[k].(*toml.Table); !ok {
			if _, arr := doc.M[k].([]*toml.Table); !arr {
				return nil, &Error{Path: path, Key: k, Msg: "ключ вне раздела [" + Section + "]"}
			}
		}
	}
	sec, _ := doc.M[Section].(*toml.Table)
	if sec == nil {
		return l, nil
	}
	for _, raw := range sec.Keys {
		key, ok := Lookup(raw)
		if !ok {
			return nil, &Error{Path: path, Key: raw, Msg: "неизвестный ключ (список — orcaton config --init)"}
		}
		if _, dup := l.Vals[key.Name]; dup {
			return nil, &Error{Path: path, Key: raw, Msg: "ключ задан дважды"}
		}
		v, err := convert(key, sec.M[raw])
		if err != nil {
			return nil, &Error{Path: path, Key: raw, Msg: err.Error()}
		}
		if key.Path {
			v = resolve(key, v, filepath.Dir(path))
		}
		l.Keys = append(l.Keys, key.Name)
		l.Vals[key.Name] = v
	}
	return l, nil
}

func convert(k Key, v any) (any, error) {
	switch k.Kind {
	case Bool:
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("ожидается true или false")
	case Int:
		if n, ok := v.(toml.Num); ok && n.IsInt {
			return int(n.V), nil
		}
		return nil, fmt.Errorf("ожидается целое число")
	case Float:
		if n, ok := v.(toml.Num); ok {
			return n.V, nil
		}
		return nil, fmt.Errorf("ожидается число")
	case String:
		if s, ok := v.(string); ok {
			return s, nil
		}
		return nil, fmt.Errorf("ожидается строка в кавычках")
	}
	switch x := v.(type) {
	case string:
		return []string{x}, nil
	case []any:
		out := []string{}
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("ожидается список строк")
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, fmt.Errorf("ожидается список строк: [\"a\", \"b\"]")
}

func resolve(k Key, v any, dir string) any {
	abs := func(p string) string {
		t := strings.TrimSpace(p)
		if t == "" || filepath.IsAbs(t) || isWinAbs(t) {
			return p
		}
		return filepath.Join(dir, filepath.FromSlash(t))
	}
	switch x := v.(type) {
	case string:
		if k.Name != "map" {
			return abs(x)
		}
		parts := strings.Split(x, ";")
		for i, p := range parts {
			if !strings.Contains(p, "=") {
				parts[i] = abs(p)
			}
		}
		return strings.Join(parts, ";")
	case []string:
		out := make([]string, len(x))
		for i, p := range x {
			out[i] = abs(p)
		}
		return out
	}
	return v
}

func isWinAbs(p string) bool {
	return len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') ||
		strings.HasPrefix(p, `\\`)
}

type Settings struct {
	Layers []*Layer
}

func Load(project, explicit string) (Settings, error) {
	var s Settings
	paths := []string{UserPath()}
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return s, err
		}
		paths = append(paths, explicit)
	} else {
		paths = append(paths, ProjectPath(project))
	}
	for i, p := range paths {
		if p == "" || i > 0 && sameFile(p, paths[0]) {
			continue
		}
		l, err := Read(p)
		if err != nil {
			return s, err
		}
		if l != nil {
			s.Layers = append(s.Layers, l)
		}
	}
	return s, nil
}

func sameFile(a, b string) bool {
	if b == "" {
		return false
	}
	sa, ea := os.Stat(a)
	sb, eb := os.Stat(b)
	return ea == nil && eb == nil && os.SameFile(sa, sb)
}

func (s Settings) Get(name string) (any, *Layer) {
	for i := len(s.Layers) - 1; i >= 0; i-- {
		if v, ok := s.Layers[i].Vals[name]; ok {
			return v, s.Layers[i]
		}
	}
	return nil, nil
}

func (s Settings) Names() []string {
	var out []string
	for _, k := range Keys {
		if v, _ := s.Get(k.Name); v != nil {
			out = append(out, k.Name)
		}
	}
	return out
}

func (s Settings) Files() []string {
	var out []string
	for _, l := range s.Layers {
		out = append(out, l.Path)
	}
	return out
}
