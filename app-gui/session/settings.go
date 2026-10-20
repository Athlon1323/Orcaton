package session

import (
	"errors"
	"regexp"
	"strings"

	"orcaton/api"
)

type OptionCheck struct {
	Field string         `json:"field,omitempty"`
	Error *api.ErrorInfo `json:"error,omitempty"`
}

var optionName = regexp.MustCompile(`опция ([A-Za-z]+)`)

var engineField = map[string]string{"IdleGapSec": "maxGapSec", "IdleKeepSec": "keepGapSec"}

func ValidateOptions(o api.Options) OptionCheck {
	err := o.Validate()
	if err == nil {
		return OptionCheck{}
	}
	return OptionCheck{Field: fieldOf(err), Error: errInfo(err)}
}

func fieldOf(err error) string {
	m := optionName.FindStringSubmatch(err.Error())
	if m == nil {
		return ""
	}
	if f, ok := engineField[m[1]]; ok {
		return f
	}
	return strings.ToLower(m[1][:1]) + m[1][1:]
}

func (s *Session) SaveSettings(o api.Options, keepPaths bool) (r OptionCheck) {
	defer s.recoverTo(&r.Error)
	if r = ValidateOptions(o); r.Error != nil {
		return r
	}
	s.mu.Lock()
	s.settings.Options, s.settings.Media.KeepPaths = o, keepPaths
	s.mu.Unlock()
	path := api.UserSettingsPath()
	if path == "" {
		return r
	}
	us, err := api.LoadSettings("")
	if err != nil {
		return OptionCheck{Error: errInfo(err)}
	}
	us.Options, us.Media.KeepPaths = o, keepPaths
	if err := api.SaveSettings(path, us); err != nil {
		return OptionCheck{Field: fieldOf(err), Error: errInfo(err)}
	}
	s.log.Printf("settings saved %q", path)
	return r
}

type AboutInfo struct {
	Version      string         `json:"version"`
	APIVersion   string         `json:"apiVersion"`
	FFmpeg       api.FFmpegInfo `json:"ffmpeg"`
	FFmpegError  *api.ErrorInfo `json:"ffmpegError,omitempty"`
	Cache        api.CacheInfo  `json:"cache"`
	SettingsPath string         `json:"settingsPath"`
	LogDir       string         `json:"logDir"`
}

func (s *Session) About() (r AboutInfo) {
	defer s.recoverTo(&r.FFmpegError)
	fi, err := api.GetFFmpegInfo()
	s.mu.Lock()
	dir := s.settings.Options.CacheDir
	s.mu.Unlock()
	return AboutInfo{Version: api.ProductVersion, APIVersion: api.Version, FFmpeg: fi,
		FFmpegError: errInfo(err), Cache: api.GetCacheInfo(dir), SettingsPath: api.UserSettingsPath(),
		LogDir: s.log.Dir()}
}

type CacheReply struct {
	Cleared api.CacheCleared `json:"cleared"`
	Cache   api.CacheInfo    `json:"cache"`
	Error   *api.ErrorInfo   `json:"error,omitempty"`
}

func (s *Session) ClearCache() (r CacheReply) {
	defer s.recoverTo(&r.Error)
	_, end, busy := s.begin("clear-cache")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	s.mu.Lock()
	dir := s.settings.Options.CacheDir
	s.mu.Unlock()
	r.Cleared = api.ClearCache(dir)
	r.Cache = api.GetCacheInfo(dir)
	if len(r.Cleared.Errors) > 0 {
		r.Error = errInfo(errors.New(strings.Join(r.Cleared.Errors, "; ")))
	}
	return r
}
