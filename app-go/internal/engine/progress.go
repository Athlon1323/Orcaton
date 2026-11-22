package engine

type Stage string

const (
	StageSearch     Stage = "search"
	StageRecordTime Stage = "record_time"
	StageFeatures   Stage = "features"
	StagePairs      Stage = "pairs"
	StageRefine     Stage = "refine"
	StageSolve      Stage = "solve"
	StageLayout     Stage = "layout"
	StageWrite      Stage = "write"
)

var StageTitles = map[Stage]string{
	StageSearch: "Media search", StageRecordTime: "Recording time", StageFeatures: "Audio analysis",
	StagePairs: "Pair matching", StageRefine: "Pair refinement", StageSolve: "Group assembly",
	StageLayout: "Layout", StageWrite: "Writing",
}

type Event struct {
	Stage Stage  `json:"stage"`
	Title string `json:"title"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	File  string `json:"file,omitempty"`
}

type Progress func(Event)

func (e *Engine) emit(st Stage, done, total int, file string) {
	if e.Progress == nil {
		return
	}
	e.Progress(Event{Stage: st, Title: StageTitles[st], Done: done, Total: total, File: file})
}
