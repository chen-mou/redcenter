package log

import (
	"io"
	"io/fs"
	"os"
	"time"
)

type Logger interface {
	Info(string)
	Error(string)
	Warn(string)
}

type FileLogger struct {
	Path     string
	NameRule func([]string) string
	Time     time.Duration
	Props    []string
	writer   io.Writer
}

type DefaultLogger struct {
	Type string
}

var Default DefaultLogger

func (logger FileLogger) Info(msg string) {
	os.OpenFile("", os.O_APPEND, fs.ModeAppend)
}
