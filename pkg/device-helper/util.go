package device_helper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
