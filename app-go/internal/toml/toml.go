package toml

import (
	"fmt"
	"strconv"
	"strings"

	"orcaton/internal/num"
)

type Num struct {
	V     float64
	IsInt bool
}

func (n Num) String() string {
	if n.IsInt {
		return strconv.FormatInt(int64(n.V), 10)
	}
	return num.Repr(n.V)
}

type Table struct {
	Keys []string
	M    map[string]any
}

func newTable() *Table { return &Table{M: map[string]any{}} }

func (t *Table) set(k string, v any) {
	if _, ok := t.M[k]; !ok {
		t.Keys = append(t.Keys, k)
	}
	t.M[k] = v
}

func Parse(src string) (*Table, error) {
	root := newTable()
	cur := root
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for ln := 0; ln < len(lines); ln++ {
		line := strings.TrimSpace(StripComment(lines[ln]))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && !strings.HasSuffix(line, "]") ||
			strings.HasPrefix(line, "[[") && !strings.HasSuffix(line, "]]") {
			return nil, fmt.Errorf("line %d: unclosed section header %s", ln+1, line)
		}
		if strings.HasPrefix(line, "[[") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			parent, last, err := walk(root, name, false)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", ln+1, err)
			}
			arr, _ := parent.M[last].([]*Table)
			t := newTable()
			parent.set(last, append(arr, t))
			cur = t
			continue
		}
		if strings.HasPrefix(line, "[") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			parent, last, err := walk(root, name, false)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", ln+1, err)
			}
			t, ok := parent.M[last].(*Table)
			if !ok {
				t = newTable()
				parent.set(last, t)
			}
			cur = t
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			return nil, fmt.Errorf("line %d: expected key = value", ln+1)
		}
		key := unquoteKey(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])
		for strings.HasPrefix(val, "[") && !Balanced(val) && ln+1 < len(lines) {
			ln++
			val += " " + strings.TrimSpace(StripComment(lines[ln]))
		}
		v, rest, err := parseValue(val)
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", ln+1, err)
		}
		if strings.TrimSpace(rest) != "" {
			return nil, fmt.Errorf("line %d: extra text after the value: %q", ln+1, rest)
		}
		if _, dup := cur.M[key]; dup {
			return nil, fmt.Errorf("line %d: key %s set twice", ln+1, key)
		}
		cur.set(key, v)
	}
	return root, nil
}

func walk(root *Table, name string, _ bool) (*Table, string, error) {
	parts := strings.Split(name, ".")
	t := root
	for _, p := range parts[:len(parts)-1] {
		p = unquoteKey(strings.TrimSpace(p))
		switch x := t.M[p].(type) {
		case *Table:
			t = x
		case []*Table:
			if len(x) == 0 {
				return nil, "", fmt.Errorf("empty array of tables %s", p)
			}
			t = x[len(x)-1]
		case nil:
			nt := newTable()
			t.set(p, nt)
			t = nt
		default:
			return nil, "", fmt.Errorf("%s is not a table", p)
		}
	}
	return t, unquoteKey(strings.TrimSpace(parts[len(parts)-1])), nil
}

func unquoteKey(k string) string {
	if len(k) >= 2 && (k[0] == '"' || k[0] == '\'') && k[len(k)-1] == k[0] {
		return k[1 : len(k)-1]
	}
	return k
}

func StripComment(s string) string {
	inS, inD := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inD:
			i++
		case c == '"' && !inS:
			inD = !inD
		case c == '\'' && !inD:
			inS = !inS
		case c == '#' && !inS && !inD:
			return s[:i]
		}
	}
	return s
}

func Balanced(s string) bool {
	depth := 0
	inS, inD := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inD:
			i++
		case c == '"' && !inS:
			inD = !inD
		case c == '\'' && !inD:
			inS = !inS
		case !inS && !inD && c == '[':
			depth++
		case !inS && !inD && c == ']':
			depth--
		}
	}
	return depth <= 0
}

func parseValue(s string) (any, string, error) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return nil, "", fmt.Errorf("empty value")
	}
	switch s[0] {
	case '"':
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			c := s[i]
			if c == '\\' && i+1 < len(s) {
				i++
				switch s[i] {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '\\':
					b.WriteByte('\\')
				case '"':
					b.WriteByte('"')
				case 'u', 'U':
					n := 4
					if s[i] == 'U' {
						n = 8
					}
					if i+n < len(s) {
						r, err := strconv.ParseUint(s[i+1:i+1+n], 16, 32)
						if err == nil {
							b.WriteRune(rune(r))
							i += n
						}
					}
				default:
					b.WriteByte('\\')
					b.WriteByte(s[i])
				}
				continue
			}
			if c == '"' {
				return b.String(), s[i+1:], nil
			}
			b.WriteByte(c)
		}
		return nil, "", fmt.Errorf("unclosed string")
	case '\'':
		j := strings.IndexByte(s[1:], '\'')
		if j < 0 {
			return nil, "", fmt.Errorf("unclosed string")
		}
		return s[1 : 1+j], s[2+j:], nil
	case '[':
		var arr []any
		rest := strings.TrimLeft(s[1:], " \t")
		for {
			rest = strings.TrimLeft(rest, " \t,")
			if strings.HasPrefix(rest, "]") {
				return arr, rest[1:], nil
			}
			if rest == "" {
				return nil, "", fmt.Errorf("unclosed array")
			}
			v, r, err := parseValue(rest)
			if err != nil {
				return nil, "", err
			}
			arr = append(arr, v)
			rest = strings.TrimLeft(r, " \t")
			if strings.HasPrefix(rest, ",") {
				rest = rest[1:]
			}
		}
	}
	end := strings.IndexAny(s, ",] \t")
	tok, rest := s, ""
	if end >= 0 {
		tok, rest = s[:end], s[end:]
	}
	switch tok {
	case "true":
		return true, rest, nil
	case "false":
		return false, rest, nil
	}
	clean := strings.ReplaceAll(tok, "_", "")
	if i, err := strconv.ParseInt(clean, 10, 64); err == nil {
		return Num{float64(i), true}, rest, nil
	}
	if f, err := strconv.ParseFloat(clean, 64); err == nil {
		return Num{f, false}, rest, nil
	}
	return nil, "", fmt.Errorf("cannot understand value %q", tok)
}
