package dsp

import "math"

func PairwiseSum(a []float64) float64 {
	n := len(a)
	switch {
	case n < 8:
		s := 0.0
		for _, v := range a {
			s += v
		}
		return s
	case n <= 128:
		var r [8]float64
		copy(r[:], a[:8])
		i := 8
		for ; i < n-(n%8); i += 8 {
			r[0] += a[i]
			r[1] += a[i+1]
			r[2] += a[i+2]
			r[3] += a[i+3]
			r[4] += a[i+4]
			r[5] += a[i+5]
			r[6] += a[i+6]
			r[7] += a[i+7]
		}
		res := ((r[0] + r[1]) + (r[2] + r[3])) + ((r[4] + r[5]) + (r[6] + r[7]))
		for ; i < n; i++ {
			res += a[i]
		}
		return res
	default:
		n2 := n / 2
		n2 -= n2 % 8
		return PairwiseSum(a[:n2]) + PairwiseSum(a[n2:])
	}
}

func Mean(a []float64) float64 {
	if len(a) == 0 {
		return math.NaN()
	}
	return PairwiseSum(a) / float64(len(a))
}

func Var(a []float64) float64 {
	if len(a) == 0 {
		return math.NaN()
	}
	m := Mean(a)
	d := make([]float64, len(a))
	for i, v := range a {
		x := v - m
		d[i] = x * x
	}
	return PairwiseSum(d) / float64(len(a))
}

func Std(a []float64) float64 { return math.Sqrt(Var(a)) }

func ToF64(x []float32) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = float64(v)
	}
	return out
}

func ToF32(x []float64) []float32 {
	out := make([]float32, len(x))
	for i, v := range x {
		out[i] = float32(v)
	}
	return out
}
