package core

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"orcaton/internal/num"
)

var ntsc = num.F(1000, 1001)

type Rate struct{ FPS num.Q }

func RateQ(q num.Q) Rate { return Rate{FPS: num.Copy(q)} }

func RateInt(n int64) Rate { return Rate{FPS: num.I(n)} }

func RateFloat(v float64) Rate {
	for _, base := range []int64{24, 30, 48, 60, 120} {
		if math.Abs(v-float64(base)*1000/1001) < 0.005 {
			return Rate{FPS: num.Mul(num.I(base), ntsc)}
		}
	}
	return Rate{FPS: num.FloatLimit(v, 1001)}
}

func RateString(s string) (Rate, error) {
	s = strings.TrimSpace(s)
	if n, d, ok := strings.Cut(s, "/"); ok {
		ni, e1 := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		di, e2 := strconv.ParseInt(strings.TrimSpace(d), 10, 64)
		if e1 != nil || e2 != nil || di == 0 {
			return Rate{}, fmt.Errorf("некорректная частота %q", s)
		}
		if r := (Rate{FPS: num.F(ni, di)}); r.Valid() {
			return r, nil
		}
		return Rate{}, fmt.Errorf("некорректная частота %q", s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || !ValidFloat(f) {
		return Rate{}, fmt.Errorf("некорректная частота %q", s)
	}
	return RateFloat(f), nil
}

func RateFromXmeml(timebase string, ntscFlag *string) (Rate, error) {
	isNTSC := ntscFlag != nil && strings.ToUpper(strings.TrimSpace(*ntscFlag)) == "TRUE"
	tb, ok := num.ParseFraction(strings.TrimSpace(timebase))
	if !ok {
		return Rate{}, fmt.Errorf("некорректный timebase %q", timebase)
	}
	if isNTSC {
		return Rate{FPS: num.Mul(tb, ntsc)}, nil
	}
	return Rate{FPS: tb}, nil
}

const MaxFPS = 1000

func (r Rate) Valid() bool {
	return r.FPS != nil && r.FPS.Sign() > 0 && num.Le(r.FPS, num.I(MaxFPS)) && r.Timebase() >= 1
}

func ValidFloat(fps float64) bool { return fps >= 0.5 && fps <= MaxFPS }

func (r Rate) IsNTSC() bool { return r.FPS.Denom().Cmp(big.NewInt(1001)) == 0 }

func (r Rate) Timebase() int64 { return num.RoundInt(r.FPS) }

func (r Rate) FramesToSeconds(frames num.Q) num.Q { return num.Div(frames, r.FPS) }

func (r Rate) SecondsToFrames(sec num.Q) num.Q { return num.Mul(sec, r.FPS) }

func (r Rate) Float() float64 { return num.Float(r.FPS) }

func (r Rate) String() string {
	if r.FPS.IsInt() {
		return r.FPS.Num().String()
	}
	s := strconv.FormatFloat(num.Float(r.FPS), 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func (r Rate) Equal(o Rate) bool { return num.Eq(r.FPS, o.FPS) }

func dropFramesPerMin(r Rate) int64 { return 2 * (r.Timebase() / 30) }

func TCToFrames(tc string, r Rate, dropFrame *bool) (int64, error) {
	if !r.Valid() {
		return 0, fmt.Errorf("таймкод %q: некорректная частота", tc)
	}
	df := strings.Contains(tc, ";")
	if dropFrame != nil {
		df = *dropFrame
	}
	t := strings.ReplaceAll(strings.ReplaceAll(tc, ";", ":"), ".", ":")
	parts := strings.Split(t, ":")
	if len(parts) != 4 {
		return 0, fmt.Errorf("Некорректный таймкод: %q", tc)
	}
	var v [4]int64
	for i, p := range parts {
		n, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Некорректный таймкод: %q", tc)
		}
		v[i] = n
	}
	tb := r.Timebase()
	total := ((v[0]*60+v[1])*60+v[2])*tb + v[3]
	if df && r.IsNTSC() {
		d := dropFramesPerMin(r)
		totalMin := v[0]*60 + v[1]
		total -= d * (totalMin - pyFloorDiv(totalMin, 10))
	}
	return total, nil
}

func FramesToTC(frames int64, r Rate, dropFrame bool) string {
	if !r.Valid() {
		return "--:--:--:--"
	}
	tb := r.Timebase()
	sign := ""
	if frames < 0 {
		sign = "-"
		frames = -frames
	}
	if dropFrame && r.IsNTSC() {
		d := dropFramesPerMin(r)
		per10 := tb*600 - d*9
		perMin := tb*60 - d
		tens, rem := frames/per10, frames%per10
		if rem > d {
			frames += d*9*tens + d*((rem-d)/perMin)
		} else {
			frames += d * 9 * tens
		}
	}
	ff := frames % tb
	ss := (frames / tb) % 60
	mm := (frames / (tb * 60)) % 60
	hh := frames / (tb * 3600)
	sep := ":"
	if dropFrame && r.IsNTSC() {
		sep = ";"
	}
	return fmt.Sprintf("%s%02d:%02d:%02d%s%02d", sign, hh, mm, ss, sep, ff)
}

func pyFloorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
