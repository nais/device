package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	configDir   = "testdata"
	localLogDir = configDir + "/" + LogDir
)

func TestLogFiles(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		want    string
		rotated bool
	}{
		{
			name: "create agent log file",
			dir:  localLogDir,
			want: "agent.log",
		},
		{
			name:    "create or update agent log file and rotated log file if expired",
			dir:     localLogDir,
			want:    "agent.log",
			rotated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogFile := NewLogFile(configDir, AgentLogFileType)
			testLogFile.Setup("DEBUG")
			testLogFiles(tt.dir)

			stat := getStat(t, tt.dir, tt.want)
			// log file should not be empty
			if isEmpty(stat) {
				t.Errorf("os.Stat() file size = %v", stat)
			}

			if err := TidyLogFiles(tt.dir, tt.rotated, testLogFile.FileType); err != nil {
				t.Errorf("TidyLogFiles() error = %v", err)
			}

			if tt.rotated {
				stat = getStat(t, tt.dir, tt.want)
				if !isEmpty(stat) {
					t.Errorf("os.Stat() file size = %v", stat)
				}
			}
		})
	}
}

func getStat(t *testing.T, dir, path string) int64 {
	file, err := os.Stat(filepath.Join(dir, path))
	if err != nil {
		t.Errorf("os.Stat() error = %v", err)
	}

	return file.Size()
}

func isEmpty(fileState int64) bool {
	return fileState == 0
}

func testLogFiles(dir string) {
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", HelperLogFileType.String())), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", SystrayLogFileType.String())), []byte("test"), 0644)
	// overwrite agent.log initial content with a date in the past
	t := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).Format(Layout)
	_ = os.WriteFile(filepath.Join(
		dir,
		AgentLogFileType.String()),
		[]byte(fmt.Sprintf("%s 23:12:34.80559 - [INFO] - Successfully set up logging. Level info", t)), 0644)
}
