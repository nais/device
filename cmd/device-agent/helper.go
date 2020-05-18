package main

import (
	"context"
	"os/exec"
)

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}
