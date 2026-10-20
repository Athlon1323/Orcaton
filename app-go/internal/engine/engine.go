package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orcaton/internal/core"
	"orcaton/internal/sysres"
)

func (e *Engine) Sync(ctx context.Context, project *core.Project, seqIndex int) (*core.Project, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.O.Validate(); err != nil {
		return nil, err
	}
	if project == nil || seqIndex < 0 || seqIndex >= len(project.Sequences) {
		n := 0
		if project != nil {
			n = len(project.Sequences)
		}
		return nil, fmt.Errorf("секвенция №%d не найдена: в проекте их %d", seqIndex, n)
	}
	out := project.Clone()
	seq := out.Sequences[seqIndex]
	t0 := time.Now()

	units := e.buildUnits(seq)
	e.Report = &Report{Units: units}
	defer e.setupResources(units)()
	e.prep, e.prepLN = nil, nil
	defer func() { e.prep, e.prepLN = nil, nil }()
	defer func() { e.Report.PeakMem = sysres.PeakRSS() }()
	if err := e.loadRecordTimes(ctx, units); err != nil {
		return nil, err
	}
	if e.O.OrderBy == OrderByName {
		assignSeries(units)
	}
	tr := time.Now()
	if err := e.loadFeatures(ctx, units); err != nil {
		return nil, err
	}
	var dups []*Unit
	newTracks := out.SourceFormat != "drt"
	if e.Report.Duplicates = markDuplicates(units); e.Report.Duplicates > 0 {
		if !newTracks {
			keepDuplicatesInLane(units)
		}
		units, dups = splitDuplicates(units)
		e.Report.Units = units
	}
	t1 := time.Now()

	edges := e.correlate(ctx, units)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t2 := time.Now()

	comps, err := e.solveAll(ctx, units, edges)
	if err != nil {
		return nil, err
	}
	e.emit(StageLayout, 0, 1, "")
	e.seqFPS = nil
	if seq.Rate.Valid() {
		e.seqFPS = seq.Rate.FPS
	}
	e.layout(units, comps)
	if len(dups) > 0 {
		units = attachDuplicates(units, dups)
		if !newTracks {
			e.fitDuplicatesInPlace(units)
		}
		e.Report.Units = units
	}
	e.apply(seq, units)
	if len(dups) > 0 && newTracks {
		moveDuplicateClips(seq, units)
	}
	e.emit(StageLayout, 1, 1, "")
	t3 := time.Now()

	e.Report.Timings = []Timing{{"record_time", tr.Sub(t0).Seconds()}, {"features", t1.Sub(tr).Seconds()},
		{"correlate", t2.Sub(t1).Seconds()}, {"solve", t3.Sub(t2).Seconds()}}
	if !strings.HasSuffix(seq.Name, " - synced") {
		seq.Name += " - synced"
	}
	return out, nil
}

func (e *Engine) solveAll(ctx context.Context, units []*Unit, edges []*Edge) ([][]int, error) {
	const steps = 3
	e.emit(StageSolve, 0, steps, "")
	comps := e.solve(units, edges)
	comps = e.pruneClockOutliers(units, comps)
	e.emit(StageSolve, 1, steps, "")
	if e.O.WindowSearch && e.O.FileOrder {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		e.windowPass(units, comps)
		comps = e.pruneClockOutliers(units, comps)
	}
	e.emit(StageSolve, 2, steps, "")
	if e.O.RecordTime && e.O.FileOrder {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		e.fixEndStamps(units)
		comps = e.mergeByClock(units, comps)
		e.timePass(units, comps)
	}
	e.checkConflicts(units, comps)
	e.emit(StageSolve, steps, steps, "")
	return comps, ctx.Err()
}
