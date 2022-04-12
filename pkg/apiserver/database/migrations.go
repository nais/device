package database

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
)

//go:embed schema/*.sql
var embedMigrations embed.FS

type migration struct {
	version int
	sql     string
}

func migrations() ([]migration, error) {
	var files []migration
	err := fs.WalkDir(embedMigrations, "schema", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		version, err := strconv.Atoi(filepath.Base(path)[:4])
		if err != nil {
			return fmt.Errorf("invalid version number in %q: %w", path, err)
		}

		b, err := fs.ReadFile(embedMigrations, path)
		if err != nil {
			return err
		}

		files = append(files, migration{
			version: version,
			sql:     string(b),
		})
		return nil
	})
	return files, err
}
