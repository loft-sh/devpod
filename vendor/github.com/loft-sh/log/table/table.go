package table

import (
	"io"
	"runtime"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
)

func PrintTable(s log.Logger, header []string, values [][]string) {
	PrintTableWithOptions(s, header, values, nil)
}

// PrintTableWithOptions prints a table with header columns and string values
func PrintTableWithOptions(s log.Logger, header []string, values [][]string, modify func(table *tablewriter.Table)) {
	reader, writer := io.Pipe()
	defer writer.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			s.WriteString(logrus.InfoLevel, "  "+sa.Text()+"\n")
		}
	}()

	table := tablewriter.NewWriter(writer)
	table.SetHeader(header)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		colors := []tablewriter.Colors{}
		for range header {
			colors = append(colors, tablewriter.Color(tablewriter.FgGreenColor))
		}
		table.SetHeaderColor(colors...)
	}

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.AppendBulk(values)
	if modify != nil {
		modify(table)
	}

	// Render
	_, _ = writer.Write([]byte("\n"))
	table.Render()
	_, _ = writer.Write([]byte("\n"))
	_ = writer.Close()
	<-done
}
