package cli

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/media"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
)

func tcOf(seq *core.Sequence, sec float64) string {
	fr := math.RoundToEven((sec + num.Float(seq.TCStart)) * seq.Rate.Float())
	return core.FramesToTC(int64(fr), seq.Rate, false)
}

type counter struct {
	keys []string
	n    map[string]int
}

func newCounter() *counter { return &counter{n: map[string]int{}} }

func (k *counter) inc(s string) {
	if _, ok := k.n[s]; !ok {
		k.keys = append(k.keys, s)
	}
	k.n[s]++
}

func (k *counter) String() string {
	keys := append([]string(nil), k.keys...)
	sort.SliceStable(keys, func(i, j int) bool { return k.n[keys[i]] > k.n[keys[j]] })
	var parts []string
	for _, s := range keys {
		parts = append(parts, fmt.Sprintf("%s×%d", s, k.n[s]))
	}
	return strings.Join(parts, ", ")
}

func intOrNone(p *int) string {
	if p == nil {
		return "None"
	}
	return fmt.Sprint(*p)
}

func splitList(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func sf(m *pyio.OMap, k string) float64 {
	v, _ := m.Get(k)
	switch x := v.(type) {
	case int:
		return float64(x)
	case pyio.Float:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

func sv(m *pyio.OMap, k string) string {
	v, _ := m.Get(k)
	switch x := v.(type) {
	case int:
		return fmt.Sprint(x)
	case pyio.Float:
		return num.Repr(float64(x))
	}
	return fmt.Sprint(v)
}

func anyList(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

func ffmpegFail(c *ctx, err error) int {
	var nf *media.FFmpegNotFound
	if errors.As(err, &nf) {
		c.println(err.Error())
		return 3
	}
	if app.KindOf(err) == app.KindCancelled {
		c.println("Cancelled")
		return 130
	}
	c.printf("Error: %v\n", err)
	return 1
}

func pathStr(p string) string { return filepath.Clean(p) }

func f1(m *pyio.OMap, k string) string {
	v, _ := m.Get(k)
	return num.Repr(float64(v.(pyio.Float)))
}

func pyValOf(m *pyio.OMap, k string) string {
	v, _ := m.Get(k)
	return pyVal(v)
}

func pyVal(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case pyio.Float:
		return num.Repr(float64(x))
	}
	return fmt.Sprint(v)
}
