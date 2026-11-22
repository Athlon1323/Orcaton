package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"orcaton/api"
)

var Languages = []string{"en", "de", "ru", "ka"}

type DialogTexts struct {
	OpenTimeline   string `json:"openTimeline"`
	TimelineFilter string `json:"timelineFilter"`
	AllFiles       string `json:"allFiles"`
	ChooseFolder   string `json:"chooseFolder"`
	SaveTimeline   string `json:"saveTimeline"`
	SaveReport     string `json:"saveReport"`
}

var defaultDialogs = DialogTexts{
	OpenTimeline:   "Open a timeline",
	TimelineFilter: "Timeline (*.xml, *.drt)",
	AllFiles:       "All files (*.*)",
	ChooseFolder:   "Choose a folder",
	SaveTimeline:   "Save synchronized timeline",
	SaveReport:     "Save report",
}

func (d DialogTexts) withDefaults() DialogTexts {
	pick := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	return DialogTexts{
		OpenTimeline:   pick(d.OpenTimeline, defaultDialogs.OpenTimeline),
		TimelineFilter: pick(d.TimelineFilter, defaultDialogs.TimelineFilter),
		AllFiles:       pick(d.AllFiles, defaultDialogs.AllFiles),
		ChooseFolder:   pick(d.ChooseFolder, defaultDialogs.ChooseFolder),
		SaveTimeline:   pick(d.SaveTimeline, defaultDialogs.SaveTimeline),
		SaveReport:     pick(d.SaveReport, defaultDialogs.SaveReport),
	}
}

func PrefsPath() string {
	p := api.UserSettingsPath()
	if p == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(p), "gui.json")
}

func loadPrefs(path string) (map[string]json.RawMessage, error) {
	m := map[string]json.RawMessage{}
	if path == "" {
		return m, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]json.RawMessage{}, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

func (s *Session) savedLanguage() string {
	m, err := loadPrefs(PrefsPath())
	if err != nil {
		s.log.Printf("gui.json: %v", err)
	}
	var lang string
	if raw, ok := m["language"]; ok && json.Unmarshal(raw, &lang) == nil && slices.Contains(Languages, lang) {
		return lang
	}
	return ""
}

func (s *Session) SetLanguage(lang string, d DialogTexts) *api.ErrorInfo {
	if !slices.Contains(Languages, lang) {
		return &api.ErrorInfo{Kind: api.ErrBadOptions, Message: "unknown window language: " + lang}
	}
	s.mu.Lock()
	s.dialogs = d.withDefaults()
	s.mu.Unlock()
	path := PrefsPath()
	if path == "" {
		return nil
	}
	m, err := loadPrefs(path)
	if err != nil {
		s.log.Printf("gui.json: %v — overwriting", err)
	}
	raw, _ := json.Marshal(lang)
	m["language"] = raw
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &api.ErrorInfo{Kind: api.ErrWrite, Message: err.Error()}
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return &api.ErrorInfo{Kind: api.ErrWrite, Message: err.Error()}
	}
	return nil
}

func (s *Session) Dialogs() DialogTexts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dialogs.withDefaults()
}
