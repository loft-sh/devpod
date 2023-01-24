package log

import (
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/utils/pkg/log"
	"github.com/sirupsen/logrus"
	"io"
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

// Logger defines the devspace common logging interface
type Logger interface {
	log.Logger
	// WithLevel creates a new logger with the given level
	WithLevel(level logrus.Level) Logger
	Question(params *survey.QuestionOptions) (string, error)
	ErrorStreamOnly() Logger
	WithPrefix(prefix string) Logger
	WithPrefixColor(prefix, color string) Logger
	WithSink(sink Logger) Logger
	AddSink(sink Logger)

	Writer(level logrus.Level, raw bool) io.WriteCloser
	WriteString(level logrus.Level, message string)
}
