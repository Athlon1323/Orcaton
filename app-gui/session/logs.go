package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Log struct {
	mu  sync.Mutex
	dir string
	f   *os.File
}

func DefaultLogDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "orcaton", "logs")
}

func OpenLog(dir string) *Log {
	l := &Log{dir: dir}
	if dir == "" {
		return l
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return l
	}
	name := filepath.Join(dir, "gui-"+time.Now().Format("2006-01-02")+".log")
	if f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		l.f = f
	}
	return l
}

func (l *Log) Dir() string {
	if l == nil {
		return ""
	}
	return l.dir
}

func (l *Log) Printf(format string, a ...any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		fmt.Fprintf(l.f, "%s %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, a...))
	}
}

func (l *Log) JSON(kind string, v any) {
	if l == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		l.Printf("%s %v", kind, v)
		return
	}
	l.Printf("%s %s", kind, b)
}

func (l *Log) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		l.f.Close()
		l.f = nil
	}
}
