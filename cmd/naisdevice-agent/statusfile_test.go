package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// start runs a status file in dir with a heartbeat short enough for a test, and
// stops it when the test ends.
func start(t *testing.T, dir string, log logrus.FieldLogger) *statusFile {
	t.Helper()

	sf := newStatusFile(dir, log)
	sf.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		sf.run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("status file did not stop")
		}
	})

	return sf
}

func read(t *testing.T, path string) agentStatusFileContent {
	t.Helper()

	in, err := os.ReadFile(path)
	require.NoError(t, err)

	content := agentStatusFileContent{}
	require.NoError(t, json.Unmarshal(in, &content))
	return content
}

func exists(path string) func() bool {
	return func() bool {
		_, err := os.Stat(path)
		return err == nil
	}
}

func TestStatusFile(t *testing.T) {
	t.Run("writes the file at startup, before any transition", func(t *testing.T) {
		dir := t.TempDir()
		sf := start(t, dir, logrus.New())
		path := filepath.Join(dir, statusFileName)

		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)

		content := read(t, path)
		assert.Equal(t, pb.AgentState_Disconnected.String(), content.ConnectionState)
		assert.Empty(t, content.Tenant)
		assert.Equal(t, statusFileWarning, content.Warning)
		assert.Equal(t, int(sf.interval.Seconds()), content.HeartbeatSeconds)
		assert.NotEmpty(t, content.UpdatedAt)

		// The payload states the real cadence, so a reader can pick a staleness
		// threshold without reading the README.
		assert.Equal(t, 30*time.Second, newStatusFile(dir, logrus.New()).interval)
	})

	t.Run("the heartbeat rewrites the file without a transition", func(t *testing.T) {
		dir := t.TempDir()
		start(t, dir, logrus.New())
		path := filepath.Join(dir, statusFileName)

		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
		require.NoError(t, os.Remove(path))

		// Nothing happens in between, so only the ticker can bring it back.
		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
	})

	t.Run("a transition rewrites the file", func(t *testing.T) {
		dir := t.TempDir()
		sf := start(t, dir, logrus.New())
		path := filepath.Join(dir, statusFileName)

		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
		sf.update(pb.AgentState_Connected, &pb.Tenant{Name: "NAV"})

		require.Eventually(t, func() bool {
			return read(t, path).ConnectionState == pb.AgentState_Connected.String()
		}, time.Second, 5*time.Millisecond)

		assert.Equal(t, "NAV", read(t, path).Tenant)
	})

	t.Run("no active tenant is written as an empty tenant", func(t *testing.T) {
		dir := t.TempDir()
		sf := start(t, dir, logrus.New())
		path := filepath.Join(dir, statusFileName)

		sf.update(pb.AgentState_Disconnected, nil)
		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
		assert.Empty(t, read(t, path).Tenant)
	})

	t.Run("the agent keeps running when the config directory is read-only", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		logger, hook := logrustest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		sf := start(t, dir, logger)
		path := filepath.Join(dir, statusFileName)

		// Every write fails. Updates must still return, and the loop must survive.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range 100 {
				sf.update(pb.AgentState_Connected, &pb.Tenant{Name: "NAV"})
			}
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("update blocked on a failing write")
		}

		require.Eventually(t, func() bool {
			return len(hook.AllEntries()) > 3
		}, time.Second, 5*time.Millisecond)
		assert.NoFileExists(t, path)

		// The loop is still alive: it writes again once the directory allows it.
		require.NoError(t, os.Chmod(dir, 0o700))
		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
	})

	t.Run("only the first failure in a row is logged at error level", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		logger, hook := logrustest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		start(t, dir, logger)

		require.Eventually(t, func() bool {
			return len(hook.AllEntries()) > 5
		}, time.Second, 5*time.Millisecond)

		errors := 0
		for _, entry := range hook.AllEntries() {
			if entry.Level == logrus.ErrorLevel {
				errors++
			}
		}
		assert.Equal(t, 1, errors, "a failing disk fails on every heartbeat, so only the first one is an error")
	})

	t.Run("updates do not make the file write more often than the interval", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		// Connected pushes a status update every 20 seconds while the heartbeat
		// is 30, so updates arrive faster than the interval without ever being
		// frequent. A ticker that is not reset after a write adds its own writes
		// on top. Scaled down: updates every 70ms against a 100ms interval.
		const (
			interval   = 100 * time.Millisecond
			updateWait = 70 * time.Millisecond
			duration   = 2 * time.Second
		)

		// A read-only directory turns every write attempt into a log entry, which
		// is how the test counts writes without the agent knowing it is counted.
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		logger, hook := logrustest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)

		sf := newStatusFile(dir, logger)
		sf.interval = interval

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			sf.run(ctx)
		}()

		updates := 0
		for ctx.Err() == nil {
			time.Sleep(updateWait)
			sf.update(pb.AgentState_Connected, &pb.Tenant{Name: "NAV"})
			updates++
		}
		<-done

		// One write per update, one at startup, and a little slack for a tick that
		// lands while a write is in flight. Without the reset the ticker fires
		// roughly duration/interval extra times on top of this.
		writes := len(hook.AllEntries())
		assert.LessOrEqual(t, writes, updates+5,
			"%d writes for %d updates, the ticker is writing on top of them", writes, updates)
	})

	t.Run("a reader never sees a half-written file", func(t *testing.T) {
		dir := t.TempDir()
		sf := newStatusFile(dir, logrus.New())
		sf.interval = time.Millisecond
		path := filepath.Join(dir, statusFileName)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			sf.run(ctx)
		}()

		require.Eventually(t, exists(path), time.Second, time.Millisecond)

		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			assert.NotEmpty(t, read(t, path).ConnectionState)
		}

		// Stop the writer before the test ends, or removing the temp directory
		// races the next write and the cleanup fails.
		cancel()
		<-done
	})

	t.Run("removes the file on shutdown", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, statusFileName)

		sf := newStatusFile(dir, logrus.New())
		sf.interval = 10 * time.Millisecond

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			sf.run(ctx)
		}()

		require.Eventually(t, exists(path), time.Second, 5*time.Millisecond)
		cancel()
		<-done

		assert.NoFileExists(t, path)
	})
}
