package helper

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

const (
	TunnelNetworkPrefix = "10.255.24."
)

func RegularFileExists(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%v is a directory", filepath)
	}

	return nil
}

func runCommands(ctx context.Context, commands [][]string) error {
	for _, s := range commands {
		cmd := exec.CommandContext(ctx, s[0], s[1:]...)

		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
		}

		time.Sleep(100 * time.Millisecond) // avoid serializable race conditions with kernel
	}
	return nil
}

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
