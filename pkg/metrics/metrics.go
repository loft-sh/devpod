package metrics

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

const filename = "/tmp/metrics.csv"

func ObserveShellSession(length int64) {
	writeToCSV(os.Getenv("LOFT_TRACE_ID"), "shell", length)
}

func ObserveSSHSession(t string, length int64) {
	writeToCSV(os.Getenv("LOFT_TRACE_ID"), t, length)
}

func ObserveSSHSessionWithID(traceId, t string, length int64) {
	writeToCSV(traceId, t, length)
}

func writeToCSV(traceId, sessionType string, length int64) (err error) {
	var file *os.File
	// Check file exists
	if _, err = os.Stat(filename); err != nil {
		// Create if not
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
	} else {
		// Or open in append mode if exists
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	data := []string{traceId, sessionType, fmt.Sprintf("%dms", length), time.Now().Format("2006-01-02-15:04:05")}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(data)
	if err != nil {
		return err
	}

	return nil
}
