package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	localLogDir      = "testdata/logs"
	localLogDirClean = "testdata/logs-clean"
)

func TestAssembleLogFilesDescend(t *testing.T) {
	tests := []struct {
		name        string
		want        []string
		createFiles func()
	}{
		{
			name: "assemble log files",
			want: []string{"2023-01-03-agent.log", "2023-01-02-agent.log", "2023-01-01-agent.log"},
			createFiles: func() {
				writeTestLogFiles(t, localLogDir, 3, AgentLogFileType)

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.createFiles()
			got, err := AssembleLogFilesDescend(localLogDir, AgentLogFileType)
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

func TestCleanUpLogFiles(t *testing.T) {
	tests := []struct {
		name        string
		want        []string
		createFiles func()
	}{
		{
			name: "clean up log files",
			want: []string{"2023-01-04-agent.log", "2023-01-03-agent.log", "2023-01-02-agent.log"},
			createFiles: func() {
				writeTestLogFiles(t, localLogDirClean, 4, AgentLogFileType)
			},
		},
		{
			name: "clean up log files with legacy log file",
			want: []string{"2023-01-03-agent.log", "2023-01-02-agent.log", "2023-01-01-agent.log"},
			createFiles: func() {
				// Create legacy log file
				writeTestLogFiles(t, localLogDirClean, 3, AgentLogFileType)
				_ = os.WriteFile(filepath.Join(localLogDirClean, fmt.Sprintf("%s", AgentLogFileType.String())), []byte("test"), 0644)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.createFiles()
			if err := CleanUpLogFiles(localLogDirClean, AgentLogFileType); err != nil {
				t.Errorf("CleanUpLogFiles() error = %v", err)
			}

			got, err := AssembleLogFilesDescend(localLogDirClean, AgentLogFileType)
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

func writeTestLogFiles(t *testing.T, dir string, files int, fileType LogFileType) {
	// Create log files
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, os.ModePerm)
	for i := 0; i < files; i++ {
		date := "2023-01-" + fmt.Sprintf("%02d", i+1)
		err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s-%s", date, fileType.String())), []byte("test"), 0644)
		if err != nil {
			t.Errorf("unable to create test log file: %v", err)
		}
	}
	_ = os.WriteFile(filepath.Join(localLogDirClean, fmt.Sprintf("%s", HelperLogFileType.String())), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(localLogDirClean, fmt.Sprintf("%s", SystrayLogFileType.String())), []byte("test"), 0644)
}
