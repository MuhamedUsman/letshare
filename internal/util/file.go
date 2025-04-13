package util

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
)

// UserFriendlyFilesize converts filesize to user-friendly string
//
// Parameters:
//   - bytes: Filesize in bytes
//
// Returns:
//   - string: formated strings of filesize (KB, MB, GB)
func UserFriendlyFilesize(bytes int64) string {
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.2fKB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.2fMB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2fGB", gb)
}

func ZipDir(dirPath string) (string, error) {
	root, err := os.OpenRoot(dirPath)
	if err != nil { // if err is not nil it will always be of type *os.PathError
		return "", fmt.Errorf("opening dirPath as root %q: %w", dirPath, err)
	}
	defer func() {
		if err = root.Close(); err != nil {
			slog.Error("closing dirPath as root", "path", dirPath, "err", err)
		}
	}()
	zipInto := fmt.Sprint(dirPath, ".zip")
	archive, err := os.Create(zipInto)
	if err != nil {
		return "", fmt.Errorf("creating empty zip archive: %w", err)
	}
	defer func() {
		if err = archive.Close(); err != nil {
			slog.Error("closing zip archive", "path", zipInto, "err", err)
		}
	}()
	w := zip.NewWriter(archive)
	defer func() {
		if err = w.Close(); err != nil {
			slog.Error("closing root FS", "path", zipInto, "err", err)
		}
	}()
	if err = zipDir(w, root.FS()); err != nil {
		return "", fmt.Errorf("zipping directory into %q: %w", zipInto, err)
	}
	return zipInto, nil
}

// zipDir actually writes/adds the fs root to the archive
func zipDir(w *zip.Writer, fs fs.FS) error {
	if err := w.AddFS(fs); err != nil {
		return fmt.Errorf("writing zip archive, method deflate: %w", err)
	}
	return nil
}
