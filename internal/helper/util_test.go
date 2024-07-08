package helper

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllFilesAreZippedWhenPresent(t *testing.T) {
	filesToZip := createTempFiles(t)
	zipLocation, err := ZipLogFiles(filesToZip)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(removeTestFile(t, zipLocation))

	zipfile, err := zip.OpenReader(zipLocation)
	if err != nil {
		t.Fatal(err)
	}
	defer zipfile.Close()

	filesInZip := zipfile.File
	assert.Equal(t, len(filesToZip), len(filesInZip))
}

func removeTestFile(t *testing.T, file string) func() {
	return func() {
		if err := os.Remove(file); err != nil {
			t.Logf("unable to clean up temp file: %v", err)
		}
	}
}

func TestNonExistingFilesAreSkipped(t *testing.T) {
	files := createTempFiles(t)
	files = append(files, "NonExisting")
	zipLocation, err := ZipLogFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(removeTestFile(t, zipLocation))

	zipfile, err := zip.OpenReader(zipLocation)
	if err != nil {
		t.Fatal(err)
	}
	defer zipfile.Close()

	filesInZip := zipfile.File
	assert.Equal(t, len(files)-1, len(filesInZip))
}

func TestEmptyFileListYieldsError(t *testing.T) {
	var files []string
	_, err := ZipLogFiles(files)
	assert.Error(t, err)
}

func createTempFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for i := 0; i <= 5; i++ {
		tempDir := t.TempDir()
		file, err := os.CreateTemp(tempDir, "txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Remove(file.Name()); err != nil {
				t.Logf("unable to clean up temp file: %v", err)
			}
		})
		files = append(files, file.Name())
	}
	return files
}
