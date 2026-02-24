// Package internal implements the admin workflow plugin.
package internal

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed config.yaml
var configData []byte

//go:embed all:ui_dist
var embeddedUIFS embed.FS

// extractAssets extracts embedded UI assets to the working directory
// if they don't already exist there.
func extractAssets() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	uiDir := filepath.Join(dir, "ui_dist")
	if _, err := os.Stat(filepath.Join(uiDir, "index.html")); err == nil {
		return nil // already extracted
	}

	return fs.WalkDir(embeddedUIFS, "ui_dist", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		target := filepath.Join(dir, path)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := embeddedUIFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0644)
	})
}
