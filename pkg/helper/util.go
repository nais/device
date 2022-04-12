package helper

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	TunnelNetworkPrefix = "10.255.24."
)

func filesExist(files ...string) error {
	for _, file := range files {
		if err := RegularFileExists(file); err != nil {
			return err
		}
	}

	return nil
}

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
		} else {
			log.Debugf("cmd: %v: %v\n", cmd, string(out))
		}

		time.Sleep(100 * time.Millisecond) // avoid serializable race conditions with kernel
	}
	return nil
}

func ZipLogFiles(files []string) (string, error) {
	if len(files) == 0 {
		return "nil", errors.New("can't be bothered to zip nothing")
	}
	archive, err := ioutil.TempFile(os.TempDir(), "naisdevice_logs.*.zip")
	if err != nil {
		return "nil", err
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)
	for _, filename := range files {
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			log.Printf("%s does not exist so I can't zip it\n", filename)
			continue
		}
		logFile, err := os.Open(filename)
		if err != nil {
			log.Errorf("%s %v", filename, err)
			return "nil", err
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
