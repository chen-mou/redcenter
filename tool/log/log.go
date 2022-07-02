package log

import (
	"fmt"
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

func (log DefaultLogger) base(typ string, msg string) {
	fmt.Println("[" + typ + "]" + msg + "\t" + time.Now().Format("2006-01-02 15:04:05"))
}

func (log DefaultLogger) Info(msg string) {
	log.base("INFO", msg)
}

func (log DefaultLogger) Error(msg string) {
	log.base("ERROR", msg)
}

func (log DefaultLogger) Warn(msg string) {
	log.base("WARN", msg)
}
