package num

import (
	"math"
	"strconv"
	"strings"
)

func Round(x float64, n int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	v, err := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	if err != nil {
		return x
	}
	return v
}

func RoundToInt(x float64) int64 { return int64(math.RoundToEven(x)) }

func Fixed(x float64, n int) string {
	if math.IsInf(x, 1) {
		return "inf"
	}
	if math.IsInf(x, -1) {
		return "-inf"
	}
	if math.IsNaN(x) {
		return "nan"
	}
	return strconv.FormatFloat(x, 'f', n, 64)
}

func Signed(x float64, n int) string {
	s := Fixed(x, n)
	if !strings.HasPrefix(s, "-") {
		s = "+" + s
	}
	return s
}

func Percent(x float64, n int) string { return Fixed(x*100, n) + "%" }

func Repr(x float64) string {
	switch {
	case math.IsInf(x, 1):
		return "inf"
	case math.IsInf(x, -1):
		return "-inf"
	case math.IsNaN(x):
		return "nan"
	}
	if x == 0 {
		if math.Signbit(x) {
			return "-0.0"
		}
		return "0.0"
	}
	e := strconv.FormatFloat(x, 'e', -1, 64)
	neg := strings.HasPrefix(e, "-")
	if neg {
		e = e[1:]
	}
	mant, expS, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expS)
	digits := strings.Replace(mant, ".", "", 1)
	var out string
	if exp < -4 || exp >= 16 {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		}
		es := strconv.Itoa(exp)
		if len(es) < 2 {
			es = "0" + es
		}
		out = m + "e" + sign + es
	} else {
		pt := exp + 1
		switch {
		case pt <= 0:
			out = "0." + strings.Repeat("0", -pt) + digits
		case pt >= len(digits):
			out = digits + strings.Repeat("0", pt-len(digits)) + ".0"
		default:
			out = digits[:pt] + "." + digits[pt:]
		}
	}
	if neg {
		out = "-" + out
	}
	return out
}

func Str(x float64) string { return Repr(x) }
