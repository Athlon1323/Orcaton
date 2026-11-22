package session

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"orcaton/api"
)

type Session struct {
	ev  emitter
	log *Log

	mu       sync.Mutex
	op       string
	cancel   context.CancelFunc
	project  *api.Project
	path     string
	folders  []string
	result   *api.Result
	saved    string
	settings api.Settings
	answers  chan api.Answer
	dialogs  DialogTexts

	features, times []string
}

func New(em Emitter, log *Log) *Session {
	return &Session{ev: emitter{em: em, log: log}, log: log, settings: api.DefaultSettings(),
		answers: make(chan api.Answer, 1)}
}

func (s *Session) begin(op string) (context.Context, func(), *api.ErrorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.op != "" {
		return nil, nil, &api.ErrorInfo{Kind: ErrBusy, Message: "another operation is running: " + s.op}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.op, s.cancel = op, cancel
	s.log.Printf("begin %s", op)
	return ctx, func() {
		cancel()
		s.mu.Lock()
		s.op, s.cancel = "", nil
		s.mu.Unlock()
		s.log.Printf("end %s", op)
	}, nil
}

func (s *Session) Cancel() {
	s.mu.Lock()
	c := s.cancel
	s.mu.Unlock()
	if c != nil {
		s.log.Printf("cancel")
		c()
	}
}

func (s *Session) Busy() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.op
}

type StartupInfo struct {
	Version       string          `json:"version"`
	APIVersion    string          `json:"apiVersion"`
	OS            string          `json:"os"`
	FFmpeg        *api.ErrorInfo  `json:"ffmpeg,omitempty"`
	FFmpegOld     *api.FFmpegInfo `json:"ffmpegOld,omitempty"`
	Options       api.Options     `json:"options"`
	Defaults      api.Options     `json:"defaults"`
	KeepPaths     bool            `json:"keepPaths"`
	SettingsPath  string          `json:"settingsPath"`
	SettingsError *api.ErrorInfo  `json:"settingsError,omitempty"`
	UnsyncedModes []string        `json:"unsyncedModes"`
	OrderModes    []string        `json:"orderModes"`
	SinglesModes  []string        `json:"singlesModes"`
	AudioModes    []string        `json:"audioModes"`
	LogDir        string          `json:"logDir"`
	Language      string          `json:"language"`
	Languages     []string        `json:"languages"`
}

func (s *Session) Startup() (r StartupInfo) {
	defer s.recoverTo(&r.SettingsError)
	st, err := api.LoadSettings("")
	if err != nil {
		st = api.DefaultSettings()
		r.SettingsError = errInfo(err)
		s.log.Printf("settings: %v", err)
	}
	s.mu.Lock()
	s.settings = st
	s.mu.Unlock()
	r.Version, r.APIVersion, r.OS = api.ProductVersion, api.Version, runtime.GOOS
	r.FFmpeg = s.CheckFFmpeg()
	if r.FFmpeg == nil {
		if fi, err := api.GetFFmpegInfo(); err == nil && fi.Old {
			r.FFmpegOld = &fi
		}
	}
	r.Options, r.Defaults, r.KeepPaths = st.Options, api.DefaultOptions(), st.Media.KeepPaths
	r.SettingsPath = api.UserSettingsPath()
	r.UnsyncedModes = append([]string{}, api.UnsyncedModes...)
	r.OrderModes = append([]string{}, api.OrderModes...)
	r.SinglesModes = append([]string{}, api.SinglesModes...)
	r.AudioModes = append([]string{}, api.AudioModes...)
	r.LogDir = s.log.Dir()
	r.Language, r.Languages = s.savedLanguage(), append([]string{}, Languages...)
	s.log.Printf("startup %s api %s %s", r.Version, r.APIVersion, r.OS)
	return r
}

func (s *Session) CheckFFmpeg() *api.ErrorInfo { return errInfo(api.FFmpegAvailable()) }

type ProjectReply struct {
	Info          *api.ProjectInfo `json:"info,omitempty"`
	Folders       []string         `json:"folders"`
	Options       *api.Options     `json:"options,omitempty"`
	KeepPaths     bool             `json:"keepPaths"`
	SettingsError *api.ErrorInfo   `json:"settingsError,omitempty"`
	Error         *api.ErrorInfo   `json:"error,omitempty"`
}

func (s *Session) Open(path string) (r ProjectReply) {
	defer s.recoverTo(&r.Error)
	_, end, busy := s.begin("open")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	st, err := api.LoadSettings(path)
	if err != nil {
		st = api.DefaultSettings()
		r.SettingsError = errInfo(err)
	}
	p, err := api.Open(path, api.OpenOptions{PathMap: st.Media.Map})
	if err != nil {
		r.Error = errInfo(err)
		s.log.Printf("open %q: %v", path, err)
		r.Folders = s.Folders()
		return r
	}
	info := p.Info()
	s.mu.Lock()
	s.project, s.path, s.result, s.saved, s.settings = p, path, nil, "", st
	s.folders = uniq(append(st.Media.SearchRoots(), s.folders...))
	r.Folders = append([]string{}, s.folders...)
	s.mu.Unlock()
	r.Info, r.Options, r.KeepPaths = &info, &st.Options, st.Media.KeepPaths
	s.log.Printf("open %q: %d files, %d missing, %d offline", path, len(info.Files), info.Missing, info.Offline)
	return r
}

func (s *Session) Folders() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.folders...)
}

func (s *Session) RemoveFolder(dir string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.folders[:0:0]
	for _, f := range s.folders {
		if f != dir {
			out = append(out, f)
		}
	}
	s.folders = out
	return append([]string{}, out...)
}

func (s *Session) FindMedia(add string) (r ProjectReply) {
	defer s.recoverTo(&r.Error)
	ctx, end, busy := s.begin("search")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	s.mu.Lock()
	if add = strings.TrimSpace(add); add != "" && !contains(s.folders, add) {
		s.folders = append(s.folders, add)
	}
	p, roots, depth := s.project, append([]string{}, s.folders...), s.settings.Media.SearchDepth
	s.mu.Unlock()
	r.Folders = roots
	if p == nil {
		return r
	}
	err := s.findMedia(ctx, p, roots, depth)
	info := p.Info()
	r.Info, r.Error, r.Folders = &info, errInfo(err), s.Folders()
	return r
}

func (s *Session) findMedia(ctx context.Context, p *api.Project, roots []string, depth int) error {
	left, err := p.FindMedia(ctx, api.FindRequest{Roots: roots, Depth: depth, Ask: api.AskFunc(s.ask),
		OnSearch: s.ev.search, OnEvent: s.ev.event})
	s.log.Printf("find media: %d left, err %v", len(left), err)
	return err
}

func (s *Session) ask(ctx context.Context, missing []api.MissingFile) (api.Answer, error) {
	for drained := false; !drained; {
		select {
		case <-s.answers:
		default:
			drained = true
		}
	}
	s.ev.em.Emit(EventMissing, missing)
	select {
	case a := <-s.answers:
		s.log.Printf("answer %s %q", a.Action, a.Folder)
		if a.Action == api.AnswerFolder && strings.TrimSpace(a.Folder) != "" {
			s.mu.Lock()
			if !contains(s.folders, a.Folder) {
				s.folders = append(s.folders, a.Folder)
			}
			s.mu.Unlock()
		}
		return a, nil
	case <-ctx.Done():
		return api.Answer{}, ctx.Err()
	}
}

func (s *Session) Answer(a api.Answer) {
	select {
	case s.answers <- a:
	default:
	}
}

type SyncReply struct {
	Summary *api.Summary     `json:"summary,omitempty"`
	Files   []api.FileResult `json:"files,omitempty"`
	Info    *api.ProjectInfo `json:"info,omitempty"`
	Folders []string         `json:"folders,omitempty"`
	Error   *api.ErrorInfo   `json:"error,omitempty"`
	Field   string           `json:"field,omitempty"`
}

func (s *Session) Sync(o api.Options) (r SyncReply) {
	defer s.recoverTo(&r.Error)
	if c := ValidateOptions(o); c.Error != nil {
		r.Error, r.Field = c.Error, c.Field
		return r
	}
	ctx, end, busy := s.begin("sync")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	s.mu.Lock()
	p, roots, st := s.project, append([]string{}, s.folders...), s.settings
	features, times := s.features, s.times
	s.mu.Unlock()
	if p == nil {
		r.Error = noProject()
		return r
	}
	if len(features) == 0 && p.Info().Missing > 0 {
		err := s.findMedia(ctx, p, roots, st.Media.SearchDepth)
		info := p.Info()
		r.Info, r.Folders = &info, s.Folders()
		if err != nil {
			r.Error = errInfo(err)
			return r
		}
	}
	res, err := api.Sync(ctx, p, api.SyncRequest{Options: o, Originals: st.Media.Originals,
		Features: features, Times: times}, s.ev.event)
	if err != nil {
		r.Error = errInfo(err)
		s.log.Printf("sync: %v", err)
		return r
	}
	s.mu.Lock()
	s.result, s.saved = res, ""
	s.settings.Options = o
	s.mu.Unlock()
	r.Summary, r.Files = &res.Summary, res.Files
	s.log.JSON("summary", res.Summary)
	return r
}

type PathReply struct {
	Path  string         `json:"path,omitempty"`
	Error *api.ErrorInfo `json:"error,omitempty"`
}

func (s *Session) SaveName(format string) (dir, name string) {
	s.mu.Lock()
	path, p := s.path, s.project
	s.mu.Unlock()
	ext := filepath.Ext(path)
	if p != nil {
		for _, f := range p.OutputFormats() {
			if f.Name == format && len(f.Extensions) > 0 {
				ext = f.Extensions[0]
			}
		}
	}
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return filepath.Dir(path), stem + " - synced" + ext
}

func (s *Session) ReportName() (dir, name string) {
	s.mu.Lock()
	out := s.saved
	s.mu.Unlock()
	if out == "" {
		d, n := s.SaveName("")
		out = filepath.Join(d, n)
	}
	return filepath.Dir(out), strings.TrimSuffix(filepath.Base(out), filepath.Ext(out)) + "_report.csv"
}

func (s *Session) Save(path, format string, keepPaths bool) (r PathReply) {
	defer s.recoverTo(&r.Error)
	_, end, busy := s.begin("save")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	s.mu.Lock()
	res := s.result
	s.mu.Unlock()
	if res == nil {
		r.Error = &api.ErrorInfo{Kind: ErrNoProject, Message: "no result: synchronize first"}
		return r
	}
	if err := res.Save(path, api.SaveOptions{Format: format, KeepPaths: keepPaths}, s.ev.event); err != nil {
		r.Error = errInfo(err)
		return r
	}
	s.mu.Lock()
	s.saved = path
	s.mu.Unlock()
	s.log.Printf("saved %q (%s, keep paths %v)", path, format, keepPaths)
	r.Path = path
	return r
}

func (s *Session) SaveReport(path string) (r PathReply) {
	defer s.recoverTo(&r.Error)
	s.mu.Lock()
	res := s.result
	s.mu.Unlock()
	if res == nil {
		r.Error = &api.ErrorInfo{Kind: ErrNoProject, Message: "no result: synchronize first"}
		return r
	}
	if err := res.SaveReport(path); err != nil {
		r.Error = errInfo(err)
		return r
	}
	s.log.Printf("report %q", path)
	r.Path = path
	return r
}

func (s *Session) Close() {
	s.Cancel()
	s.log.Close()
}

func noProject() *api.ErrorInfo {
	return &api.ErrorInfo{Kind: ErrNoProject, Message: "no timeline is open"}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func uniq(xs []string) []string {
	out := []string{}
	for _, x := range xs {
		if x = strings.TrimSpace(x); x != "" && !contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}
