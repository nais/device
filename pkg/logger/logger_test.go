package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const (
	configDir   = "testdata"
	localLogDir = configDir + "/" + LogDir
)

func TestLogFiles(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		want       []string
		legacy     bool
		prefixed   bool
		tidy       bool
		extraFiles int
	}{
		{
			name:       "assemble log files",
			dir:        localLogDir,
			want:       []string{"2023-01-03-agent.log", "2023-01-02-agent.log", "2023-01-01-agent.log"},
			prefixed:   true,
			extraFiles: 2,
		},
		{
			name:       "tidy log files, should remove the oldest file (descending)",
			dir:        localLogDir,
			want:       []string{"2023-01-04-agent.log", "2023-01-03-agent.log", "2023-01-02-agent.log"},
			tidy:       true,
			prefixed:   true,
			extraFiles: 3,
		},
		{
			name:       "tidy log files with legacy log file, should remove legacy file",
			dir:        localLogDir,
			want:       []string{"2023-01-03-agent.log", "2023-01-02-agent.log", "2023-01-01-agent.log"},
			legacy:     true,
			tidy:       true,
			prefixed:   true,
			extraFiles: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogFile := NewLogFile(configDir, AgentLogFileType)

			// remove local logs
			_ = os.RemoveAll(tt.dir)
			time := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
			testLogFile.Setup("DEBUG", time, tt.prefixed)
			writeExtraTestLogFiles(testLogFile, tt.dir, time, tt.extraFiles, tt.prefixed, tt.legacy)

			if tt.tidy {
				if err := testLogFile.Tidy(); err != nil {
					t.Errorf("Tidy() error = %v", err)
				}
			}

			got, err := AssembleLogFilesDescend(tt.dir, AgentLogFileType)
			if err != nil {
				t.Errorf("AssembleLogFilesDescend() error = %v", err)
			}

			var gotFileNames []string
			for _, file := range got {
				gotFileNames = append(gotFileNames, file.Name())
			}

			if !reflect.DeepEqual(gotFileNames, tt.want) {
				t.Errorf("AssembleLogFilesDescend() = %v, want %v", gotFileNames, tt.want)
			}
		})
	}
}

func writeExtraTestLogFiles(logFile LogFile, dir string, time time.Time, files int, prefixed, legacy bool) {
	for i := 0; i < files; i++ {
		_ = logFile.OpenFile(time.AddDate(0, 0, i+1).Format("2006-01-02"), prefixed)
	}

	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", HelperLogFileType.String())), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s", SystrayLogFileType.String())), []byte("test"), 0644)

	if legacy {
		_ = os.WriteFile(filepath.Join(dir, AgentLogFileType.String()), []byte("test"), 0644)
	}
}
