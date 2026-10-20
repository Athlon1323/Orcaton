package engine

import (
	"math"
	"runtime"
	"runtime/debug"
	"sync"

	"orcaton/internal/media"
	"orcaton/internal/sysres"
)

const gateShare = 0.5

func (e *Engine) setupResources(units []*Unit) func() {
	path := ""
	for _, u := range units {
		if u.MediaPath != nil && *u.MediaPath != "" {
			path = *u.MediaPath
			break
		}
	}
	if _, bundle := e.Features.(*media.FeatureBundle); bundle {
		path = ""
	}
	e.res = sysres.Resolve(e.O.MemLimitMB, e.O.Threads, e.O.Jobs, path)
	if path != "" {
		e.res.FFmpeg = media.FFmpegVersion()
	}
	e.gate = newMemGate(int64(float64(e.res.MemLimit) * gateShare))
	e.Report.Resources = e.res
	prev := debug.SetMemoryLimit(-1)
	if prev != math.MaxInt64 {
		return func() {}
	}
	debug.SetMemoryLimit(int64(e.res.MemLimit))
	return func() { debug.SetMemoryLimit(prev) }
}

func (e *Engine) workers() int {
	if e.res.Threads > 0 {
		return e.res.Threads
	}
	return max(runtime.NumCPU(), 1)
}

type memGate struct {
	mu            sync.Mutex
	cond          *sync.Cond
	limit, used   int64
	peak          int64
	next, serving uint64
}

func newMemGate(limit int64) *memGate {
	g := &memGate{limit: max(limit, 1)}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *memGate) acquire(n int64) int64 {
	n = min(max(n, 0), g.limit)
	g.mu.Lock()
	t := g.next
	g.next++
	for t != g.serving || (g.used > 0 && g.used+n > g.limit) {
		g.cond.Wait()
	}
	g.serving++
	g.used += n
	g.peak = max(g.peak, g.used)
	g.cond.Broadcast()
	g.mu.Unlock()
	return n
}

func (g *memGate) release(n int64) {
	g.mu.Lock()
	g.used -= n
	g.cond.Broadcast()
	g.mu.Unlock()
}

func parallelForMem(n, workers int, g *memGate, cost func(i int) int64, f func(i int)) {
	if g == nil || workers <= 1 || n <= 1 {
		parallelFor(n, workers, f)
		return
	}
	parallelFor(n, workers, func(i int) {
		c := g.acquire(cost(i))
		defer g.release(c)
		f(i)
	})
}
