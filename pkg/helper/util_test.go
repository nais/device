package helper

import (
	"archive/zip"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestAllFilesAreZippedWhenPresent(t *testing.T) {
	filesToZip := createTempFiles()
	zipLocation, _ := ZipLogFiles(filesToZip)

	zipfile, _ := zip.OpenReader(zipLocation)
	defer zipfile.Close()

	filesInZip := zipfile.File
	assert.Equal(t, len(filesToZip), len(filesInZip))
}

func TestNonExistingFilesAreSkipped(t *testing.T) {
	files := createTempFiles()
	files = append(files, "NonExisting")
	zipLocation, _ := ZipLogFiles(files)

	zipfile, _ := zip.OpenReader(zipLocation)
	defer zipfile.Close()

	filesInZip := zipfile.File
	assert.Equal(t, len(files)-1, len(filesInZip))
}

func TestEmptyFileListYieldsError(t *testing.T) {
	var files []string
	_, err := ZipLogFiles(files)
	assert.Error(t, err)
}

func createTempFiles() []string {
	var files []string
	for i := 0; i <= 5; i++ {
		file, _ := ioutil.TempFile(os.TempDir(), "txt")
		files = append(files, file.Name())
	}
	return files
}
