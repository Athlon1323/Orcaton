package dsp

import (
	"math"
	"sync"
)

type stage struct {
	r, m, s int
	tw      []complex128
}

type plan struct {
	n      int
	stages []stage
}

type rplan struct {
	n    int
	half *plan
	w    []complex128
}

var (
	planMu    sync.Mutex
	plans     = map[int]*plan{}
	rplans    = map[int]*rplan{}
	planElems = 0
)

const planBudget = 1 << 23

func cis(a float64) complex128 { return complex(math.Cos(a), math.Sin(a)) }

func newPlan(n int) *plan {
	p := &plan{n: n}
	ln, s := n, 1
	for ln > 1 {
		var r int
		switch {
		case ln%4 == 0:
			r = 4
		case ln%2 == 0:
			r = 2
		case ln%3 == 0:
			r = 3
		case ln%5 == 0:
			r = 5
		default:
			r = ln
			for f := 7; f*f <= ln; f += 2 {
				if ln%f == 0 {
					r = f
					break
				}
			}
		}
		m := ln / r
		tw := make([]complex128, m*(r-1))
		for q := 0; q < m; q++ {
			for j := 1; j < r; j++ {
				tw[q*(r-1)+j-1] = cis(-2 * math.Pi * float64(j*q) / float64(ln))
			}
		}
		p.stages = append(p.stages, stage{r, m, s, tw})
		ln = m
		s *= r
	}
	return p
}

func cachePut(elems int) {
	planElems += elems
	if planElems > planBudget {
		plans = map[int]*plan{}
		rplans = map[int]*rplan{}
		planElems = elems
	}
}

func getPlan(n int) *plan {
	planMu.Lock()
	defer planMu.Unlock()
	if p, ok := plans[n]; ok {
		return p
	}
	p := newPlan(n)
	cachePut(n)
	plans[n] = p
	return p
}

func getRPlan(n int) *rplan {
	planMu.Lock()
	if p, ok := rplans[n]; ok {
		planMu.Unlock()
		return p
	}
	planMu.Unlock()
	h := getPlan(n / 2)
	w := make([]complex128, n/2+1)
	for k := range w {
		w[k] = cis(-2 * math.Pi * float64(k) / float64(n))
	}
	p := &rplan{n: n, half: h, w: w}
	planMu.Lock()
	cachePut(len(w))
	rplans[n] = p
	planMu.Unlock()
	return p
}

var bufPool = sync.Pool{}

func getBuf(n int) []complex128 {
	if v := bufPool.Get(); v != nil {
		b := v.([]complex128)
		if cap(b) >= n {
			return b[:n]
		}
	}
	return make([]complex128, n)
}

func putBuf(b []complex128) { bufPool.Put(b[:0]) }

const (
	sin60 = 0.86602540378443864676
)

var (
	c51 = math.Cos(2 * math.Pi / 5)
	c52 = math.Cos(4 * math.Pi / 5)
	s51 = math.Sin(2 * math.Pi / 5)
	s52 = math.Sin(4 * math.Pi / 5)
)

func negI(z complex128) complex128 { return complex(imag(z), -real(z)) }

func (p *plan) transform(x, work []complex128) {
	a, b := x, work
	for _, st := range p.stages {
		r, m, s, tw := st.r, st.m, st.s, st.tw
		switch r {
		case 4:
			for q := 0; q < m; q++ {
				w1, w2, w3 := tw[3*q], tw[3*q+1], tw[3*q+2]
				i0, i1, i2, i3 := s*q, s*(q+m), s*(q+2*m), s*(q+3*m)
				o := s * 4 * q
				for t := 0; t < s; t++ {
					x0, x1, x2, x3 := a[i0+t], a[i1+t], a[i2+t], a[i3+t]
					s02, d02 := x0+x2, x0-x2
					s13, j13 := x1+x3, negI(x1-x3)
					b[o+t] = s02 + s13
					b[o+s+t] = (d02 + j13) * w1
					b[o+2*s+t] = (s02 - s13) * w2
					b[o+3*s+t] = (d02 - j13) * w3
				}
			}
		case 2:
			for q := 0; q < m; q++ {
				w := tw[q]
				i0, i1 := s*q, s*(q+m)
				o := s * 2 * q
				for t := 0; t < s; t++ {
					x0, x1 := a[i0+t], a[i1+t]
					b[o+t] = x0 + x1
					b[o+s+t] = (x0 - x1) * w
				}
			}
		case 3:
			for q := 0; q < m; q++ {
				w1, w2 := tw[2*q], tw[2*q+1]
				i0, i1, i2 := s*q, s*(q+m), s*(q+2*m)
				o := s * 3 * q
				for t := 0; t < s; t++ {
					x0, x1, x2 := a[i0+t], a[i1+t], a[i2+t]
					sm, d := x1+x2, x1-x2
					tt := x0 - sm*0.5
					u := negI(d) * sin60
					b[o+t] = x0 + sm
					b[o+s+t] = (tt + u) * w1
					b[o+2*s+t] = (tt - u) * w2
				}
			}
		case 5:
			for q := 0; q < m; q++ {
				w1, w2, w3, w4 := tw[4*q], tw[4*q+1], tw[4*q+2], tw[4*q+3]
				i0, i1, i2, i3, i4 := s*q, s*(q+m), s*(q+2*m), s*(q+3*m), s*(q+4*m)
				o := s * 5 * q
				for t := 0; t < s; t++ {
					x0, x1, x2, x3, x4 := a[i0+t], a[i1+t], a[i2+t], a[i3+t], a[i4+t]
					s14, d14 := x1+x4, x1-x4
					s23, d23 := x2+x3, x2-x3
					r1 := x0 + s14*complex(c51, 0) + s23*complex(c52, 0)
					r2 := x0 + s14*complex(c52, 0) + s23*complex(c51, 0)
					j1 := negI(d14*complex(s51, 0) + d23*complex(s52, 0))
					j2 := negI(d14*complex(s52, 0) - d23*complex(s51, 0))
					b[o+t] = x0 + s14 + s23
					b[o+s+t] = (r1 + j1) * w1
					b[o+2*s+t] = (r2 + j2) * w2
					b[o+3*s+t] = (r2 - j2) * w3
					b[o+4*s+t] = (r1 - j1) * w4
				}
			}
		default:
			wr := make([]complex128, r)
			for k := range wr {
				wr[k] = cis(-2 * math.Pi * float64(k) / float64(r))
			}
			xs := make([]complex128, r)
			for q := 0; q < m; q++ {
				for t := 0; t < s; t++ {
					for k := 0; k < r; k++ {
						xs[k] = a[t+s*(q+k*m)]
					}
					for j := 0; j < r; j++ {
						var acc complex128
						for k := 0; k < r; k++ {
							acc += xs[k] * wr[(j*k)%r]
						}
						if j > 0 {
							acc *= tw[q*(r-1)+j-1]
						}
						b[t+s*(r*q+j)] = acc
					}
				}
			}
		}
		a, b = b, a
	}
	if len(p.stages)%2 == 1 {
		copy(x, a)
	}
}

func FFT(x []complex128) []complex128 {
	n := len(x)
	out := make([]complex128, n)
	copy(out, x)
	if n <= 1 {
		return out
	}
	work := getBuf(n)
	getPlan(n).transform(out, work)
	putBuf(work)
	return out
}

func IFFT(x []complex128) []complex128 {
	n := len(x)
	y := make([]complex128, n)
	for i, v := range x {
		y[i] = complex(real(v), -imag(v))
	}
	if n > 1 {
		work := getBuf(n)
		getPlan(n).transform(y, work)
		putBuf(work)
	}
	inv := 1 / float64(n)
	for i, v := range y {
		y[i] = complex(real(v)*inv, -imag(v)*inv)
	}
	return y
}

func RFFT(x []float64, n int) []complex128 {
	if n%2 != 0 {
		return rfftOdd(x, n)
	}
	return RFFTInto(x, n, make([]complex128, n/2+1))
}

func rfftOdd(x []float64, n int) []complex128 {
	c := make([]complex128, n)
	for i := 0; i < n && i < len(x); i++ {
		c[i] = complex(x[i], 0)
	}
	return FFT(c)[:n/2+1]
}

func RFFTInto(x []float64, n int, out []complex128) []complex128 {
	h := n / 2
	rp := getRPlan(n)
	z := getBuf(h)
	for k := 0; k < h; k++ {
		var re, im float64
		if 2*k < len(x) {
			re = x[2*k]
		}
		if 2*k+1 < len(x) {
			im = x[2*k+1]
		}
		z[k] = complex(re, im)
	}
	if h > 1 {
		work := getBuf(h)
		rp.half.transform(z, work)
		putBuf(work)
	}
	out = out[:h+1]
	for k := 0; k <= h; k++ {
		zk := z[k%h]
		zc := z[(h-k)%h]
		zc = complex(real(zc), -imag(zc))
		e := (zk + zc) * 0.5
		o := (zk - zc) * complex(0, -0.5)
		out[k] = e + rp.w[k]*o
	}
	putBuf(z)
	return out
}

func IRFFT(X []complex128, n int) []float64 {
	if n%2 != 0 {
		c := make([]complex128, n)
		for k := 0; k < n; k++ {
			switch {
			case k < len(X):
				c[k] = X[k]
			case n-k < len(X):
				v := X[n-k]
				c[k] = complex(real(v), -imag(v))
			}
		}
		y := IFFT(c)
		out := make([]float64, n)
		for i := range out {
			out[i] = real(y[i])
		}
		return out
	}
	h := n / 2
	rp := getRPlan(n)
	z := getBuf(h)
	for k := 0; k < h; k++ {
		xk := X[k]
		xc := X[h-k]
		xc = complex(real(xc), -imag(xc))
		e := (xk + xc) * 0.5
		w := rp.w[k]
		o := (xk - xc) * 0.5 * complex(real(w), -imag(w))
		v := e + complex(-imag(o), real(o))
		z[k] = complex(real(v), -imag(v))
	}
	if h > 1 {
		work := getBuf(h)
		rp.half.transform(z, work)
		putBuf(work)
	}
	out := make([]float64, n)
	inv := 1 / float64(h)
	for k := 0; k < h; k++ {
		out[2*k] = real(z[k]) * inv
		out[2*k+1] = -imag(z[k]) * inv
	}
	putBuf(z)
	return out
}

func NextFastLen(n int) int {
	if n <= 2 {
		return 2
	}
	best := math.MaxInt64
	for p5 := 1; p5 < 2*n+10; p5 *= 5 {
		for p35 := p5; p35 < 2*n+10; p35 *= 3 {
			v := p35
			for v < n {
				v *= 2
			}
			if v%2 != 0 {
				v *= 2
			}
			if v < best {
				best = v
			}
		}
	}
	return best
}
