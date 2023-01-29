package logger

import (
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LogFileType string

const (
	LogDir                         = "logs"
	AgentLogFileType   LogFileType = "agent.log"
	HelperLogFileType  LogFileType = "helper.log"
	SystrayLogFileType LogFileType = "systray.log"
)

func (c LogFileType) String() string {
	return string(c)
}

type Logger struct {
	Name     string
	Dir      string
	FileType LogFileType
}

func SetupLogger(level, logDir, filename string) {
	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		log.Fatalf("Creating log dir: %v", err)
	}

	logFilePath := filepath.Join(logDir, filename)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		log.Fatalf("unable to open log file %s, error: %v", logFilePath, err)
	}

	// file must be before os.Stdout here because when running as windows service writes to stdout fail.
	mw := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(mw)

	loglevel, err := log.ParseLevel(level)
	if err != nil {
		log.Errorf("unable to parse log level %s, error: %v", level, err)
		return
	}
	log.SetLevel(loglevel)
	log.SetFormatter(&easy.Formatter{TimestampFormat: "2006-01-02 15:04:05.00000", LogFormat: "%time% - [%lvl%] - %msg%\n"})
	log.Infof("Successfully set up logging. Level %s", loglevel)
}

func Setup(level string) {
	log.SetFormatter(&log.JSONFormatter{FieldMap: log.FieldMap{
		log.FieldKeyMsg: "message",
	}})

	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(l)
}

func CleanUpLogFiles(logDir string, logFileType LogFileType) error {
	fileList, err := AssembleLogFilesDescend(logDir, logFileType)
	if err != nil {
		return err
	}
	if len(fileList) > 3 {
		removeOldest(fileList, logDir)
	}
	return nil
}

func AssembleLogFilesDescend(logDir string, logFileType LogFileType) ([]os.DirEntry, error) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	sortFileNameDescend(files)
	fileList := make([]os.DirEntry, 0)
	for _, file := range files {
		if strings.Contains(file.Name(), logFileType.String()) {
			fileList = append(fileList, file)
		}
	}
	return fileList, nil
}

func removeOldest(fileList []os.DirEntry, logDir string) {
	if fileList[0].Name() == AgentLogFileType.String() {
		err := os.Remove(filepath.Join(logDir, fileList[0].Name()))
		if err != nil {
			log.Errorf("unable to remove legacy log file: %v", err)
		}
		return
	}

	for _, file := range fileList[3:] {
		err := os.Remove(filepath.Join(logDir, file.Name()))
		if err != nil {
			log.Errorf("unable to remove oldest log file: %v", err)
		}
	}
}

func sortFileNameDescend(files []os.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
}

func GetLogDir(configDir string) string {
	return filepath.Join(configDir, LogDir)
}
