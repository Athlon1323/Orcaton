package app

import (
	"context"
	"errors"
	"io/fs"

	"orcaton/internal/engine"
	"orcaton/internal/formats"
	"orcaton/internal/media"
	"orcaton/internal/settings"
)

type Kind string

const (
	KindCancelled  Kind = "cancelled"
	KindNoFFmpeg   Kind = "no_ffmpeg"
	KindNotFound   Kind = "not_found"
	KindBadFormat  Kind = "bad_format"
	KindBadOptions Kind = "bad_options"
	KindBadOutput  Kind = "bad_output"
	KindWrite      Kind = "write"
	KindFailed     Kind = "failed"
)

type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func wrap(k Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: k, Err: err}
}

var errUserCancel = errors.New("отменено пользователем")

func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var ae *Error
	var nf *media.FFmpegNotFound
	var oe *engine.OptionError
	var fe *formats.FormatError
	var se *settings.Error
	switch {
	case errors.As(err, &ae):
		return ae.Kind
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return KindCancelled
	case errors.As(err, &nf):
		return KindNoFFmpeg
	case errors.As(err, &oe), errors.As(err, &se):
		return KindBadOptions
	case errors.As(err, &fe):
		return KindBadFormat
	case errors.Is(err, fs.ErrNotExist):
		return KindNotFound
	}
	return KindFailed
}
