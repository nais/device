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
		rotate  bool
		date    string
	}{
		{
			name: "create folders and agent log file with properties with rotation disabled",
			dir:  localLogDir,
			want: "agent.log",
		},
		{
			name:    "rotation of log file is enabled, purged existing logfile",
			dir:     localLogDir,
			want:    "agent.log",
			rotate:  true,
			rotated: true,
			date:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Format(TimeStampFormat),
		},
		{
			name:    "append on agent log file and dont purge file with days left to keep",
			dir:     localLogDir,
			want:    "agent.log",
			rotate:  true,
			rotated: false,
			date:    time.Now().Format(TimeStampFormat),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogFile := NewLogFile(configDir, AgentLogFileType)
			testLogFile.Setup("DEBUG")
			testLogFiles(tt.dir, tt.want, tt.date, tt.rotate)
			stat := getStat(t, tt.dir, tt.want)
			// log file should not be empty
			if isEmpty(stat) {
				t.Errorf("os.Stat() file size = %v", stat)
			}

			if err := TidyLogFiles(tt.dir, tt.rotate, testLogFile.FileType); err != nil {
				t.Errorf("TidyLogFiles() error = %v", err)
			}

			if tt.rotate {
				stat = getStat(t, tt.dir, tt.want)
				if tt.rotated {
					if !isEmpty(stat) {
						t.Errorf("os.Stat() file size = %v", stat)
					}
				} else {
					if isEmpty(stat) {
						t.Errorf("os.Stat() file size = %v", stat)
					}
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

func testLogFiles(dir, fileType, date string, rotate bool) {
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", HelperLogFileType.String())), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", SystrayLogFileType.String())), []byte("test"), 0644)
	// overwrite agent.log initial content with a custom date
	if rotate {
		_ = os.WriteFile(filepath.Join(
			dir,
			fileType),
			[]byte(fmt.Sprintf("%s - [INFO] - Successfully set up logging. Level info", date)), 0644)
	}
}
