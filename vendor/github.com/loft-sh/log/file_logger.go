package log

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/acarl005/stripansi"
	"github.com/go-logr/logr"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type fileLogger struct {
	logger *logrus.Logger

	m     *sync.Mutex
	level logrus.Level
	// sinks    []Logger
	prefixes []string
}

var _ Logger = &fileLogger{}

// NewFileLogger returns a logger instance for the specified filename
func NewFileLogger(logFile string, level logrus.Level) Logger {
	newLogger := &fileLogger{
		logger: logrus.New(),
		m:      &sync.Mutex{},
	}
	newLogger.logger.Formatter = &logrus.JSONFormatter{}
	newLogger.logger.SetOutput(&lumberjack.Logger{
		Filename:   logFile,
		MaxAge:     12,
		MaxBackups: 4,
		MaxSize:    10 * 1024 * 1024,
	})

	newLogger.SetLevel(level)
	return newLogger
}

func (f *fileLogger) addPrefixes(message string) string {
	prefix := ""
	for _, p := range f.prefixes {
		prefix += p
	}

	return prefix + message
}

func (f *fileLogger) Debug(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.DebugLevel {
		return
	}

	f.logger.Debug(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Debugf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.DebugLevel {
		return
	}

	f.logger.Debugf(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Info(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Infof(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Warn(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.WarnLevel {
		return
	}

	f.logger.Warn(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Warnf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.WarnLevel {
		return
	}

	f.logger.Warn(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Error(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.ErrorLevel {
		return
	}

	f.logger.Error(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Errorf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.ErrorLevel {
		return
	}

	f.logger.Error(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Fatal(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.FatalLevel {
		return
	}

	f.logger.Fatal(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Fatalf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.FatalLevel {
		return
	}

	f.logger.Fatal(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Done(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Donef(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Print(level logrus.Level, args ...interface{}) {
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

func (f *fileLogger) Printf(level logrus.Level, format string, args ...interface{}) {
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

func (f *fileLogger) StartWait(message string) {
	// Noop operation
}

func (f *fileLogger) StopWait() {
	// Noop operation
}

func (f *fileLogger) SetLevel(level logrus.Level) {
	f.m.Lock()
	defer f.m.Unlock()

	f.level = level
}

func (f *fileLogger) GetLevel() logrus.Level {
	f.m.Lock()
	defer f.m.Unlock()

	return f.level
}

func (f *fileLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < level {
		return &NopCloser{io.Discard}
	}

	return &NopCloser{f}
}

func (f *fileLogger) Write(message []byte) (int, error) {
	return f.logger.Out.Write(message)
}

func (f *fileLogger) WriteString(level logrus.Level, message string) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	_, _ = f.logger.Out.Write([]byte(stripEscapeSequences(message)))
}

func stripEscapeSequences(str string) string {
	return stripansi.Strip(strings.TrimSpace(str))
}

func (f *fileLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", errors.New("questions in file logger not supported")
}

// WithLevel implements logger interface
func (f *fileLogger) WithLevel(level logrus.Level) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.level = level
	return &n
}

func (f *fileLogger) WithPrefix(prefix string) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.prefixes = append(n.prefixes, prefix)
	return &n
}

func (f *fileLogger) WithPrefixColor(prefix, color string) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.prefixes = append(n.prefixes, prefix)
	return &n
}

func (f *fileLogger) ErrorStreamOnly() Logger {
	return f
}

// --- Logr LogSink ---

type fileLogSink struct {
	logger        *fileLogger
	name          string
	keysAndValues []interface{}
}

var _ logr.LogSink = &fileLogSink{}

// Enabled implements logr.LogSink.
func (s *fileLogSink) Enabled(level int) bool {
	// if the logrus level is debug or trace, we always log
	if s.logger.level > logrus.InfoLevel {
		return true
	}

	// if the logr level is 0, we log if the logrus level is info or higher
	return s.logger.level <= logrus.InfoLevel && level == 0
}

// Error implements logr.LogSink.
func (s *fileLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	s.logger.WithPrefix(s.name).Error(err, msg, append(s.keysAndValues, keysAndValues...))
}

// Info implements logr.LogSink.
func (s *fileLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	if level == 0 {
		s.logger.WithPrefix(s.name).Info(msg, append(s.keysAndValues, keysAndValues...))
	} else {
		s.logger.WithPrefix(s.name).Debug(msg, append(s.keysAndValues, keysAndValues...))
	}
}

// Init implements logr.LogSink.
func (*fileLogSink) Init(info logr.RuntimeInfo) {}

// WithName implements logr.LogSink.
func (s *fileLogSink) WithName(name string) logr.LogSink {
	if s.name != "" {
		name = s.name + "." + name
	}

	return &fileLogSink{
		logger:        s.logger,
		name:          name,
		keysAndValues: s.keysAndValues,
	}
}

// WithValues implements logr.LogSink.
func (s *fileLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &fileLogSink{
		logger:        s.logger,
		name:          s.name,
		keysAndValues: append(s.keysAndValues, keysAndValues...),
	}
}

// LogrLogSink implements Logger.
func (f *fileLogger) LogrLogSink() logr.LogSink {
	return &fileLogSink{
		logger:        f,
		name:          "",
		keysAndValues: []interface{}{},
	}
}
