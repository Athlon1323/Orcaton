package api

import (
	"context"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
)

type Stage string

const (
	StageSearch     = Stage(engine.StageSearch)
	StageRecordTime = Stage(engine.StageRecordTime)
	StageFeatures   = Stage(engine.StageFeatures)
	StagePairs      = Stage(engine.StagePairs)
	StageRefine     = Stage(engine.StageRefine)
	StageSolve      = Stage(engine.StageSolve)
	StageLayout     = Stage(engine.StageLayout)
	StageWrite      = Stage(engine.StageWrite)
)

type Event struct {
	Stage Stage  `json:"stage"`
	Title string `json:"title"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	File  string `json:"file,omitempty"`
}

func progress(on func(Event)) engine.Progress {
	if on == nil {
		return nil
	}
	return func(ev engine.Event) {
		on(Event{Stage: Stage(ev.Stage), Title: ev.Title, Done: ev.Done, Total: ev.Total, File: ev.File})
	}
}

type SyncRequest struct {
	Options   Options  `json:"options"`
	Features  []string `json:"features,omitempty"`
	Originals []string `json:"originals,omitempty"`
	Times     []string `json:"times,omitempty"`
}

const (
	ResultAudio     = "audio"
	ResultClock     = "clock"
	ResultUnmatched = "unmatched"
	ResultOffline   = "offline"
)

type FileResult struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Lane        string   `json:"lane"`
	Camera      string   `json:"camera"`
	Result      string   `json:"result"`
	Status      string   `json:"status"`
	PositionSec *float64 `json:"positionSec,omitempty"`
	DurationSec float64  `json:"durationSec"`
	SourceInSec float64  `json:"sourceInSec"`
	Group       int      `json:"group"`
	Partner     string   `json:"partner,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	Ratio       float64  `json:"ratio,omitempty"`
	RecordStart string   `json:"recordStart,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	ReasonCode  string   `json:"reasonCode,omitempty"`
	Tracks      []string `json:"tracks"`
	StartSec    float64  `json:"startSec"`
	PositionTC  string   `json:"positionTC,omitempty"`
}

const (
	ReasonNoMatch     = engine.ReasonNoMatch
	ReasonShortAudio  = engine.ReasonShortAudio
	ReasonNoAudio     = engine.ReasonNoAudio
	ReasonOffline     = engine.ReasonOffline
	ReasonNoPath      = engine.ReasonNoPath
	ReasonNotFound    = engine.ReasonNotFound
	ReasonDecodeError = engine.ReasonDecodeError
	ReasonConflict    = engine.ReasonConflict
	ReasonByClock     = engine.ReasonByClock
)

type Timing struct {
	Name string  `json:"name"`
	Sec  float64 `json:"sec"`
}

type Summary struct {
	Files           int            `json:"files"`
	Audio           int            `json:"audio"`
	Clock           int            `json:"clock"`
	Unmatched       int            `json:"unmatched"`
	Offline         int            `json:"offline"`
	TriedPairs      int            `json:"triedPairs"`
	Edges           int            `json:"edges"`
	Threshold       float64        `json:"threshold"`
	Noise           float64        `json:"noise"`
	Ambiguous       int            `json:"ambiguous"`
	RejectedOverlap int            `json:"rejectedOverlap"`
	RejectedOrder   int            `json:"rejectedOrder"`
	Duplicates      int            `json:"duplicates"`
	WindowSynced    int            `json:"windowSynced"`
	WithRecordTime  int            `json:"withRecordTime"`
	TimeAudio       int            `json:"timeAudio"`
	TimeShift       int            `json:"timeShift"`
	ClockMerged     int            `json:"clockMerged"`
	GapsCompressed  int            `json:"gapsCompressed"`
	Inconsistent    int            `json:"inconsistent"`
	Conflicts       int            `json:"conflicts"`
	Seconds         float64        `json:"seconds"`
	Reasons         map[string]int `json:"reasons"`
	Timings         []Timing       `json:"timings"`
	Resources       Resources      `json:"resources"`
}

type Resources struct {
	RAMMB      int    `json:"ramMB"`
	MemLimitMB int    `json:"memLimitMB"`
	Threads    int    `json:"threads"`
	Jobs       int    `json:"jobs"`
	Disk       string `json:"disk"`
	PeakMB     int    `json:"peakMB"`
	FFmpeg     string `json:"ffmpeg"`
}

type Result struct {
	run     *app.Run
	Summary Summary      `json:"summary"`
	Files   []FileResult `json:"files"`
}

func Sync(ctx context.Context, p *Project, req SyncRequest, onEvent func(Event)) (*Result, error) {
	o, err := req.Options.engine()
	if err != nil {
		return nil, err
	}
	run, err := app.Sync(ctx, p.p, app.SyncRequest{Options: o, Features: req.Features,
		Originals: req.Originals, Times: req.Times}, progress(onEvent))
	if err != nil {
		return nil, err
	}
	return newResult(run), nil
}

func newResult(run *app.Run) *Result {
	r := run.Report
	res := &Result{run: run, Files: []FileResult{}}
	partners := app.BestPartners(r)
	seq := sequenceInfo(run.Project.Main(), 0)
	for _, u := range app.ReportOrder(r) {
		f := FileResult{Name: u.Name(), Lane: u.Lane, Camera: u.Device, Status: string(u.Status),
			DurationSec: num.Float(u.Duration), SourceInSec: num.Float(u.SourceIn),
			Group: u.Component, Reason: app.Reason(u), ReasonCode: app.ReasonCode(u),
			Tracks: append([]string{}, u.Tracks...), StartSec: num.Float(u.Start0)}
		if u.MediaPath != nil {
			f.Path = *u.MediaPath
		}
		if u.Position != nil {
			v := num.Float(u.Position)
			f.PositionSec = &v
			f.PositionTC = FormatTC(v, seq)
		}
		if pt, ok := partners[u.UID]; ok {
			f.Partner, f.Confidence, f.Ratio = pt.Name, pt.Score, pt.Ratio
		}
		if u.RecTime != nil {
			f.RecordStart = pyio.ISOUTC(*u.RecTime)
		}
		switch {
		case u.Status == core.StatusSynced:
			f.Result = ResultAudio
		case u.Status == core.StatusByTimecode:
			f.Result = ResultClock
		case u.Offline:
			f.Result = ResultOffline
		default:
			f.Result = ResultUnmatched
		}
		res.Files = append(res.Files, f)
	}
	s := &res.Summary
	s.Files = len(r.Units)
	s.Audio, s.Clock = r.Count(core.StatusSynced), r.Count(core.StatusByTimecode)
	s.Offline = r.Offline
	s.Unmatched = r.Count(core.StatusUnsynced) - r.Offline
	s.TriedPairs, s.Edges = r.TriedPairs, len(r.Edges)
	s.Threshold, s.Noise = num.Round(r.Threshold, 1), num.Round(r.NoiseLevel, 1)
	s.Ambiguous, s.RejectedOverlap, s.RejectedOrder = r.Ambiguous, r.RejectedOverlap, r.RejectedOrder
	s.Duplicates = r.Duplicates
	s.WindowSynced, s.WithRecordTime, s.TimeAudio = r.WindowSynced, r.WithRecTime, r.TimeAudio
	s.TimeShift = r.TimeShiftTested
	s.ClockMerged, s.GapsCompressed = r.ClockMerged, r.GapsCompressed
	s.Inconsistent, s.Conflicts = r.InconsistentEdges, r.Conflicts
	s.Seconds = num.Round(r.Seconds(), 1)
	s.Reasons = map[string]int{}
	for _, u := range r.Units {
		if u.Error != nil && *u.Error != "" {
			s.Reasons[*u.Error]++
		}
	}
	for _, t := range r.Timings {
		s.Timings = append(s.Timings, Timing{t.Name, t.Sec})
	}
	rs := r.Resources
	s.Resources = Resources{RAMMB: int(rs.RAM >> 20), MemLimitMB: int(rs.MemLimit >> 20),
		Threads: rs.Threads, Jobs: rs.Jobs, Disk: rs.Disk, PeakMB: int(r.PeakMem >> 20), FFmpeg: rs.FFmpeg}
	return res
}

type SaveOptions struct {
	Format    string `json:"format"`
	KeepPaths bool   `json:"keepPaths"`
}

func (r *Result) Save(path string, o SaveOptions, onEvent func(Event)) error {
	return app.Save(r.run, path, o.Format, o.KeepPaths, progress(onEvent))
}

func (r *Result) SaveReport(path string) error { return app.WriteReport(path, r.run.Report) }
