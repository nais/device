package program

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
)

// sets up a context that will cancel on SIGINT or SIGTERM, and after a timeout, force-exit the program
func MainContext(gracefulShutdownTimeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	context.AfterFunc(ctx, func() {
		time.Sleep(gracefulShutdownTimeout)
		fmt.Printf("timed out waiting for program to exit gracefully (%v), force-exiting\n", gracefulShutdownTimeout)
		os.Exit(1)
	})

	return ctx, cancel
}
