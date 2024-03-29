package logger

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRotate(t *testing.T) {
	now, err := time.Parse(time.DateOnly, "2021-01-05")

	week := time.Hour * 24 * 7

	tests := []struct {
		name         string
		expectDelete []string
		expectKeep   []string
	}{
		{
			name:         "don't touch other files",
			expectDelete: []string{},
			expectKeep: []string{
				"other_file.log",
				"other_file.log",
				"agent_2021-01-05.log",
				"agent_2019-99-99.log",
				createLogFileName("other_systray", now.Add(-3*week)),
			},
		},
		{
			name:         "keep young files",
			expectDelete: []string{},
			expectKeep: []string{
				createLogFileName("agent", now.Add(-time.Hour*24*2)),
				createLogFileName("systray", now.Add(-time.Hour*24*3)),
				createLogFileName("helper", now.Add(-time.Hour*24*4)),
				createLogFileName("helper", now.Add(-time.Hour*24*4)),
			},
		},
		{
			name: "delete old files",
			expectDelete: []string{
				createLogFileName("agent", now.Add(-2*week)),
				createLogFileName("systray", now.Add(-3*week)),
				createLogFileName("helper", now.Add(-512*week)),
			},
			expectKeep: []string{},
		},
		{
			name: "mix of old and new files",
			expectDelete: []string{
				createLogFileName("agent", now.Add(-2*week)),
				createLogFileName("systray", now.Add(-3*week)),
				createLogFileName("helper", now.Add(-4*week)),
			},
			expectKeep: []string{
				createLogFileName("agent", now.Add(-time.Hour*24*2)),
				createLogFileName("systray", now.Add(-time.Hour*24*3)),
				createLogFileName("helper", now.Add(-time.Hour*24*4)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			assert.NoError(t, err)

			err := os.Chdir(testDir)
			assert.NoError(t, err)

			defer os.Chdir("..")

			for _, fileName := range tt.expectDelete {
				_, err = os.Create(fileName)
				assert.NoError(t, err)
			}

			for _, fileName := range tt.expectKeep {
				_, err = os.Create(fileName)
				assert.NoError(t, err)
			}

			deleteOldLogFiles(testDir, now.Add(-time.Hour*24*7))

			logdirFiles, err := os.ReadDir(".")
			assert.NoError(t, err)
			for _, shouldBeDeleted := range tt.expectDelete {
				for _, f := range logdirFiles {
					if f.Name() == shouldBeDeleted {
						t.Errorf("file %q should be deleted", shouldBeDeleted)
					}
				}
			}
		outer:
			for _, shouldBeKept := range tt.expectKeep {
				for _, f := range logdirFiles {
					if f.Name() == shouldBeKept {
						continue outer
					}
				}
				t.Errorf("file %q should be kept", shouldBeKept)
			}
		})
	}
}

func TestLatestFilename(t *testing.T) {
	now, err := time.Parse(time.DateOnly, "2021-01-05")
	assert.NoError(t, err)
	existingFiles := []string{
		createLogFileName("agent", now),
		createLogFileName("agent", now.Add(-time.Hour*24*3)),
		createLogFileName("systray", now),
		createLogFileName("systray", now.Add(-time.Hour*24*3)),
		createLogFileName("helper", now),
		createLogFileName("helper", now.Add(-time.Hour*24*4)),
	}

	tests := []struct {
		name   string
		prefix string
		files  []string
		want   string
	}{
		{
			name:   "latest agent file name",
			prefix: "agent",
			files:  existingFiles,
			want:   createLogFileName("agent", now),
		},
		{
			name:   "latest helper file name",
			prefix: "helper",
			files:  existingFiles,
			want:   createLogFileName("helper", now),
		},
		{
			name:   "latest systray file name",
			prefix: "systray",
			files:  existingFiles,
			want:   createLogFileName("systray", now),
		},
		{
			name:   "current date if no matching files",
			prefix: "systray",
			files: []string{
				createLogFileName("agent", now),
				createLogFileName("agent", now.Add(-time.Hour*24*3)),
				createLogFileName("helper", now),
				createLogFileName("helper", now.Add(-time.Hour*24*4)),
			},
			want: createLogFileName("systray", time.Now()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			assert.NoError(t, err)

			err := os.Chdir(testDir)
			assert.NoError(t, err)

			for _, fileName := range tt.files {
				_, err = os.Create(fileName)
				assert.NoError(t, err)
			}

			actual := LatestFilename(testDir, tt.prefix)
			assert.Equal(t, tt.want, actual)
		})
	}
}
