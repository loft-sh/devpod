package up

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
)

func findMessage(reader io.Reader, message string) error {
	scan := scanner.NewScanner(reader)
	for scan.Scan() {
		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}

		lineObject := &log.Line{}
		err := json.Unmarshal(line, lineObject)
		if err == nil && strings.Contains(lineObject.Message, message) {
			return nil
		}
	}

	return fmt.Errorf("couldn't find message '%s' in log", message)
}

func verifyLogStream(reader io.Reader) error {
	scan := scanner.NewScanner(reader)
	for scan.Scan() {
		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}

		lineObject := &log.Line{}
		err := json.Unmarshal(line, lineObject)
		if err != nil {
			return fmt.Errorf("error reading line %s: %w", string(line), err)
		}
	}

	return nil
}

func createTarGzArchive(outputFilePath string, filePaths []string) error {
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer gzipWriter.Close()

	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			return err
		}

		fileInfoHdr, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return err
		}

		err = tarWriter.WriteHeader(fileInfoHdr)
		if err != nil {
			return err
		}

		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return err
		}
	}
	return nil
}
