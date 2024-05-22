package log

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
)

// logFunctionType type
type logFunctionType uint32

const (
	fatalFn logFunctionType = iota
	errorFn
	warnFn
	infoFn
	debugFn
	doneFn
)

// BaseLogger defines the common logging interface
type BaseLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Done(args ...interface{})
	Donef(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	Print(level logrus.Level, args ...interface{})
	Printf(level logrus.Level, format string, args ...interface{})

	SetLevel(level logrus.Level)
	GetLevel() logrus.Level

	LogrLogSink() logr.LogSink
}

type SimpleLogger interface {
	Infof(format string, args ...interface{})
}

// Logger defines the devspace common logging interface
type Logger interface {
	BaseLogger

	Question(params *survey.QuestionOptions) (string, error)
	ErrorStreamOnly() Logger

	Writer(level logrus.Level, raw bool) io.WriteCloser
	WriteString(level logrus.Level, message string)
}
