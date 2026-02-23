//go:build unix

package platform

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxExtractFileSize is the maximum size of a single extracted file (500MB)
const maxExtractFileSize = 500 * 1024 * 1024

type unixArchiver struct{}

// NewArchiver creates a Unix archiver for tar.gz files
func NewArchiver() Archiver {
	return &unixArchiver{}
}

// Extract extracts a tar.gz archive to the destination directory
func (a *unixArchiver) Extract(archivePath string, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close() //nolint:errcheck // read-only archive

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close() //nolint:errcheck // gzip reader cleanup

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		// Check for path traversal vulnerability
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		cleanPath := filepath.Clean(target)
		if !strings.HasPrefix(cleanPath, cleanDest) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, io.LimitReader(tr, maxExtractFileSize)); err != nil {
				outFile.Close() //nolint:errcheck // error path cleanup
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}

			outFile.Close() //nolint:errcheck // success path cleanup

			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to chmod %s: %w", target, err)
			}
		}
	}

	return nil
}

// GetExpectedFormat returns the expected archive format
func (a *unixArchiver) GetExpectedFormat() string {
	return "tar.gz"
}
