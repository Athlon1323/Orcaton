package engine

import (
	"math"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/dsp"
	"orcaton/internal/media"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
	"orcaton/internal/sysres"
)

type Placement string

const (
	PlaceEdges     Placement = "edges"
	PlaceOrderTime Placement = "order_time"
	PlaceEnd       Placement = "end"
	PlaceRemove    Placement = "remove"
)

const (
	OrderByName       = "name"
	OrderByTime       = "time"
	SinglesSequential = "sequential"
	SinglesStacked    = "stacked"
)

var (
	OrderModes   = []string{OrderByName, OrderByTime}
	SinglesModes = []string{SinglesSequential, SinglesStacked}
)

func ParsePlacement(s string) (Placement, bool) {
	switch Placement(s) {
	case PlaceEdges, PlaceOrderTime, PlaceEnd, PlaceRemove:
		return Placement(s), true
	}
	return "", false
}

type Options struct {
	SampleRate        int
	Hop               int
	AudioMode         string
	MinConfidence     float64
	AutoThreshold     bool
	NoiseK            float64
	MinRatio          float64
	Dominance         float64
	AmbiguityFloor    float64
	CompetitorRatio   float64
	MinOverlapSec     float64
	MaxOffsetSec      float64
	CoarseFactor      int
	RefineRadiusSec   float64
	IntraLane         bool
	Unsynced          Placement
	OrderBy           string
	Singles           string
	UnsyncedLabel     string
	GroupGapSec       float64
	FileOrder         bool
	WindowSearch      bool
	WindowMinConf     float64
	WindowRatio       float64
	RecordTime        bool
	TimeSearchSec     float64
	TimeMinConf       float64
	TimeRatio         float64
	TimeMinActivity   float64
	ClockToleranceSec float64
	MaxTimeGapSec     float64
	IdleGapSec        float64
	IdleKeepSec       float64
	SpannedTolSec     float64
	TimeLabel         string
	Jobs              int
	Threads           int
	MemLimitMB        int
	CacheDir          string
}

func DefaultOptions() Options {
	return Options{
		SampleRate: 8000, Hop: 80, AudioMode: "mix",
		MinConfidence: 12.0, AutoThreshold: true, NoiseK: 6.0, MinRatio: 1.5,
		Dominance: 1.5, AmbiguityFloor: 0.75, CompetitorRatio: 1.3,
		MinOverlapSec: 2.0, MaxOffsetSec: -1, CoarseFactor: 4, RefineRadiusSec: 1.0,
		IntraLane: false, Unsynced: PlaceOrderTime, OrderBy: OrderByName, Singles: SinglesSequential, UnsyncedLabel: "Rose", GroupGapSec: 0.0,
		FileOrder: true, WindowSearch: true, WindowMinConf: 10.0, WindowRatio: 2.0,
		RecordTime: true, TimeSearchSec: 3.0, TimeMinConf: 8.0, TimeRatio: 2.0,
		TimeMinActivity: 0.05, ClockToleranceSec: 3.0, MaxTimeGapSec: 600.0,
		IdleGapSec: 0.0, IdleKeepSec: 0.0, SpannedTolSec: 1.5, TimeLabel: "Mango", Jobs: 0,
	}
}

type OrderKey struct {
	T      float64
	Name   string
	Series string
	Seq    float64
}

func (a OrderKey) Less(b OrderKey) bool {
	if a.Series != "" && a.Series == b.Series && a.Seq != b.Seq {
		return a.Seq < b.Seq
	}
	return a.Before(b)
}

func (a OrderKey) Before(b OrderKey) bool {
	if a.T != b.T {
		return a.T < b.T
	}
	return a.Name < b.Name
}

type Unit struct {
	UID       int
	Lane      string
	Clips     []*core.Clip
	Tracks    []string
	trackSet  map[string]bool
	MediaPath *string
	SourceIn  num.Q
	Duration  num.Q
	Anchor    num.Q
	Start0    num.Q
	Order     OrderKey
	HasAudio  bool
	Device    string
	RecTime   *float64
	recAlt    []media.NameStamp
	TC        *float64
	Offline   bool
	Feature   []float32
	DupOf     *Unit
	inLane    bool
	Error     *string
	Reason    string
	Position  num.Q
	Status    core.SyncStatus
	Component int
	BestScore float64
	dur       float64
}

func (u *Unit) Name() string {
	if u.Clips[0].Media != nil {
		return u.Clips[0].Media.Name
	}
	return u.Clips[0].Name
}

func (u *Unit) SharesTrack(v *Unit) bool {
	for _, t := range u.Tracks {
		if v.trackSet[t] {
			return true
		}
	}
	return false
}

func (u *Unit) setError(code, s string) { u.Error, u.Reason = &s, code }

func (u *Unit) clearError() { u.Error, u.Reason = nil, "" }

const (
	ReasonNoMatch     = "no_match"
	ReasonShortAudio  = "short_audio"
	ReasonNoAudio     = "no_audio"
	ReasonOffline     = "offline"
	ReasonNoPath      = "no_path"
	ReasonNotFound    = "not_found"
	ReasonDecodeError = "decode_error"
	ReasonConflict    = "conflict"
	ReasonByClock     = "by_clock"
	ReasonDuplicate   = "duplicate"
)

func (u *Unit) errorIs(s string) bool { return u.Error != nil && *u.Error == s }

func (u *Unit) placed() bool {
	return u.Status == core.StatusSynced || u.Status == core.StatusByTimecode
}

type Edge struct {
	A, B    int
	Offset  float64
	Res     dsp.OffsetResult
	CoarseZ float64
}

func (e *Edge) Score() float64 { return e.Res.Confidence }

type Timing struct {
	Name string
	Sec  float64
}

type Report struct {
	Units             []*Unit
	Edges             []*Edge
	TriedPairs        int
	Threshold         float64
	NoiseLevel        float64
	TreeEdges         []*Edge
	RejectedOverlap   int
	RejectedOrder     int
	SplitStamps       int
	Ambiguous         int
	Duplicates        int
	InconsistentEdges int
	Conflicts         int
	WindowSynced      int
	TimeAudio         int
	TimeShiftTested   int
	TimePlaced        int
	ClockSpread       float64
	ClockMerged       int
	ClockDropped      int
	EndStampCams      []string
	TimeInserted      int
	GapsCompressed    int
	Offline           int
	WithRecTime       int
	Timings           []Timing
	Resources         sysres.Resources
	PeakMem           uint64
}

func (r *Report) Count(s core.SyncStatus) int {
	n := 0
	for _, u := range r.Units {
		if u.Status == s {
			n++
		}
	}
	return n
}

func (r *Report) Seconds() float64 {
	s := 0.0
	for _, t := range r.Timings {
		s += t.Sec
	}
	return s
}

func (r *Report) Summary() *pyio.OMap {
	reasons := pyio.NewOMap()
	for _, u := range r.Units {
		if u.Error != nil && *u.Error != "" {
			v, _ := reasons.Get(*u.Error)
			n, _ := v.(int)
			reasons.Set(*u.Error, n+1)
		}
	}
	m := pyio.NewOMap()
	m.Set("units", len(r.Units)).
		Set("synced", r.Count(core.StatusSynced)).
		Set("by_time", r.Count(core.StatusByTimecode)).
		Set("unsynced", r.Count(core.StatusUnsynced)).
		Set("offline", r.Offline).
		Set("tried_pairs", r.TriedPairs).
		Set("edges", len(r.Edges)).
		Set("threshold", pyio.Float(num.Round(r.Threshold, 1))).
		Set("noise", pyio.Float(num.Round(r.NoiseLevel, 1))).
		Set("ambiguous", r.Ambiguous).
		Set("duplicates", r.Duplicates).
		Set("rejected_overlap", r.RejectedOverlap).
		Set("rejected_order", r.RejectedOrder).
		Set("window_synced", r.WindowSynced).
		Set("with_rec_time", r.WithRecTime).
		Set("time_audio", r.TimeAudio).
		Set("time_shift", r.TimeShiftTested).
		Set("time_placed", r.TimePlaced).
		Set("clock_merged", r.ClockMerged).
		Set("clock_dropped", r.ClockDropped).
		Set("end_stamp_cams", len(r.EndStampCams)).
		Set("time_inserted", r.TimeInserted).
		Set("gaps_compressed", r.GapsCompressed).
		Set("inconsistent", r.InconsistentEdges).
		Set("conflicts", r.Conflicts).
		Set("seconds", pyio.Float(num.Round(r.Seconds(), 1))).
		Set("reasons", reasons)
	return m
}

const (
	localNormSec   = 2.0
	localNormFloor = 0.05
)

var overlapTol = num.F(1, 25)

const overlapTolF = 1.0 / 25

type Engine struct {
	O         Options
	Progress  Progress
	Report    *Report
	Features  media.FeatureSource
	Times     *media.RecordTimes
	res       sysres.Resources
	gate      *memGate
	prep      *prepCache
	prepLN    *prepCache
	stampSure map[string]bool
	seqFPS    num.Q
}

func New(o Options, progress Progress, features media.FeatureSource, times *media.RecordTimes) *Engine {
	if progress == nil {
		progress = func(Event) {}
	}
	return &Engine{O: o, Progress: progress, Report: &Report{}, Features: features, Times: times}
}

func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n == 0 {
		return math.NaN()
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func qOr0(q num.Q) num.Q {
	if q == nil {
		return num.I(0)
	}
	return q
}

func fOr0(q num.Q) float64 {
	if q == nil {
		return 0
	}
	return num.Float(q)
}
