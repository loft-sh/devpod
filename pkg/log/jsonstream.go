package log

import (
	"encoding/json"
	"io"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/sirupsen/logrus"
)

func PipeJSONStream(logger log.Logger) (io.WriteCloser, chan struct{}) {
	done := make(chan struct{})
	reader, writer := io.Pipe()
	go func() {
		ReadJSONStream(reader, logger)
		close(done)
	}()

	return writer, done
}

func ReadJSONStream(reader io.Reader, logger log.Logger) {
	scan := scanner.NewScanner(reader)
	for scan.Scan() {
		lineObject, err := Unmarshal(scan.Bytes())
		if err == nil && lineObject.Message != "" {
			switch lineObject.Level {
			case logrus.TraceLevel:
				logger.Debug(lineObject.Message)
			case logrus.DebugLevel:
				logger.Debug(lineObject.Message)
			case logrus.InfoLevel:
				logger.Info(lineObject.Message)
			case logrus.WarnLevel:
				logger.Warn(lineObject.Message)
			case logrus.ErrorLevel:
				logger.Error(lineObject.Message)
			case logrus.PanicLevel:
				logger.Error(lineObject.Message)
			case logrus.FatalLevel:
				logger.Error(lineObject.Message)
			}
		}
	}
}

func Unmarshal(line []byte) (*log.Line, error) {
	lineObject := &log.Line{}
	err := json.Unmarshal(line, lineObject)
	if err != nil {
		return nil, err
	}

	return lineObject, nil
}
