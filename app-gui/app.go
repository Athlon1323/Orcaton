package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"orcaton-gui/session"
	"orcaton/api"
)

type App struct {
	ctx context.Context
	s   *session.Session
	log *session.Log
}

func NewApp() *App {
	a := &App{log: session.OpenLog(session.DefaultLogDir())}
	a.s = session.New(emitter{a}, a.log)
	return a
}

type emitter struct{ a *App }

func (e emitter) Emit(name string, data any) {
	if e.a.ctx != nil {
		runtime.EventsEmit(e.a.ctx, name, data)
	}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) beforeClose(ctx context.Context) bool {
	a.s.Cancel()
	return false
}

func (a *App) shutdown(ctx context.Context) { a.s.Close() }

func (a *App) Startup() session.StartupInfo { return a.s.Startup() }

func (a *App) CheckFFmpeg() *api.ErrorInfo { return a.s.CheckFFmpeg() }

func (a *App) FFmpegPlan() session.FFmpegPlan { return a.s.FFmpegPlan() }

func (a *App) InstallFFmpeg() session.InstallReply { return a.s.InstallFFmpeg() }

func (a *App) SetLanguage(lang string, d session.DialogTexts) *api.ErrorInfo {
	return a.s.SetLanguage(lang, d)
}

func (a *App) ChooseTimeline() *session.ProjectReply {
	d := a.s.Dialogs()
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: d.OpenTimeline,
		Filters: []runtime.FileFilter{
			{DisplayName: d.TimelineFilter, Pattern: "*.xml;*.drt"},
			{DisplayName: d.AllFiles, Pattern: "*.*"},
		},
	})
	if err != nil {
		return &session.ProjectReply{Folders: a.s.Folders(), Error: a.dialogError(err)}
	}
	if path == "" {
		return nil
	}
	r := a.s.Open(path)
	return &r
}

func (a *App) Open(path string) session.ProjectReply { return a.s.Open(path) }

func (a *App) ChooseFolder() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: a.s.Dialogs().ChooseFolder,
		CanCreateDirectories: false})
	if err != nil {
		a.log.Printf("folder dialog: %v", err)
		return ""
	}
	return dir
}

func (a *App) AddFolder() *session.ProjectReply {
	dir := a.ChooseFolder()
	if dir == "" {
		return nil
	}
	r := a.s.FindMedia(dir)
	return &r
}

func (a *App) RemoveFolder(dir string) []string { return a.s.RemoveFolder(dir) }

func (a *App) Locate() session.ProjectReply { return a.s.FindMedia("") }

func (a *App) Answer(ans api.Answer) { a.s.Answer(ans) }

func (a *App) Sync(o api.Options) session.SyncReply { return a.s.Sync(o) }

func (a *App) Cancel() { a.s.Cancel() }

func (a *App) SaveTimeline(format string, keepPaths bool) *session.PathReply {
	dir, name := a.s.SaveName(format)
	path, err := a.saveDialog(a.s.Dialogs().SaveTimeline, dir, name)
	if err != nil {
		return &session.PathReply{Error: a.dialogError(err)}
	}
	if path == "" {
		return nil
	}
	r := a.s.Save(path, format, keepPaths)
	return &r
}

func (a *App) SaveReport() *session.PathReply {
	dir, name := a.s.ReportName()
	path, err := a.saveDialog(a.s.Dialogs().SaveReport, dir, name)
	if err != nil {
		return &session.PathReply{Error: a.dialogError(err)}
	}
	if path == "" {
		return nil
	}
	r := a.s.SaveReport(path)
	return &r
}

func (a *App) saveDialog(title, dir, name string) (string, error) {
	ext := filepath.Ext(name)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		dir = ""
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: title, DefaultDirectory: dir, DefaultFilename: name, CanCreateDirectories: true,
		Filters: []runtime.FileFilter{{DisplayName: "*" + ext, Pattern: "*" + ext}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if filepath.Ext(path) == "" {
		path += ext
	}
	return path, nil
}

func (a *App) ShowInFolder(path string) {
	name, args := session.RevealCommand(goruntime.GOOS, path)
	a.start(name, args)
}

func (a *App) OpenLogs() {
	dir := a.log.Dir()
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	name, args := session.OpenCommand(goruntime.GOOS, dir)
	a.start(name, args)
}

func (a *App) CopyText(text string) {
	if err := runtime.ClipboardSetText(a.ctx, text); err != nil {
		a.log.Printf("clipboard: %v", err)
	}
}

func (a *App) start(name string, args []string) {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		a.log.Printf("start %s %v: %v", name, args, err)
		return
	}
	go func() { _ = cmd.Wait() }()
}

func (a *App) ValidateOptions(o api.Options) session.OptionCheck { return session.ValidateOptions(o) }

func (a *App) SaveSettings(o api.Options, keepPaths bool) session.OptionCheck {
	return a.s.SaveSettings(o, keepPaths)
}

func (a *App) About() session.AboutInfo { return a.s.About() }

func (a *App) ClearCache() session.CacheReply { return a.s.ClearCache() }

func (a *App) CLITool() session.CLITool { return a.s.CLITool() }

func (a *App) InstallCLITool(prompt string) session.CLIToolReply { return a.s.InstallCLITool(prompt) }

func (a *App) RemoveCLITool(prompt string) session.CLIToolReply { return a.s.RemoveCLITool(prompt) }

func (a *App) dialogError(err error) *api.ErrorInfo {
	a.log.Printf("dialog: %v", err)
	return &api.ErrorInfo{Kind: api.ErrFailed, Message: "OS dialog: " + err.Error()}
}
