package log

import (
	"io"

	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/utils/pkg/log"
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

// Logger defines the devspace common logging interface
type Logger interface {
	log.Logger

	Question(params *survey.QuestionOptions) (string, error)
	ErrorStreamOnly() Logger

	Writer(level logrus.Level, raw bool) io.WriteCloser
	WriteString(level logrus.Level, message string)
}
