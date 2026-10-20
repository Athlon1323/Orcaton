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
	ErrNoFFmpeg:   "Установите FFmpeg или укажите папку с ним в переменной ORCATON_FFMPEG_DIR.",
	ErrNotFound:   "Проверьте путь; если диск внешний — что он подключён и доступен программе.",
	ErrBadFormat:  "Нужен таймлайн FCP7 XML (.xml) или .drt.",
	ErrBadOptions: "Проверьте настройки синхронизации.",
	ErrBadOutput:  "Выберите другой формат выхода: .drt — только для входа .drt.",
	ErrWrite:      "Проверьте, что папка существует, доступна для записи и файл не открыт в другой программе.",
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
