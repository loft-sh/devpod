package log

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
)

var Discard = NewDiscardLogger(logrus.InfoLevel)

type discardLogger struct {
	m sync.Mutex

	level logrus.Level
}

// NewDiscardLogger returns a logger instance for the
func NewDiscardLogger(level logrus.Level) Logger {
	newLogger := &discardLogger{
		level: level,
	}

	newLogger.SetLevel(level)
	return newLogger
}

func (f *discardLogger) Debug(args ...interface{}) {}

func (f *discardLogger) Debugf(format string, args ...interface{}) {}

func (f *discardLogger) Info(args ...interface{}) {}

func (f *discardLogger) Infof(format string, args ...interface{}) {}

func (f *discardLogger) Warn(args ...interface{}) {}

func (f *discardLogger) Warnf(format string, args ...interface{}) {}

func (f *discardLogger) Error(args ...interface{}) {}

func (f *discardLogger) Errorf(format string, args ...interface{}) {}

func (f *discardLogger) Fatal(args ...interface{}) {
	os.Exit(1)
}

func (f *discardLogger) Fatalf(format string, args ...interface{}) {
	os.Exit(1)
}

func (f *discardLogger) Done(args ...interface{}) {}

func (f *discardLogger) Donef(format string, args ...interface{}) {}

func (f *discardLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		f.Info(args...)
	case logrus.DebugLevel:
		f.Debug(args...)
	case logrus.WarnLevel:
		f.Warn(args...)
	case logrus.ErrorLevel:
		f.Error(args...)
	case logrus.FatalLevel:
		f.Fatal(args...)
	case logrus.PanicLevel:
		f.Fatal(args...)
	case logrus.TraceLevel:
		f.Debug(args...)
	}
}

func (f *discardLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		f.Infof(format, args...)
	case logrus.DebugLevel:
		f.Debugf(format, args...)
	case logrus.WarnLevel:
		f.Warnf(format, args...)
	case logrus.ErrorLevel:
		f.Errorf(format, args...)
	case logrus.FatalLevel:
		f.Fatalf(format, args...)
	case logrus.PanicLevel:
		f.Fatalf(format, args...)
	case logrus.TraceLevel:
		f.Debugf(format, args...)
	}
}

func (f *discardLogger) StartWait(message string) {
	// Noop operation
}

func (f *discardLogger) StopWait() {
	// Noop operation
}

func (f *discardLogger) SetLevel(level logrus.Level) {
	f.m.Lock()
	defer f.m.Unlock()

	f.level = level
}

func (f *discardLogger) GetLevel() logrus.Level {
	f.m.Lock()
	defer f.m.Unlock()

	return f.level
}

func (f *discardLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	f.m.Lock()
	defer f.m.Unlock()

	return &NopCloser{io.Discard}
}

func (f *discardLogger) Write(message []byte) (int, error) {
	return len(message), nil
}

func (f *discardLogger) WriteString(level logrus.Level, message string) {

}

func (f *discardLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", errors.New("questions in file logger not supported")
}

// WithLevel implements logger interface
func (f *discardLogger) WithLevel(level logrus.Level) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := discardLogger{
		m:     sync.Mutex{},
		level: level,
	}
	return &n
}

func (f *discardLogger) ErrorStreamOnly() Logger {
	return f
}

// --- Logr LogSink ---

type discordLogSink struct{}

var _ logr.LogSink = discordLogSink{}

// Enabled implements logr.LogSink.
func (discordLogSink) Enabled(level int) bool { return false }

// Error implements logr.LogSink.
func (discordLogSink) Error(err error, msg string, keysAndValues ...interface{}) {}

// Info implements logr.LogSink.
func (discordLogSink) Info(level int, msg string, keysAndValues ...interface{}) {}

// Init implements logr.LogSink.
func (discordLogSink) Init(info logr.RuntimeInfo) {}

// WithName implements logr.LogSink.
func (a discordLogSink) WithName(name string) logr.LogSink { return a }

// WithValues implements logr.LogSink.
func (a discordLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink { return a }

func (f *discardLogger) LogrLogSink() logr.LogSink { return discordLogSink{} }
