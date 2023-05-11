package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	goansi "github.com/k0kubun/go-ansi"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// var startTime = time.Now()

var Default = NewStdoutLogger(os.Stdin, stdout, stderr, logrus.InfoLevel)

var Colors = []string{
	"blue",
	"blue+h",
	"blue+b",
	"green",
	"green+h",
	"green+b",
	"yellow",
	"yellow+h",
	"yellow+b",
	"magenta",
	"magenta+h",
	"magenta+b",
	"cyan",
	"cyan+h",
	"cyan+b",
	"white",
	"white+h",
	"white+b",
}

var stdout = goansi.NewAnsiStdout()
var stderr = goansi.NewAnsiStderr()

type Format int

const (
	TextFormat Format = iota
	TimeFormat Format = iota
	JSONFormat Format = iota
	RawFormat  Format = iota
)

func NewStdoutLogger(stdin io.Reader, stdout, stderr io.Writer, level logrus.Level) *StreamLogger {
	return &StreamLogger{
		m:           &sync.Mutex{},
		level:       level,
		format:      TextFormat,
		isTerminal:  terminal.IsTerminal(stdin),
		stream:      stdout,
		errorStream: stderr,
		survey:      survey.NewSurvey(),
	}
}

func NewStreamLogger(stdout, stderr io.Writer, level logrus.Level) *StreamLogger {
	return &StreamLogger{
		m:           &sync.Mutex{},
		level:       level,
		format:      TextFormat,
		isTerminal:  false,
		stream:      stdout,
		errorStream: stderr,
	}
}

func NewStreamLoggerWithFormat(stdout, stderr io.Writer, level logrus.Level, format Format) *StreamLogger {
	return &StreamLogger{
		m:           &sync.Mutex{},
		level:       level,
		isTerminal:  false,
		format:      format,
		stream:      stdout,
		errorStream: stderr,
	}
}

type StreamLogger struct {
	m     *sync.Mutex
	level logrus.Level

	prefixes []Prefix

	format      Format
	isTerminal  bool
	stream      io.Writer
	errorStream io.Writer

	survey survey.Survey

	sinks []Logger
}

type Prefix struct {
	Prefix string
	Color  string
}

type Line struct {
	// Time is when this log message occurred
	Time time.Time `json:"time,omitempty"`

	// Message is when the message of the log message
	Message string `json:"message,omitempty"`

	// Level is the log level this message has used
	Level logrus.Level `json:"level,omitempty"`
}

type fnTypeInformation struct {
	tag      string
	color    string
	logLevel logrus.Level
}

var fnTypeInformationMap = map[logFunctionType]*fnTypeInformation{
	debugFn: {
		tag:      "debug ",
		color:    "green+b",
		logLevel: logrus.DebugLevel,
	},
	infoFn: {
		tag:      "info ",
		color:    "cyan+b",
		logLevel: logrus.InfoLevel,
	},
	warnFn: {
		tag:      "warn ",
		color:    "red+b",
		logLevel: logrus.WarnLevel,
	},
	errorFn: {
		tag:      "error ",
		color:    "red+b",
		logLevel: logrus.ErrorLevel,
	},
	fatalFn: {
		tag:      "fatal ",
		color:    "red+b",
		logLevel: logrus.FatalLevel,
	},
	doneFn: {
		tag:      "done ",
		color:    "green+b",
		logLevel: logrus.InfoLevel,
	},
}

func formatInt(i int) string {
	formatted := strconv.Itoa(i)
	if len(formatted) == 1 {
		formatted = "0" + formatted
	}
	return formatted
}

func (s *StreamLogger) GetFormat() Format {
	s.m.Lock()
	defer s.m.Unlock()

	return s.format
}

func (s *StreamLogger) SetFormat(format Format) {
	s.m.Lock()
	defer s.m.Unlock()

	s.format = format
}

func (s *StreamLogger) ErrorStreamOnly() Logger {
	s.m.Lock()
	defer s.m.Unlock()

	n := *s
	n.m = &sync.Mutex{}
	n.stream = s.errorStream
	return &n
}

func (s *StreamLogger) MakeRaw() {
	s.m.Lock()
	defer s.m.Unlock()

	s.format = RawFormat
}

func (s *StreamLogger) WithPrefix(prefix string) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	hashNumber := int(hash.StringToNumber(prefix))
	if hashNumber < 0 {
		hashNumber = hashNumber * -1
	}

	n := *s
	n.m = &sync.Mutex{}
	n.prefixes = []Prefix{}
	n.prefixes = append(n.prefixes, s.prefixes...)
	n.prefixes = append(n.prefixes, Prefix{
		Prefix: prefix,
		Color:  Colors[hashNumber%len(Colors)],
	})
	return &n
}

func (s *StreamLogger) WithPrefixColor(prefix, color string) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	n := *s
	n.m = &sync.Mutex{}
	n.prefixes = []Prefix{}
	n.prefixes = append(n.prefixes, s.prefixes...)
	n.prefixes = append(n.prefixes, Prefix{
		Prefix: prefix,
		Color:  color,
	})
	return &n
}

func (s *StreamLogger) AddSink(log Logger) {
	s.m.Lock()
	defer s.m.Unlock()

	s.sinks = append(s.sinks, log)
}

func (s *StreamLogger) WithSink(log Logger) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	n := *s
	n.m = &sync.Mutex{}
	n.sinks = []Logger{}
	n.sinks = append(n.sinks, s.sinks...)
	n.sinks = append(n.sinks, log)
	return &n
}

func (s *StreamLogger) WithLevel(level logrus.Level) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	n := *s
	n.m = &sync.Mutex{}
	n.level = level
	return &n
}

func (s *StreamLogger) getStream(level logrus.Level) io.Writer {
	if level <= logrus.WarnLevel {
		return s.errorStream
	}

	return s.stream
}

func (s *StreamLogger) writePrefixes(message string) string {
	prefix := ""
	for _, prefixDef := range s.prefixes {
		if prefixDef.Color != "" {
			prefix += ansi.Color(prefixDef.Prefix, prefixDef.Color)
		} else {
			prefix += prefixDef.Prefix
		}
	}

	return prefix + message
}

func (s *StreamLogger) writeMessage(fnType logFunctionType, message string) {
	fnInformation := fnTypeInformationMap[fnType]
	message = s.writePrefixes(message)
	for _, s := range s.sinks {
		if fnInformation.logLevel == logrus.PanicLevel || fnInformation.logLevel == logrus.FatalLevel {
			s.Print(logrus.ErrorLevel, message)
		} else {
			s.Print(fnInformation.logLevel, message)
		}
	}

	if s.level >= fnInformation.logLevel {
		stream := s.getStream(fnInformation.logLevel)
		if s.format == RawFormat {
			_, _ = stream.Write([]byte(message))
		} else if s.format == TimeFormat {
			now := time.Now()
			_, _ = stream.Write([]byte(ansi.Color(formatInt(now.Hour())+":"+formatInt(now.Minute())+":"+formatInt(now.Second())+" ", "white+b")))
			_, _ = stream.Write([]byte(message))
		} else if s.format == TextFormat {
			now := time.Now()
			_, _ = stream.Write([]byte(ansi.Color(formatInt(now.Hour())+":"+formatInt(now.Minute())+":"+formatInt(now.Second())+" ", "white+b")))
			_, _ = stream.Write([]byte(ansi.Color(fnInformation.tag, fnInformation.color)))
			_, _ = stream.Write([]byte(message))
		} else if s.format == JSONFormat {
			s.writeJSON(message, fnInformation.logLevel)
		}
	}
}

func (s *StreamLogger) JSON(level logrus.Level, value interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level >= level && s.format == JSONFormat {
		stream := s.getStream(level)
		line, err := json.Marshal(value)
		if err == nil {
			_, _ = stream.Write([]byte(string(line) + "\n"))
		}
	}
}

func (s *StreamLogger) writeJSON(message string, level logrus.Level) {
	if message != "" {
		stream := s.getStream(level)
		line, err := json.Marshal(&Line{
			Time:    time.Now(),
			Message: stripansi.Strip(strings.TrimSpace(message)),
			Level:   level,
		})
		if err == nil {
			_, _ = stream.Write([]byte(string(line) + "\n"))
		}
	}
}

func (s *StreamLogger) Debug(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(debugFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Debugf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(debugFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Children() []Logger {
	return nil
}

func (s *StreamLogger) Info(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(infoFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Infof(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(infoFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Warn(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(warnFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Warnf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(warnFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Error(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(errorFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Errorf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(errorFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Fatal(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintln(args...)

	s.writeMessage(fatalFn, msg)
	os.Exit(1)
}

func (s *StreamLogger) Fatalf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintf(format, args...)

	s.writeMessage(fatalFn, msg+"\n")
	os.Exit(1)
}

func (s *StreamLogger) Done(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(doneFn, fmt.Sprintln(args...))

}

func (s *StreamLogger) Donef(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(doneFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Info(args...)
	case logrus.DebugLevel:
		s.Debug(args...)
	case logrus.WarnLevel:
		s.Warn(args...)
	case logrus.ErrorLevel:
		s.Error(args...)
	case logrus.FatalLevel:
		s.Fatal(args...)
	case logrus.PanicLevel:
		s.Fatal(args...)
	case logrus.TraceLevel:
		s.Debug(args...)
	}
}

func (s *StreamLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Infof(format, args...)
	case logrus.DebugLevel:
		s.Debugf(format, args...)
	case logrus.WarnLevel:
		s.Warnf(format, args...)
	case logrus.ErrorLevel:
		s.Errorf(format, args...)
	case logrus.FatalLevel:
		s.Fatalf(format, args...)
	case logrus.PanicLevel:
		s.Fatalf(format, args...)
	case logrus.TraceLevel:
		s.Debugf(format, args...)
	}
}

func (s *StreamLogger) SetLevel(level logrus.Level) {
	s.m.Lock()
	defer s.m.Unlock()

	s.level = level
}

func (s *StreamLogger) GetLevel() logrus.Level {
	s.m.Lock()
	defer s.m.Unlock()

	return s.level
}

func (s *StreamLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return &NopCloser{io.Discard}
	}

	reader, writer := io.Pipe()
	go func() {
		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			if raw {
				s.WriteString(level, sa.Text()+"\n")
			} else {
				s.Print(level, sa.Text())
			}
		}
	}()

	return writer
}

func (s *StreamLogger) WriteString(level logrus.Level, message string) {
	s.m.Lock()
	defer s.m.Unlock()

	for _, s := range s.sinks {
		s.WriteString(level, message)
	}

	if s.level < level {
		return
	}
	_, _ = s.write(level, []byte(message))
}

func (s *StreamLogger) write(level logrus.Level, message []byte) (int, error) {
	var (
		n   int
		err error
	)
	if s.format == JSONFormat {
		s.writeJSON(string(message), logrus.InfoLevel)
		n = len(message)
	} else {
		s.getStream(level)
		n, err = s.stream.Write(message)
	}
	return n, err
}

func (s *StreamLogger) Question(params *survey.QuestionOptions) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isTerminal && !params.DefaultValueSet {
		return "", fmt.Errorf("cannot ask question '%s' because currently you're not using devspace in a terminal and default value is also not provided", params.Question)
	} else if !s.isTerminal && params.DefaultValueSet {
		return params.DefaultValue, nil
	}

	// Check if we can ask the question
	if s.level < logrus.InfoLevel {
		return "", errors.Errorf("cannot ask question '%s' because log level is too low", params.Question)
	}

	_, _ = s.write(logrus.InfoLevel, []byte("\n"))
	return s.survey.Question(params)
}

func WithNopCloser(writer io.Writer) io.WriteCloser {
	return &NopCloser{writer}
}

type NopCloser struct {
	io.Writer
}

func (NopCloser) Close() error { return nil }
