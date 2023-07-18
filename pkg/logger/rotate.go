package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
)

func deleteOldLogFiles(logDirPath string, treshold time.Time) error {
	logFiles, err := os.ReadDir(logDirPath)
	if err != nil {
		return fmt.Errorf("open log dir: %w", err)
	}

	filesBeforeTreshold, err := filterOldFilesByDate(logFiles, treshold)
	if err != nil {
		return fmt.Errorf("filter old files: %w", err)
	}

	for _, logFile := range filesBeforeTreshold {
		err := os.Remove(filepath.Join(logDirPath, logFile.Name()))
		if err != nil {
			return fmt.Errorf("remove log file: %w", err)
		}
	}

	return nil
}

func filterOldFilesByDate(files []os.DirEntry, treshold time.Time) ([]os.DirEntry, error) {
	var filesBeforeTreshold []os.DirEntry
	filenameFormat := regexp.MustCompile(`^(agent|helper|systray)_(\d{4}-\d{2}-\d{2})\.log$`)

	for _, logFile := range files {
		matches := filenameFormat.FindAllStringSubmatch(logFile.Name(), -1)
		if len(matches) != 1 || len(matches[0]) != 3 {
			log.Debug("ignoring file: ", logFile)
			continue
		}

		date := matches[0][2]
		logDate, err := time.Parse(time.DateOnly, date)
		if err != nil {
			log.Errorf("filter old log files: unable to parse date: %q, err: %v", date, err)
			continue
		}

		if logDate.Before(treshold) {
			filesBeforeTreshold = append(filesBeforeTreshold, logFile)
		}
	}

	return filesBeforeTreshold, nil
}

func createLogFileName(prefix string, t time.Time) string {
	return fmt.Sprintf("%s_%s.log", prefix, t.Format(time.DateOnly))
}

func LatestFilepath(logDirPath, prefix string) string {
	return filepath.Join(logDirPath, LatestFilename(logDirPath, prefix))
}

func LatestFilename(logDirPath, prefix string) string {
	logFiles, err := os.ReadDir(logDirPath)
	if err != nil {
		log.Errorf("open log dir: %v", err)
	}

	newestFilename := createLogFileName(prefix, time.Now())
	newestDate := time.Time{}

	filenameFormat := regexp.MustCompile(fmt.Sprintf(`^(%s)_(\d{4}-\d{2}-\d{2})\.log$`, prefix))

	for _, logFile := range logFiles {
		matches := filenameFormat.FindAllStringSubmatch(logFile.Name(), -1)
		if len(matches) != 1 || len(matches[0]) != 3 {
			log.Debug("ignoring file: ", logFile)
			continue
		}

		date := matches[0][2]
		logDate, err := time.Parse(time.DateOnly, date)
		if err != nil {
			log.Errorf("inferring latest log file: unable to parse date: %q, err: %v", date, err)
			continue
		}

		if logDate.After(newestDate) {
			newestDate = logDate
			newestFilename = logFile.Name()
		}
	}

	return newestFilename
}
