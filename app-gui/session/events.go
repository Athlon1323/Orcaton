package session

import (
	"fmt"
	"runtime/debug"

	"orcaton/api"
)

const (
	EventSync    = "sync:event"
	EventSearch  = "media:search"
	EventMissing = "media:missing"
)

type Emitter interface {
	Emit(name string, data any)
}

const ErrBusy api.ErrorKind = "busy"

const ErrNoProject api.ErrorKind = "no_project"

func errInfo(err error) *api.ErrorInfo { return api.Describe(err) }

func (s *Session) recoverTo(dst **api.ErrorInfo) {
	if r := recover(); r != nil {
		s.log.Printf("panic: %v\n%s", r, debug.Stack())
		*dst = &api.ErrorInfo{Kind: api.ErrFailed, Message: fmt.Sprintf("internal window error: %v", r),
			Hint: "Copy the details (Copy details) and the log from Developer tools."}
	}
}

type emitter struct {
	em  Emitter
	log *Log
}

func (e emitter) event(ev api.Event) {
	e.log.JSON("event", ev)
	e.em.Emit(EventSync, ev)
}

func (e emitter) search(r api.SearchReport) {
	e.log.JSON("search", r)
	e.em.Emit(EventSearch, r)
}
