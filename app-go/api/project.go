package api

import (
	"context"
	"os"
	"sort"
	"strings"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/formats"
	_ "orcaton/internal/formats/all"
	"orcaton/internal/media"
	"orcaton/internal/num"
)

type Project struct {
	p    *core.Project
	path string
}

type OpenOptions struct {
	Format  string `json:"format"`
	PathMap string `json:"pathMap"`
}

func Open(path string, o OpenOptions) (*Project, error) {
	pm := o.PathMap
	p, err := formats.ReadProject(path, o.Format, &pm)
	if err != nil {
		return nil, err
	}
	return &Project{p: p, path: path}, nil
}

const (
	FileOK      = "ok"
	FileMissing = "missing"
	FileOffline = "offline"
	FileNoMedia = "nomedia"
)

type MediaFile struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	OriginalPath string   `json:"originalPath,omitempty"`
	Lane         string   `json:"lane"`
	Camera       string   `json:"camera"`
	DurationSec  float64  `json:"durationSec"`
	HasAudio     bool     `json:"hasAudio"`
	State        string   `json:"state"`
	Tracks       []string `json:"tracks"`
	StartSec     float64  `json:"startSec"`
	StartTC      string   `json:"startTC"`
}

type Camera struct {
	ID     string   `json:"id"`
	Lane   string   `json:"lane"`
	Folder string   `json:"folder"`
	Files  int      `json:"files"`
	Tracks []string `json:"tracks"`
	Title  string   `json:"title"`
	ext    string
}

type SequenceInfo struct {
	Name        string  `json:"name"`
	FPS         string  `json:"fps"`
	FPSNum      int64   `json:"fpsNum"`
	FPSDen      int64   `json:"fpsDen"`
	StartSec    float64 `json:"startSec"`
	StartTC     string  `json:"startTC"`
	DurationSec float64 `json:"durationSec"`
	Tracks      int     `json:"tracks"`
	Clips       int     `json:"clips"`
}

type OutputFormat struct {
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Extensions []string `json:"extensions"`
}

type ProjectInfo struct {
	Path          string         `json:"path"`
	Name          string         `json:"name"`
	Format        string         `json:"format"`
	Sequences     []SequenceInfo `json:"sequences"`
	Cameras       []Camera       `json:"cameras"`
	Files         []MediaFile    `json:"files"`
	Missing       int            `json:"missing"`
	Offline       int            `json:"offline"`
	OutputFormats []OutputFormat `json:"outputFormats"`
}

func (p *Project) Info() ProjectInfo {
	inf := ProjectInfo{Path: p.path, Name: p.p.Name, Format: p.p.SourceFormat,
		OutputFormats: p.OutputFormats()}
	for _, s := range p.p.Sequences {
		n := 0
		for _, t := range s.Tracks {
			n += len(t.Clips)
		}
		inf.Sequences = append(inf.Sequences, sequenceInfo(s, n))
	}
	seq := inf.Sequences[0]
	cams := map[string]int{}
	camTracks := map[string]map[string]bool{}
	for _, u := range engine.PlanUnits(p.p.Sequences[0]) {
		f := mediaFile(u, seq)
		switch f.State {
		case FileMissing:
			inf.Missing++
		case FileOffline:
			inf.Offline++
		}
		inf.Files = append(inf.Files, f)
		i, ok := cams[u.Device]
		if !ok {
			lane, folder, _ := strings.Cut(u.Device, "|")
			folder, ext, _ := strings.Cut(folder, "|")
			i = len(inf.Cameras)
			cams[u.Device] = i
			inf.Cameras = append(inf.Cameras, Camera{ID: u.Device, Lane: lane, Folder: folder, ext: ext})
			camTracks[u.Device] = map[string]bool{}
		}
		inf.Cameras[i].Files++
		for _, t := range u.Tracks {
			camTracks[u.Device][t] = true
		}
	}
	for i := range inf.Cameras {
		c := &inf.Cameras[i]
		c.Tracks = sortedKeys(camTracks[c.ID])
		c.Title = cameraTitle(c.Folder, c.ext, c.Tracks)
	}
	return inf
}

func sequenceInfo(s *core.Sequence, clips int) SequenceInfo {
	si := SequenceInfo{Name: s.Name, FPS: s.Rate.String(), DurationSec: num.Float(s.Duration()),
		Tracks: len(s.Tracks), Clips: clips}
	if s.Rate.FPS != nil {
		si.FPSNum, si.FPSDen = s.Rate.FPS.Num().Int64(), s.Rate.FPS.Denom().Int64()
	}
	if s.TCStart != nil {
		si.StartSec = num.Float(s.TCStart)
	}
	si.StartTC = FormatTC(0, si)
	return si
}

func cameraTitle(folder, ext string, tracks []string) string {
	ts := append([]string(nil), tracks...)
	sort.SliceStable(ts, func(i, j int) bool { return ts[i][0] == 'V' && ts[j][0] != 'V' })
	t := strings.Join(ts, "+")
	name := formats.BaseName(folder)
	if ext != "" {
		name = strings.TrimSpace(name + " " + strings.ToUpper(strings.TrimPrefix(ext, ".")))
	}
	switch {
	case name == "":
		return t
	case t == "":
		return name
	}
	return name + " · " + t
}

func sortedKeys(m map[string]bool) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mediaFile(u *engine.Unit, seq SequenceInfo) MediaFile {
	f := MediaFile{Name: u.Name(), Lane: u.Lane, Camera: u.Device,
		DurationSec: num.Float(u.Duration), HasAudio: u.HasAudio,
		Tracks: append([]string{}, u.Tracks...), StartSec: num.Float(u.Start0)}
	f.StartTC = FormatTC(f.StartSec, seq)
	m := u.Clips[0].Media
	switch {
	case m == nil:
		f.State = FileNoMedia
		return f
	case m.Path != nil:
		f.Path = *m.Path
	}
	if m.OriginalPath != nil && *m.OriginalPath != f.Path {
		f.OriginalPath = *m.OriginalPath
	}
	switch {
	case m.Offline || f.Path == "":
		f.State = FileOffline
	case !isFile(f.Path):
		f.State = FileMissing
	default:
		f.State = FileOK
	}
	return f
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func (p *Project) OutputFormats() []OutputFormat {
	var out, xml []OutputFormat
	for _, e := range formats.Exporters() {
		f := OutputFormat{e.Name(), e.Title(), e.Extensions()}
		switch e.Name() {
		case p.p.SourceFormat:
			out = append(out, f)
		case "xmeml":
			xml = append(xml, f)
		}
	}
	if len(out) == 0 {
		return xml
	}
	return out
}

type MissingFile struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
}

const (
	AnswerFolder = string(app.AnswerFolder)
	AnswerSkip   = string(app.AnswerSkip)
	AnswerCancel = string(app.AnswerCancel)
)

type Answer struct {
	Action string `json:"action"`
	Folder string `json:"folder,omitempty"`
}

type Asker interface {
	AskMissing(ctx context.Context, missing []MissingFile) (Answer, error)
}

type AskFunc func(ctx context.Context, missing []MissingFile) (Answer, error)

func (f AskFunc) AskMissing(ctx context.Context, missing []MissingFile) (Answer, error) {
	return f(ctx, missing)
}

type SearchReport struct {
	Answer    bool          `json:"answer"`
	Roots     []string      `json:"roots"`
	BadRoots  []string      `json:"badRoots"`
	Before    int           `json:"before"`
	Found     int           `json:"found"`
	Ambiguous []Ambiguous   `json:"ambiguous"`
	NotFound  []MissingFile `json:"notFound"`
}

type Ambiguous struct {
	File       MissingFile `json:"file"`
	Candidates []string    `json:"candidates"`
}

type FindRequest struct {
	Roots    []string
	Depth    int
	Ask      Asker
	OnSearch func(SearchReport)
	OnEvent  func(Event)
}

func (p *Project) FindMedia(ctx context.Context, req FindRequest) ([]MissingFile, error) {
	depth := req.Depth
	if depth <= 0 {
		depth = media.DefaultSearchDepth
	}
	r := app.FindRequest{Roots: req.Roots, Depth: depth, Progress: progress(req.OnEvent)}
	if req.Ask != nil {
		r.Ask = app.AskFunc(func(ctx context.Context, ms []*core.MediaRef) (app.Answer, error) {
			a, err := req.Ask.AskMissing(ctx, missingFiles(ms))
			return app.Answer{Action: app.Action(a.Action), Folder: a.Folder}, err
		})
	}
	if req.OnSearch != nil {
		r.OnSearch = func(s app.SearchReport) {
			out := SearchReport{Answer: s.Answer, Roots: s.Roots, BadRoots: s.BadRoots,
				Before: s.Before, Found: s.Found, NotFound: missingFiles(s.NotFound)}
			for _, a := range s.Ambiguous {
				out.Ambiguous = append(out.Ambiguous, Ambiguous{missingFile(a.Media), a.Candidates})
			}
			req.OnSearch(out)
		}
	}
	ms, err := app.FindMedia(ctx, p.p, r)
	return missingFiles(ms), err
}

func missingFile(m *core.MediaRef) MissingFile {
	f := MissingFile{Name: formats.BaseName(*m.Path), Path: *m.Path, OriginalPath: *m.Path}
	if m.OriginalPath != nil {
		f.OriginalPath = *m.OriginalPath
	}
	return f
}

func missingFiles(ms []*core.MediaRef) []MissingFile {
	out := []MissingFile{}
	for _, m := range ms {
		out = append(out, missingFile(m))
	}
	return out
}
