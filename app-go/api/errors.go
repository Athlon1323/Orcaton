package api

import (
	"orcaton/internal/app"
	"orcaton/internal/media"
)

type ErrorKind string

const (
	ErrCancelled  ErrorKind = ErrorKind(app.KindCancelled)
	ErrNoFFmpeg   ErrorKind = ErrorKind(app.KindNoFFmpeg)
	ErrNotFound   ErrorKind = ErrorKind(app.KindNotFound)
	ErrBadFormat  ErrorKind = ErrorKind(app.KindBadFormat)
	ErrBadOptions ErrorKind = ErrorKind(app.KindBadOptions)
	ErrBadOutput  ErrorKind = ErrorKind(app.KindBadOutput)
	ErrWrite      ErrorKind = ErrorKind(app.KindWrite)
	ErrFailed     ErrorKind = ErrorKind(app.KindFailed)
)

func KindOf(err error) ErrorKind { return ErrorKind(app.KindOf(err)) }

type ErrorInfo struct {
	Kind    ErrorKind `json:"kind"`
	Message string    `json:"message"`
	Hint    string    `json:"hint,omitempty"`
}

var hints = map[ErrorKind]string{
	ErrNoFFmpeg:   "Install FFmpeg or set the folder with it in the ORCATON_FFMPEG_DIR variable.",
	ErrNotFound:   "Check the path; if the drive is external, make sure it is connected and accessible to the program.",
	ErrBadFormat:  "An FCP7 XML (.xml) or .drt timeline is required.",
	ErrBadOptions: "Check the synchronization settings.",
	ErrBadOutput:  "Choose another output format: .drt is only for .drt input.",
	ErrWrite:      "Check that the folder exists and is writable, and that the file is not open in another program.",
}

func Describe(err error) *ErrorInfo {
	if err == nil {
		return nil
	}
	k := KindOf(err)
	return &ErrorInfo{Kind: k, Message: err.Error(), Hint: hints[k]}
}

func FFmpegAvailable() error {
	for _, n := range []string{"ffmpeg", "ffprobe"} {
		if _, err := media.Bin(n); err != nil {
			return err
		}
	}
	return nil
}
