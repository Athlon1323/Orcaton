package dsp

import (
	"math"
	"sync"
)

const FeatureVersion = 2

const (
	nfftOnset = 512
	nBands    = 24
)

var (
	melMu    sync.Mutex
	melCache = map[[3]int][][]float64{}
)

func linspace(start, stop float64, num int, endpoint bool) []float64 {
	out := make([]float64, num)
	if num == 0 {
		return out
	}
	div := num
	if endpoint {
		div = num - 1
	}
	if div <= 0 {
		out[0] = start
		return out
	}
	delta := stop - start
	step := delta / float64(div)
	for i := range out {
		if step == 0 {
			out[i] = float64(i)/float64(div)*delta + start
		} else {
			out[i] = float64(i)*step + start
		}
	}
	if endpoint && num > 1 {
		out[num-1] = stop
	}
	return out
}

func melFilterbank(sr, nfft, nb int) [][]float64 {
	key := [3]int{sr, nfft, nb}
	melMu.Lock()
	defer melMu.Unlock()
	if fb, ok := melCache[key]; ok {
		return fb
	}
	fmax := math.Min(3800.0, float64(sr)/2*0.95)
	hz2mel := func(f float64) float64 { return 2595.0 * math.Log10(1.0+f/700.0) }
	mel2hz := func(m float64) float64 { return 700.0 * (math.Pow(10.0, m/2595.0) - 1.0) }
	mels := linspace(hz2mel(80.0), hz2mel(fmax), nb+2, true)
	bins := make([]int, len(mels))
	for i, m := range mels {
		bins[i] = int(math.Floor(float64(nfft+1) * mel2hz(m) / float64(sr)))
	}
	cols := nfft/2 + 1
	fb := make([][]float64, nb)
	for i := 0; i < nb; i++ {
		fb[i] = make([]float64, cols)
		lo := bins[i]
		c := max(bins[i+1], bins[i]+1)
		hi := max(bins[i+2], bins[i+1]+2)
		up := linspace(0.0, 1.0, c-lo, false)
		for k, v := range up {
			if lo+k >= 0 && lo+k < cols {
				fb[i][lo+k] = v
			}
		}
		down := linspace(1.0, 0.0, hi-c, false)
		for k, v := range down {
			if c+k >= 0 && c+k < cols {
				fb[i][c+k] = v
			}
		}
	}
	melCache[key] = fb
	return fb
}

func hanning(m int) []float64 {
	w := make([]float64, m)
	if m == 1 {
		w[0] = 1
		return w
	}
	for i := 0; i < m; i++ {
		n := float64(1 - m + 2*i)
		w[i] = 0.5 + 0.5*math.Cos(math.Pi*n/float64(m-1))
	}
	return w
}

func OnsetFeature(x []float32, sr, hop int) []float32 {
	s := NewOnsetStream(sr, hop)
	s.Write(x)
	return s.Finish()
}

func MaskOnsetArtifacts(f []float32, fps, guardSec, minSilenceSec float64) []float32 {
	x := append([]float32(nil), f...)
	n := len(x)
	if n < 2 {
		return x
	}
	guard := int(math.RoundToEven(guardSec * fps))
	if guard < 1 {
		guard = 1
	}
	thr := float32(1e-4)
	same := make([]bool, n-1)
	for i := 0; i < n-1; i++ {
		d := x[i+1] - x[i]
		if d < 0 {
			d = -d
		}
		same[i] = d < thr
	}
	lead := 0
	for lead < n-1 && same[lead] {
		lead++
	}
	for i := 0; i < min(n, lead+1+guard); i++ {
		x[i] = 0
	}
	minRun := int(math.RoundToEven(minSilenceSec * fps))
	if minRun < 2 {
		minRun = 2
	}
	i := lead + 1
	for i < n-1 {
		if !same[i] {
			i++
			continue
		}
		j := i
		for j < n-1 && same[j] {
			j++
		}
		if j-i+1 >= minRun {
			for k := j + 1; k < min(n, j+1+guard); k++ {
				x[k] = 0
			}
		}
		i = j + 1
	}
	return x
}

func LocalNormalize(f []float32, fps, winSec, floor float64) []float32 {
	n := len(f)
	out := make([]float32, n)
	if n == 0 {
		return out
	}
	half := max(int(math.RoundToEven(winSec*fps))/2, 1)
	c1 := make([]float64, n+1)
	c2 := make([]float64, n+1)
	for i, v := range f {
		x := float64(v)
		c1[i+1] = c1[i] + x
		c2[i+1] = c2[i] + x*x
	}
	mean := c1[n] / float64(n)
	gv := math.Max(c2[n]/float64(n)-mean*mean, 0)
	for i := range f {
		a, b := max(i-half, 0), min(i+half+1, n)
		k := float64(b - a)
		m := (c1[b] - c1[a]) / k
		v := math.Max((c2[b]-c2[a])/k-m*m, 0)
		d := v + floor*gv
		if d <= 0 {
			continue
		}
		out[i] = float32((float64(f[i]) - m) / math.Sqrt(d))
	}
	return out
}
