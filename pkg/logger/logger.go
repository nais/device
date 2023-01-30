package logger

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

type LogFile struct {
	Dir      string
	FileType LogFileType
}

func NewLogFile(configDir string, fileType LogFileType) LogFile {
	return LogFile{
		Dir:      filepath.Join(configDir, LogDir),
		FileType: fileType,
	}
}

func (l *LogFile) Setup(level string, date time.Time, prefixed bool) {
	err := os.MkdirAll(l.Dir, 0o755)
	if err != nil {
		log.Fatalf("Creating log dir: %v", err)
	}

	logFile := l.OpenFile(date.Format("2006-01-02"), prefixed)

	loglevel, err := log.ParseLevel(level)
	if err != nil {
		log.Errorf("unable to parse log level %s, error: %v", level, err)
		return
	}

	// file must be before os.Stdout here because when running as Windows service writes to stdout fail.
	mw := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(mw)
	log.SetLevel(loglevel)
	log.SetFormatter(&easy.Formatter{TimestampFormat: "2006-01-02 15:04:05.00000", LogFormat: "%time% - [%lvl%] - %msg%\n"})
	log.Infof("Successfully set up logging. Level %s", loglevel)
}

func (l *LogFile) OpenFile(format string, prefixed bool) *os.File {
	var logFileName = l.FileType.String()
	if prefixed {
		logFileName = fmt.Sprintf("%s-%s", format, l.FileType.String())
	}

	logFilePath := filepath.Join(l.Dir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		log.Fatalf("unable to open log file %s, error: %v", logFilePath, err)
	}

	return logFile
}

func (l *LogFile) Tidy() error {
	fileList, err := AssembleLogFilesDescend(l.Dir, l.FileType)
	if err != nil {
		return err
	}
	if len(fileList) > 3 {
		l.removeOldest(fileList)
	}
	return nil
}

func (l *LogFile) removeOldest(fileList []os.DirEntry) {
	if fileList[0].Name() == AgentLogFileType.String() {
		err := os.Remove(filepath.Join(l.Dir, fileList[0].Name()))
		if err != nil {
			log.Errorf("unable to remove legacy log file: %v", err)
		}
		fileList = append(fileList[:0], fileList[1:]...)
	}

	if len(fileList) > 3 {
		for _, file := range fileList[3:] {
			err := os.Remove(filepath.Join(l.Dir, file.Name()))
			if err != nil {
				log.Errorf("unable to remove oldest log file: %v", err)
			}
		}
	}
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

func sortFileNameDescend(files []os.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
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
