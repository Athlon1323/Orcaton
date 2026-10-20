package dsp

import (
	"math"
	"sync"
)

type OffsetResult struct {
	OffsetSec  float64
	Confidence float64
	Peak       float64
	Ratio      float64
	OverlapSec float64
}

type Prepared struct {
	X    []float64
	Size int
	C1   []float64
	C2   []float64
	Var  float64
	mu   sync.Mutex
	spec map[specKey][]complex128
}

type specKey struct {
	n   int
	rev bool
}

func NewPrepared(f []float32) *Prepared {
	x := ToF64(f)
	p := &Prepared{X: x, Size: len(x), spec: map[specKey][]complex128{}}
	p.C1 = make([]float64, len(x)+1)
	p.C2 = make([]float64, len(x)+1)
	for i, v := range x {
		p.C1[i+1] = p.C1[i] + v
		p.C2[i+1] = p.C2[i] + v*v
	}
	p.Var = Var(x)
	if p.Var == 0 || math.IsNaN(p.Var) {
		p.Var = 1.0
	}
	return p
}

func (p *Prepared) DropCache() {
	p.mu.Lock()
	p.spec = map[specKey][]complex128{}
	p.mu.Unlock()
}

func (p *Prepared) Spectrum(n int, reverse, keep bool) []complex128 {
	k := specKey{n, reverse}
	p.mu.Lock()
	if sp, ok := p.spec[k]; ok {
		p.mu.Unlock()
		return sp
	}
	p.mu.Unlock()
	x := p.X
	if reverse {
		x = make([]float64, len(p.X))
		for i, v := range p.X {
			x[len(p.X)-1-i] = v
		}
	}
	sp := RFFT(x, n)
	if !keep {
		return sp
	}
	p.mu.Lock()
	if len(p.spec) > 4 {
		p.spec = map[specKey][]complex128{}
	}
	p.spec[k] = sp
	p.mu.Unlock()
	return sp
}

func window(na, nb, lag int) (a0, a1, b0, b1 int, n float64) {
	a0 = max(lag, 0)
	a1 = min(lag+nb, na)
	b0, b1 = a0-lag, a1-lag
	return a0, a1, b0, b1, float64(max(a1-a0, 1))
}

func pearson(pa, pb *Prepared, lag int, sab float64, minOv int, thrA, thrB float64) (
	rv, n float64, ok bool) {
	a0, a1, b0, b1, n := window(pa.Size, pb.Size, lag)
	sa := pa.C1[a1] - pa.C1[a0]
	sb := pb.C1[b1] - pb.C1[b0]
	va := (pa.C2[a1] - pa.C2[a0]) - sa*sa/n
	vb := (pb.C2[b1] - pb.C2[b0]) - sb*sb/n
	if !(n >= float64(minOv) && va >= thrA*n && vb >= thrB*n) {
		return 0, n, false
	}
	return (sab - sa*sb/n) / math.Sqrt(math.Max(va*vb, 1e-12)), n, true
}

func scores(pa, pb *Prepared, lag0 int, sab []float64, minOv int, minActivity float64) []float64 {
	z := make([]float64, len(sab))
	thrA := minActivity * pa.Var
	thrB := minActivity * pb.Var
	for k := range sab {
		z[k] = math.Inf(-1)
		if rv, n, ok := pearson(pa, pb, lag0+k, sab[k], minOv, thrA, thrB); ok {
			z[k] = math.Atanh(clip(rv, -0.9999, 0.9999)) * math.Sqrt(math.Max(n-3, 1))
		}
	}
	return z
}

func peakResult(pa, pb *Prepared, lag0 int, sab, z []float64, fps float64, minOv int,
	minActivity float64, withRatio bool) *OffsetResult {
	i := argmax(z)
	if !finite(z[i]) {
		return nil
	}
	rv, n, _ := pearson(pa, pb, lag0+i, sab[i], minOv, minActivity*pa.Var, minActivity*pb.Var)
	ratio := 0.0
	if withRatio {
		ratio = ratioAround(z, i, fps)
	}
	return &OffsetResult{
		OffsetSec:  (float64(lag0+i) + peak(z, i)) / fps,
		Confidence: z[i],
		Peak:       rv,
		Ratio:      ratio,
		OverlapSec: n / fps,
	}
}

func clip(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func argmax(z []float64) int {
	best := 0
	for i := 1; i < len(z); i++ {
		if z[i] > z[best] || (math.IsNaN(z[best]) && !math.IsNaN(z[i])) {
			best = i
		}
	}
	return best
}

func finite(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) }

func peak(z []float64, i int) float64 {
	if 0 < i && i < len(z)-1 && finite(z[i-1]) && finite(z[i+1]) {
		y0, y1, y2 := z[i-1], z[i], z[i+1]
		den := y0 - 2*y1 + y2
		if den != 0 {
			return clip(0.5*(y0-y2)/den, -0.5, 0.5)
		}
	}
	return 0
}

func ratioAround(z []float64, i int, fps float64) float64 { return RatioAround(z, i, int(fps)) }

func RatioAround(z []float64, i int, win int) float64 {
	if win < 1 {
		win = 1
	}
	zi := z[i]
	lo, hi := i-win, i+win+1
	if lo < 0 {
		lo = 0
	}
	if hi > len(z) {
		hi = len(z)
	}
	second := math.Inf(-1)
	any := false
	for k, v := range z {
		if k >= lo && k < hi {
			continue
		}
		if finite(v) {
			any = true
			if v > second {
				second = v
			}
		}
	}
	if !any {
		second = 0
	}
	ratio := 99.0
	if second > 0 {
		ratio = zi / second
	}
	return math.Min(ratio, 99.0)
}

func crossDots(pa, pb *Prepared, nfft int, keepA bool) []float64 {
	m := pa.Size + pb.Size - 1
	A := pa.Spectrum(nfft, false, keepA)
	B := pb.Spectrum(nfft, true, false)
	prod := getBuf(len(A))
	for i := range A {
		prod[i] = A[i] * B[i]
	}
	out := IRFFT(prod, nfft)[:m]
	putBuf(prod)
	return out
}

func minOverlap(sec, fps float64) int {
	v := int(sec * fps)
	if v < 4 {
		v = 4
	}
	return v
}

func XcorrPrepared(pa, pb *Prepared, fps, minOverlapSec, maxOffsetSec, minActivity float64,
	nfft int) *OffsetResult {
	na, nb := pa.Size, pb.Size
	minOv := minOverlap(minOverlapSec, fps)
	if na < minOv || nb < minOv {
		return nil
	}
	m := na + nb - 1
	nf := nfft
	if nf < m || nf%2 != 0 {
		nf = NextFastLen(m)
	}
	sab := crossDots(pa, pb, nf, nfft > 0)
	k0, k1 := 0, m
	if maxOffsetSec >= 0 {
		far := func(k int) bool { return math.Abs(float64(k-(nb-1))) > maxOffsetSec*fps }
		for k0 < m && far(k0) {
			k0++
		}
		for k1 > k0 && far(k1-1) {
			k1--
		}
	}
	if k0 >= k1 {
		return nil
	}
	lag0 := k0 - (nb - 1)
	z := scores(pa, pb, lag0, sab[k0:k1], minOv, minActivity)
	return peakResult(pa, pb, lag0, sab[k0:k1], z, fps, minOv, minActivity, true)
}

func RefinePrepared(pa, pb *Prepared, fps, centerSec, radiusSec, minOverlapSec,
	minActivity float64) *OffsetResult {
	na, nb := pa.Size, pb.Size
	minOv := minOverlap(minOverlapSec, fps)
	c := int(math.RoundToEven(centerSec * fps))
	rad := int(math.RoundToEven(radiusSec * fps))
	if rad < 1 {
		rad = 1
	}
	lo, hi := c-rad, c+rad
	if lo < -(nb - 1) {
		lo = -(nb - 1)
	}
	if hi > na-1 {
		hi = na - 1
	}
	if lo > hi {
		return nil
	}
	sab := make([]float64, 0, hi-lo+1)
	for lag := lo; lag <= hi; lag++ {
		a0, a1, b0, b1, _ := window(na, nb, lag)
		s := 0.0
		if a1 > a0 {
			s = dot(pa.X[a0:a1], pb.X[b0:b1])
		}
		sab = append(sab, s)
	}
	z := scores(pa, pb, lo, sab, minOv, minActivity)
	return peakResult(pa, pb, lo, sab, z, fps, minOv, minActivity, false)
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func XcorrWindow(pa, pb *Prepared, fps, loSec, hiSec, minOverlapSec,
	minActivity float64) *OffsetResult {
	na, nb := pa.Size, pb.Size
	var lo, hi int
	if loSec*fps <= float64(-(nb - 1)) {
		lo = -(nb - 1)
	} else {
		lo = int(math.Ceil(loSec * fps))
	}
	if hiSec*fps >= float64(na-1) {
		hi = na - 1
	} else {
		hi = int(math.Floor(hiSec * fps))
	}
	if lo > hi {
		return nil
	}
	sab := windowDots(pa, pb, lo, hi)
	minOv := minOverlap(minOverlapSec, fps)
	z := scores(pa, pb, lo, sab, minOv, minActivity)
	return peakResult(pa, pb, lo, sab, z, fps, minOv, minActivity, true)
}

func WindowScores(pa, pb *Prepared, fps, loSec, hiSec, minOverlapSec, minActivity float64) (int, []float64) {
	na, nb := pa.Size, pb.Size
	lo, hi := -(nb - 1), na-1
	if loSec*fps > float64(lo) {
		lo = int(math.Ceil(loSec * fps))
	}
	if hiSec*fps < float64(hi) {
		hi = int(math.Floor(hiSec * fps))
	}
	if lo > hi {
		return 0, nil
	}
	sab := windowDots(pa, pb, lo, hi)
	return lo, scores(pa, pb, lo, sab, minOverlap(minOverlapSec, fps), minActivity)
}

func ArgMax(z []float64) (int, float64) {
	i := argmax(z)
	if len(z) == 0 || !finite(z[i]) {
		return 0, math.Inf(-1)
	}
	return i, z[i]
}

func windowDots(pa, pb *Prepared, lo, hi int) []float64 {
	na, nb := pa.Size, pb.Size
	s0, s1 := max(lo, 0), min(hi+nb, na)
	if ns := s1 - s0; ns > 0 && 2*(ns+nb) < na+nb {
		nf := NextFastLen(ns + nb - 1)
		A := RFFT(pa.X[s0:s1], nf)
		B := pb.Spectrum(nf, true, false)
		prod := getBuf(len(A))
		for i := range A {
			prod[i] = A[i] * B[i]
		}
		all := IRFFT(prod, nf)
		putBuf(prod)
		return append([]float64(nil), all[lo-s0+nb-1:hi-s0+nb]...)
	}
	all := crossDots(pa, pb, NextFastLen(na+nb-1), false)
	return all[lo+(nb-1) : hi+(nb-1)+1]
}

func PoolFeature(f []float32, factor int) []float32 {
	if factor <= 1 {
		return append([]float32(nil), f...)
	}
	n := len(f) / factor
	if n < 1 {
		return []float32{0}
	}
	g := make([]float64, n)
	for i := 0; i < n; i++ {
		mx := f[i*factor]
		for _, v := range f[i*factor+1 : (i+1)*factor] {
			if v > mx || isNaN32(v) {
				mx = v
			}
		}
		g[i] = float64(mx)
	}
	m := Mean(g)
	for i := range g {
		g[i] -= m
	}
	sd := Std(g)
	if sd > 0 {
		for i := range g {
			g[i] /= sd
		}
	}
	return ToF32(g)
}

func isNaN32(v float32) bool { return v != v }

func XcorrFeatures(fa, fb []float32, fps, minOverlapSec, maxOffsetSec,
	minActivity float64) *OffsetResult {
	return XcorrPrepared(NewPrepared(fa), NewPrepared(fb), fps, minOverlapSec, maxOffsetSec,
		minActivity, 0)
}

func FindOffset(a, b []float32, sr, hop int, maxOffsetSec, minOverlapSec float64) OffsetResult {
	fps := float64(sr) / float64(hop)
	fa := MaskOnsetArtifacts(OnsetFeature(a, sr, hop), fps, 0.15, 0.05)
	fb := MaskOnsetArtifacts(OnsetFeature(b, sr, hop), fps, 0.15, 0.05)
	res := XcorrFeatures(fa, fb, fps, minOverlapSec, maxOffsetSec, 0.25)
	if res == nil {
		return OffsetResult{}
	}
	return *res
}

func CorrBytes(na, nb int) int64 {
	if na < 1 || nb < 1 {
		return 0
	}
	return 64 * int64(NextFastLen(na+nb-1))
}
