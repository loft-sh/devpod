package metrics

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

func ObserveSession(t string, length int64) {
	writeToCSV(os.Getenv("LOFT_TRACE_ID"), t, length)
}

func writeToCSV(traceId, sessionType string, length int64) (err error) {
	var file *os.File
	if traceId == "" {
		traceId = "unknown"
	}
	filename := fmt.Sprintf("/tmp/%s", traceId)
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

	data := []string{time.Now().Add(time.Duration(-length * int64(time.Millisecond))).Format("2006-01-02-15:04:05"), fmt.Sprintf("%dms", length), short(sessionType)}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func short(str string) string {
	if len(str) <= 80 {
		return str
	}
	return str[:80] + "..."
}
