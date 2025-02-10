package log

import (
	"errors"
	"io"
	"sync"

	"github.com/go-logr/logr"
	logLib "github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
)

// CombinedLogger implements the Logger interface and delegates logging to multiple loggers
type CombinedLogger struct {
	loggers []logLib.Logger
	m       sync.Mutex
	level   logrus.Level
}

// NewCombinedLogger creates a new CombinedLogger
func NewCombinedLogger(level logrus.Level, loggers ...logLib.Logger) *CombinedLogger {
	return &CombinedLogger{
		loggers: loggers,
		level:   level,
	}
}

// log is a helper to execute a function for all loggers at the appropriate log level
func (c *CombinedLogger) log(level logrus.Level, logFunc func(logLib.Logger)) {
	c.m.Lock()
	defer c.m.Unlock()

	if level < c.level {
		return
	}

	for _, logger := range c.loggers {
		logFunc(logger)
	}
}

func (c *CombinedLogger) Debug(args ...interface{}) {
	c.log(logrus.DebugLevel, func(logger logLib.Logger) {
		logger.Debug(args...)
	})
}

func (c *CombinedLogger) Debugf(format string, args ...interface{}) {
	c.log(logrus.DebugLevel, func(logger logLib.Logger) {
		logger.Debugf(format, args...)
	})
}

func (c *CombinedLogger) Info(args ...interface{}) {
	c.log(logrus.InfoLevel, func(logger logLib.Logger) {
		logger.Info(args...)
	})
}

func (c *CombinedLogger) Infof(format string, args ...interface{}) {
	c.log(logrus.InfoLevel, func(logger logLib.Logger) {
		logger.Infof(format, args...)
	})
}

func (c *CombinedLogger) Warn(args ...interface{}) {
	c.log(logrus.WarnLevel, func(logger logLib.Logger) {
		logger.Warn(args...)
	})
}

func (c *CombinedLogger) Warnf(format string, args ...interface{}) {
	c.log(logrus.WarnLevel, func(logger logLib.Logger) {
		logger.Warnf(format, args...)
	})
}

func (c *CombinedLogger) Error(args ...interface{}) {
	c.log(logrus.ErrorLevel, func(logger logLib.Logger) {
		logger.Error(args...)
	})
}

func (c *CombinedLogger) Errorf(format string, args ...interface{}) {
	c.log(logrus.ErrorLevel, func(logger logLib.Logger) {
		logger.Errorf(format, args...)
	})
}

func (c *CombinedLogger) Fatal(args ...interface{}) {
	c.log(logrus.FatalLevel, func(logger logLib.Logger) {
		logger.Fatal(args...)
	})
}

func (c *CombinedLogger) Fatalf(format string, args ...interface{}) {
	c.log(logrus.FatalLevel, func(logger logLib.Logger) {
		logger.Fatalf(format, args...)
	})
}

func (c *CombinedLogger) Done(args ...interface{}) {
	c.log(logrus.InfoLevel, func(logger logLib.Logger) {
		logger.Done(args...)
	})
}

func (c *CombinedLogger) Donef(format string, args ...interface{}) {
	c.log(logrus.InfoLevel, func(logger logLib.Logger) {
		logger.Donef(format, args...)
	})
}

func (c *CombinedLogger) Print(level logrus.Level, args ...interface{}) {
	c.log(level, func(logger logLib.Logger) {
		logger.Print(level, args...)
	})
}

func (c *CombinedLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	c.log(level, func(logger logLib.Logger) {
		logger.Printf(level, format, args...)
	})
}

func (c *CombinedLogger) SetLevel(level logrus.Level) {
	c.m.Lock()
	defer c.m.Unlock()

	c.level = level
	for _, logger := range c.loggers {
		logger.SetLevel(level)
	}
}

func (c *CombinedLogger) GetLevel() logrus.Level {
	c.m.Lock()
	defer c.m.Unlock()

	return c.level
}

func (c *CombinedLogger) WriteString(level logrus.Level, message string) {
	c.log(level, func(logger logLib.Logger) {
		logger.WriteString(level, message)
	})
}

func (c *CombinedLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", errors.New("questions in combined logger not supported")
}

func (c *CombinedLogger) ErrorStreamOnly() logLib.Logger {
	return nil
}

func (c *CombinedLogger) LogrLogSink() logr.LogSink {
	return nil
}

func (c *CombinedLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	c.m.Lock()
	defer c.m.Unlock()

	var writers []io.WriteCloser
	for _, logger := range c.loggers {
		writer := logger.Writer(level, raw)
		if writer != nil {
			writers = append(writers, writer)
		}
	}
	return &multiWriter{writers: writers}
}

type multiWriter struct {
	writers []io.WriteCloser
}

func (m *multiWriter) Write(p []byte) (int, error) {
	for _, w := range m.writers {
		if _, err := w.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (m *multiWriter) Close() error {
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}
