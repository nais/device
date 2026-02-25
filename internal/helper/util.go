package helper

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

const (
	TunnelNetworkPrefix = "10.255.24."
)

func ZipLogFiles(files []string, log logrus.FieldLogger) (string, error) {
	if len(files) == 0 {
		return "nil", errors.New("can't be bothered to zip nothing")
	}
	archive, err := os.CreateTemp(os.TempDir(), "naisdevice_logs.*.zip")
	if err != nil {
		return "nil", err
	}
	defer ioconvenience.CloseWithLog(archive, log)
	zipWriter := zip.NewWriter(archive)
	for _, filename := range files {
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			continue
		}
		logFile, err := os.Open(filename)
		if err != nil {
			return "nil", fmt.Errorf("%s %v", filename, err)
		}
		zipEntryWiter, err := zipWriter.Create(filepath.Base(filename))
		if err != nil {
			return "nil", err
		}
		if _, err := io.Copy(zipEntryWiter, logFile); err != nil {
			return "nil", err
		}
		err = logFile.Close()
		if err != nil {
			return "nil", err
		}
	}
	err = zipWriter.Close()
	if err != nil {
		return "nil", err
	}
	return archive.Name(), nil
}
