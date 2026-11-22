package settings

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"orcaton/internal/toml"
)

type KV struct {
	Name string
	Val  any
}

func Update(path string, vals []KV) error {
	for _, kv := range vals {
		k, ok := Lookup(kv.Name)
		if !ok {
			return &Error{Path: path, Key: kv.Name, Msg: "unknown key"}
		}
		if kv.Val != nil {
			if _, err := Format(k, kv.Val); err != nil {
				return &Error{Path: path, Key: kv.Name, Msg: err.Error()}
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	nl := "\n"
	if strings.Contains(string(data), "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, kv := range vals {
		k, _ := Lookup(kv.Name)
		lines = setKey(lines, k, kv.Val)
	}
	out := strings.Join(lines, nl) + nl
	if _, err := toml.Parse(out); err != nil {
		return &Error{Path: path, Msg: "the file cannot be read after the edit: " + err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

type span struct{ from, to int }

func section(lines []string) (hdr, end int) {
	hdr, end = -1, len(lines)
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(toml.StripComment(lines[i]))
		if !strings.HasPrefix(t, "[") {
			if j := entryEnd(lines, i); j > i+1 {
				i = j - 1
			}
			continue
		}
		if hdr >= 0 {
			return hdr, i
		}
		if !strings.HasPrefix(t, "[[") && strings.TrimSpace(strings.Trim(t, "[]")) == Section {
			hdr = i
		}
	}
	return hdr, end
}

func entryEnd(lines []string, i int) int {
	t := toml.StripComment(lines[i])
	eq := strings.Index(t, "=")
	if eq < 0 {
		return i + 1
	}
	val := strings.TrimSpace(t[eq+1:])
	j := i + 1
	for strings.HasPrefix(val, "[") && !toml.Balanced(val) && j < len(lines) {
		val += " " + strings.TrimSpace(toml.StripComment(lines[j]))
		j++
	}
	return j
}

func find(lines []string, hdr, end int, name string) (span, bool) {
	for i := hdr + 1; i < end; {
		j := entryEnd(lines, i)
		t := toml.StripComment(lines[i])
		if eq := strings.Index(t, "="); eq > 0 {
			k := strings.Trim(strings.TrimSpace(t[:eq]), `"'`)
			if Norm(k) == name {
				return span{i, j}, true
			}
		}
		i = j
	}
	return span{}, false
}

func setKey(lines []string, k Key, v any) []string {
	hdr, end := section(lines)
	if hdr < 0 {
		if v == nil {
			return lines
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+Section+"]")
		hdr, end = len(lines)-1, len(lines)
	}
	sp, found := find(lines, hdr, end, k.Name)
	var repl []string
	if v != nil {
		val, _ := Format(k, v)
		line := k.Name + " = " + val
		if found && sp.to == sp.from+1 {
			if c := lines[sp.from][len(toml.StripComment(lines[sp.from])):]; c != "" {
				line += "  " + strings.TrimSpace(c)
			}
		}
		repl = []string{line}
	}
	if !found {
		if v == nil {
			return lines
		}
		at := end
		for at > hdr+1 && strings.TrimSpace(lines[at-1]) == "" {
			at--
		}
		sp = span{at, at}
	}
	out := append([]string{}, lines[:sp.from]...)
	out = append(out, repl...)
	return append(out, lines[sp.to:]...)
}

func Format(k Key, v any) (string, error) {
	switch k.Kind {
	case Bool:
		if b, ok := v.(bool); ok {
			return strconv.FormatBool(b), nil
		}
	case Int:
		if n, ok := v.(int); ok {
			return strconv.Itoa(n), nil
		}
	case Float:
		if f, ok := v.(float64); ok {
			s := strconv.FormatFloat(f, 'g', -1, 64)
			if !strings.ContainsAny(s, ".eEn") {
				s += ".0"
			}
			return s, nil
		}
	case String:
		if s, ok := v.(string); ok {
			return Quote(s), nil
		}
	case List:
		if l, ok := v.([]string); ok {
			q := make([]string, len(l))
			for i, s := range l {
				q[i] = Quote(s)
			}
			return "[" + strings.Join(q, ", ") + "]", nil
		}
	}
	return "", fmt.Errorf("value %v has the wrong type", v)
}

func Quote(s string) string {
	if !strings.ContainsAny(s, "'\n\r\t") {
		return "'" + s + "'"
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}
