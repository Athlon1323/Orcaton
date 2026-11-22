package dsp

import (
	"math"
	"sync"
)

const streamBlock = 4096

const onsetWorkers = 4

type band struct{ lo, hi int }

type OnsetStream struct {
	sr, hop int
	fb      [][]float64
	bands   []band
	win32   []float32
	buf     []float32
	base    int
	total   int
	next    int
	blocks  [][][nBands]float32
	done    bool
}

func NewOnsetStream(sr, hop int) *OnsetStream {
	fb := melFilterbank(sr, nfftOnset, nBands)
	cols := nfftOnset/2 + 1
	bands := make([]band, nBands)
	for b := 0; b < nBands; b++ {
		lo, hi := cols, 0
		for k := 0; k < cols; k++ {
			if fb[b][k] != 0 {
				if k < lo {
					lo = k
				}
				hi = k + 1
			}
		}
		if lo > hi {
			lo = hi
		}
		bands[b] = band{lo, hi}
	}
	win := hanning(nfftOnset)
	win32 := make([]float32, nfftOnset)
	for i, v := range win {
		win32[i] = float32(v)
	}
	return &OnsetStream{sr: sr, hop: hop, fb: fb, bands: bands, win32: win32,
		buf: make([]float32, nfftOnset/2, streamBlock*hop+2*nfftOnset)}
}

func (s *OnsetStream) Write(x []float32) {
	step := streamBlock * s.hop
	for len(x) > 0 {
		n := min(len(x), step)
		s.buf = append(s.buf, x[:n]...)
		s.total += n
		x = x[n:]
		s.compute(false)
	}
}

func (s *OnsetStream) ready() int {
	end := s.base + len(s.buf)
	n := s.total / s.hop
	if end >= nfftOnset {
		n = min(n, (end-nfftOnset)/s.hop+1)
	} else {
		n = 0
	}
	return n
}

func (s *OnsetStream) compute(final bool) {
	r := s.ready()
	if !final {
		r = s.next + (r-s.next)/streamBlock*streamBlock
	}
	for s.next < r {
		k1 := min(r, (s.next/streamBlock+1)*streamBlock)
		s.frames(s.next, k1)
		s.next = k1
	}
	if drop := s.next*s.hop - s.base; drop > 0 && drop <= len(s.buf) {
		n := copy(s.buf, s.buf[drop:])
		s.buf = s.buf[:n]
		s.base += drop
	}
}

func (s *OnsetStream) frames(k0, k1 int) {
	bi := k0 / streamBlock
	for len(s.blocks) <= bi {
		s.blocks = append(s.blocks, nil)
	}
	if s.blocks[bi] == nil {
		s.blocks[bi] = make([][nBands]float32, 0, streamBlock)
	}
	blk := s.blocks[bi][:k1-bi*streamBlock]
	s.blocks[bi] = blk
	cols := nfftOnset/2 + 1
	chunk := (k1 - k0 + onsetWorkers - 1) / onsetWorkers
	var wg sync.WaitGroup
	for w := 0; w < onsetWorkers; w++ {
		s0, s1 := k0+w*chunk, min(k0+(w+1)*chunk, k1)
		if s0 >= s1 {
			continue
		}
		wg.Add(1)
		go func(s0, s1 int) {
			defer wg.Done()
			frame := make([]float64, nfftOnset)
			spec := make([]float64, cols)
			X := make([]complex128, cols)
			for k := s0; k < s1; k++ {
				off := k*s.hop - s.base
				seg := s.buf[off : off+nfftOnset]
				for i := range frame {
					frame[i] = float64(seg[i] * s.win32[i])
				}
				RFFTInto(frame, nfftOnset, X)
				for i, c := range X {
					a := float32(math.Hypot(real(c), imag(c)))
					spec[i] = float64(a * a)
				}
				row := &blk[k-bi*streamBlock]
				for b := 0; b < nBands; b++ {
					sum := 0.0
					for i := s.bands[b].lo; i < s.bands[b].hi; i++ {
						sum += spec[i] * s.fb[b][i]
					}
					row[b] = float32(math.Log1p(1e4 * sum))
				}
			}
		}(s0, s1)
	}
	wg.Wait()
}

func (s *OnsetStream) at(k int) *[nBands]float32 {
	return &s.blocks[k/streamBlock][k%streamBlock]
}

func (s *OnsetStream) Finish() []float32 {
	if s.done {
		panic("OnsetStream.Finish: called twice")
	}
	s.done = true
	nOut := s.total / s.hop
	if nOut < 4 {
		return make([]float32, max(nOut, 1))
	}
	s.buf = append(s.buf, make([]float32, nfftOnset)...)
	s.compute(true)
	s.buf = nil
	for k := nOut - 1; k >= 1; k-- {
		cur, prv := s.at(k), s.at(k-1)
		for b := 0; b < nBands; b++ {
			d := cur[b] - prv[b]
			if d > 0 {
				cur[b] = d
			} else {
				cur[b] = 0
			}
		}
	}
	*s.at(0) = [nBands]float32{}
	col := make([]float64, nOut)
	for b := 0; b < nBands; b++ {
		for k := 0; k < nOut; k++ {
			col[k] = float64(s.at(k)[b])
		}
		m := float32(Mean(col)) + float32(1e-9)
		for k := 0; k < nOut; k++ {
			s.at(k)[b] /= m
		}
	}
	onset := col
	row := make([]float64, nBands)
	for k := 0; k < nOut; k++ {
		r := s.at(k)
		for b := 0; b < nBands; b++ {
			row[b] = float64(r[b])
		}
		onset[k] = float64(float32(Mean(row)))
	}
	s.blocks = nil
	m := Mean(onset)
	for k := range onset {
		onset[k] -= m
	}
	sd := Std(onset)
	if sd > 0 {
		for k := range onset {
			onset[k] /= sd
		}
	}
	return ToF32(onset)
}
