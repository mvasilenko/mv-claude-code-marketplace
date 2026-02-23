//go:build windows

package platform

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxExtractFileSize is the maximum size of a single extracted file (500MB)
const maxExtractFileSize = 500 * 1024 * 1024

type windowsArchiver struct{}

// NewArchiver creates a Windows archiver for zip files
func NewArchiver() Archiver {
	return &windowsArchiver{}
}

// Extract extracts a zip archive to the destination directory
func (a *windowsArchiver) Extract(archivePath string, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer r.Close() //nolint:errcheck // cleanup in defer

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		cleanPath := filepath.Clean(fpath)
		if !strings.HasPrefix(cleanPath, cleanDest) {
			return fmt.Errorf("illegal file path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", fpath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", fpath, err)
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close() //nolint:errcheck // best-effort cleanup
			return fmt.Errorf("failed to open file in archive %s: %w", f.Name, err)
		}

		if _, err := io.Copy(outFile, io.LimitReader(rc, maxExtractFileSize)); err != nil {
			_ = outFile.Close() //nolint:errcheck // best-effort cleanup
			_ = rc.Close()      //nolint:errcheck // best-effort cleanup
			return fmt.Errorf("failed to write file %s: %w", fpath, err)
		}

		if err := outFile.Close(); err != nil {
			_ = rc.Close() //nolint:errcheck // best-effort cleanup
			return fmt.Errorf("failed to close file %s: %w", fpath, err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("failed to close archive file %s: %w", f.Name, err)
		}
	}

	return nil
}

// GetExpectedFormat returns the expected archive format
func (a *windowsArchiver) GetExpectedFormat() string {
	return "zip"
}
