package engine

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"orcaton/internal/formats"
)

var (
	goproChapter = regexp.MustCompile(`^g[hxlp](\d{2})(\d{4})$`)
	goproFirst   = regexp.MustCompile(`^gopr(\d{4})$`)
	tailNumber   = regexp.MustCompile(`^(.*?)(\d+)$`)
)

func SeriesOf(name string) (string, float64, bool) {
	p, n, _, ok := seriesOf(name)
	return p, n, ok
}

func seriesOf(name string) (string, float64, int, bool) {
	stem := formats.Casefold(strings.TrimSuffix(name, filepath.Ext(name)))
	ext := formats.Casefold(filepath.Ext(name))
	if m := goproChapter.FindStringSubmatch(stem); m != nil {
		ch, _ := strconv.Atoi(m[1])
		f, _ := strconv.Atoi(m[2])
		return "gopro" + ext, float64(f*100 + ch), 0, true
	}
	if m := goproFirst.FindStringSubmatch(stem); m != nil {
		f, _ := strconv.Atoi(m[1])
		return "gopro" + ext, float64(f * 100), 0, true
	}
	m := tailNumber.FindStringSubmatch(stem)
	if m == nil || len(m[2]) > 15 {
		return "", 0, 0, false
	}
	n, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return "", 0, 0, false
	}
	return m[1] + ext, n, len(m[2]), true
}

func assignSeries(units []*Unit) {
	devs, by := byDevice(units)
	for _, d := range devs {
		lst := by[d]
		prefix, digits, ok := "", -1, true
		seqs := make([]float64, len(lst))
		for i, u := range lst {
			if u.Clips[0].Media == nil {
				ok = false
				break
			}
			p, n, dg, good := seriesOf(u.Name())
			if !good || i > 0 && (p != prefix || dg != digits) {
				ok = false
				break
			}
			prefix, digits, seqs[i] = p, dg, n
		}
		if !ok || len(lst) == 0 {
			continue
		}
		unwrapCounter(lst, seqs, digits)
		for i, u := range lst {
			u.Order.Series, u.Order.Seq = d+"|"+prefix, seqs[i]
		}
	}
}

func unwrapCounter(lst []*Unit, seqs []float64, digits int) {
	if digits <= 0 || len(seqs) < 2 {
		return
	}
	span := math.Pow(10, float64(digits))
	sorted := append([]float64(nil), seqs...)
	sort.Float64s(sorted)
	gap, cut := 0.0, 0.0
	for i := 1; i < len(sorted); i++ {
		if g := sorted[i] - sorted[i-1]; g > gap {
			gap, cut = g, sorted[i-1]
		}
	}
	if gap <= span/2 {
		return
	}
	lowMin, highMax := math.Inf(1), math.Inf(-1)
	for i, u := range lst {
		if u.RecTime == nil {
			return
		}
		if seqs[i] <= cut {
			lowMin = math.Min(lowMin, *u.RecTime)
		} else {
			highMax = math.Max(highMax, *u.RecTime)
		}
	}
	if lowMin > highMax {
		for i := range seqs {
			if seqs[i] <= cut {
				seqs[i] += span
			}
		}
	}
}
