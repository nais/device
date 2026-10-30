package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

const (
	statusFileName = "agent-status.json"

	// statusFileInterval is how often the file is rewritten when nothing happens.
	// The agent can sit connected for hours without a transition, so without a
	// heartbeat "written three hours ago" and "died three hours ago" look the
	// same. Thirty seconds bounds how wrong a reader can be, it is the same order
	// as the gateway health check above, and it costs one write of a few hundred
	// bytes twice a minute. Readers should allow a few missed heartbeats before
	// calling the file stale, not one.
	statusFileInterval = 30 * time.Second

	statusFileWarning = "best effort, may be missing or stale, format may change, may be removed at any time, do not depend on it"
)

// agentStatusFileContent is the payload of agent-status.json. It is a de facto
// interface for readers that cannot reach the gRPC socket, but not a supported
// one, hence the warning field: whoever finds the file reads the file, not the
// README.
type agentStatusFileContent struct {
	ConnectionState  string `json:"connectionState"`
	Tenant           string `json:"tenant"`
	UpdatedAt        string `json:"updatedAt"`
	HeartbeatSeconds int    `json:"heartbeatSeconds"`
	Warning          string `json:"warning"`
}

// statusFile publishes the connection state and the active tenant to a file for
// callers that cannot reach the gRPC socket. Everything about it is best effort:
// update never blocks the caller, and no failure stops the agent.
type statusFile struct {
	path     string
	tempPath string
	interval time.Duration
	log      logrus.FieldLogger

	// updates has a single slot and carries the newest status only. A slow disk
	// drops intermediate updates instead of backing up the status loop.
	updates chan agentStatusFileContent

	// failing is read and written by run only.
	failing bool
}

func newStatusFile(configDir string, log logrus.FieldLogger) *statusFile {
	return &statusFile{
		path:     filepath.Join(configDir, statusFileName),
		tempPath: filepath.Join(configDir, statusFileName+".tmp"),
		interval: statusFileInterval,
		log:      log,
		updates:  make(chan agentStatusFileContent, 1),
	}
}

// update queues a status for writing and returns immediately. There is one
// producer, so draining before sending replaces the queued status rather than
// blocking on a write that has not finished.
func (s *statusFile) update(state pb.AgentState, tenant *pb.Tenant) {
	name := ""
	if tenant != nil {
		name = tenant.Name
	}

	select {
	case <-s.updates:
	default:
	}

	select {
	case s.updates <- agentStatusFileContent{ConnectionState: state.String(), Tenant: name}:
	default:
	}
}

// run writes the file until ctx is done, then removes it. The first write
// happens before any transition, so a file left behind by an agent that was
// killed is corrected as soon as the next one starts. Removing it on the way out
// means a file that is present but stale only ever comes from an unclean exit.
func (s *statusFile) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	last := agentStatusFileContent{ConnectionState: pb.AgentState_Disconnected.String()}
	s.write(last)

	for {
		select {
		case <-ctx.Done():
			// Best effort, and it does not hold up shutdown: the agent is on its
			// way out either way, and a file left behind by a kill is handled by
			// the write on the next startup.
			if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
				s.log.WithError(err).Debug("remove agent status file")
			}
			return
		case last = <-s.updates:
		case <-ticker.C:
		}

		s.write(last)

		// The heartbeat measures time since the last write, not since the last
		// tick. Connected pushes a status update every 20 seconds of its own, so
		// a free-running ticker would write more often than heartbeatSeconds says.
		ticker.Reset(s.interval)
	}
}

func (s *statusFile) write(status agentStatusFileContent) {
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	status.HeartbeatSeconds = int(s.interval.Seconds())
	status.Warning = statusFileWarning

	out, err := json.Marshal(status)
	if err != nil {
		s.logFailure(err, "encode agent status")
		return
	}

	// Write and rename, not write in place: a reader polling this file while the
	// heartbeat truncates it otherwise gets an empty or half-written document.
	if err := os.WriteFile(s.tempPath, out, 0o644); err != nil {
		s.logFailure(err, "write agent status")
		return
	}

	if err := os.Rename(s.tempPath, s.path); err != nil {
		s.logFailure(err, "replace agent status")
		return
	}

	s.failing = false
}

// logFailure logs the first failure in a run of failures at error level and the
// rest at debug. A config directory that is read-only stays read-only, and an
// error twice a minute forever is a bug of its own.
func (s *statusFile) logFailure(err error, msg string) {
	entry := s.log.WithError(err)
	if s.failing {
		entry.Debug(msg)
		return
	}

	s.failing = true
	entry.Error(msg)
}
