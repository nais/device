package logger

import (
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LogFileType string

const (
	LogDir                         = "logs"
	AgentLogFileType   LogFileType = "agent.log"
	HelperLogFileType  LogFileType = "helper.log"
	SystrayLogFileType LogFileType = "systray.log"
	DaysToKeep                     = 60
	Layout                         = "2006-01-02"
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

func (l *LogFile) Setup(level string) {
	err := os.MkdirAll(l.Dir, 0o755)
	if err != nil {
		log.Fatalf("creating log dir %s: %v", l.Dir, err)
	}

	logFile := l.OpenFile()
	// file must be before os.Stdout here because when running as Windows service writes to stdout fail.
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

func (l *LogFile) OpenFile() *os.File {
	logFilePath := filepath.Join(l.Dir, l.FileType.String())
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		log.Fatalf("unable to open log file %s, error: %v", logFilePath, err)
	}

	return logFile
}

func TidyLogFiles(configDir string, logRotation bool, fileType LogFileType) error {
	logFilePath := filepath.Join(configDir, "/", fileType.String())
	if logRotation {
		err := TruncateLogFile(logFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func TruncateLogFile(logFilePath string) error {
	content, err := readFileContent(logFilePath)
	if err != nil {
		return err
	}

	startDate := getLogFileStartDate(content)
	if startDate == "" {
		log.Errorf("unable to get log file start date")
		return nil
	}

	if rotate(startDate) {
		_, err = os.Create(logFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func rotate(startDate string) bool {
	date, err := time.Parse(Layout, startDate)
	if err != nil {
		log.Errorf("unable to parse log file start date %s, error: %v", startDate, err)
		return false
	}

	return time.Now().After(date.AddDate(0, 0, -DaysToKeep))
}

func readFileContent(logFilePath string) (string, error) {
	rawBytes, err := os.ReadFile(logFilePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(rawBytes), "\n")
	for i, line := range lines {
		if i == 0 {
			return line, nil
		}
		break
	}
	return "", nil
}

func getLogFileStartDate(line string) string {
	l := strings.Split(line, " ")
	if l != nil && l[0] != "" {
		return l[0]
	}
	return ""
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
