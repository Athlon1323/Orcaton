package pyio

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"orcaton/internal/num"
)

type OMap struct {
	keys []string
	vals map[string]any
}

func NewOMap() *OMap { return &OMap{vals: map[string]any{}} }

func (m *OMap) Set(k string, v any) *OMap {
	if _, ok := m.vals[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.vals[k] = v
	return m
}

func (m *OMap) Get(k string) (any, bool) {
	v, ok := m.vals[k]
	return v, ok
}

func (m *OMap) Keys() []string { return append([]string(nil), m.keys...) }

func (m *OMap) Len() int { return len(m.keys) }

type Float float64

func Dumps(v any, indent int) string {
	var b strings.Builder
	enc(&b, v, indent, 0)
	return b.String()
}

func enc(b *strings.Builder, v any, indent, level int) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.Itoa(x))
	case int64:
		b.WriteString(strconv.FormatInt(x, 10))
	case Float:
		b.WriteString(floatJSON(float64(x)))
	case float64:
		b.WriteString(floatJSON(x))
	case string:
		b.WriteString(Quote(x))
	case []any:
		encList(b, x, indent, level)
	case []string:
		l := make([]any, len(x))
		for i, s := range x {
			l[i] = s
		}
		encList(b, l, indent, level)
	case *OMap:
		if x == nil {
			b.WriteString("null")
			return
		}
		if len(x.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{")
		for i, k := range x.keys {
			if i > 0 {
				sep(b, indent)
			}
			nl(b, indent, level+1)
			b.WriteString(Quote(k))
			b.WriteString(": ")
			enc(b, x.vals[k], indent, level+1)
		}
		nl(b, indent, level)
		b.WriteString("}")
	case map[string]int:
		m := NewOMap()
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			m.Set(k, x[k])
		}
		enc(b, m, indent, level)
	default:
		b.WriteString(Quote(fmt.Sprint(x)))
	}
}

func encList(b *strings.Builder, x []any, indent, level int) {
	if len(x) == 0 {
		b.WriteString("[]")
		return
	}
	b.WriteString("[")
	for i, e := range x {
		if i > 0 {
			sep(b, indent)
		}
		nl(b, indent, level+1)
		enc(b, e, indent, level+1)
	}
	nl(b, indent, level)
	b.WriteString("]")
}

func sep(b *strings.Builder, indent int) {
	if indent < 0 {
		b.WriteString(", ")
	} else {
		b.WriteString(",")
	}
}

func nl(b *strings.Builder, indent, level int) {
	if indent < 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", indent*level))
}

func floatJSON(x float64) string {
	switch {
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case math.IsNaN(x):
		return "NaN"
	}
	return num.Repr(x)
}

func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
