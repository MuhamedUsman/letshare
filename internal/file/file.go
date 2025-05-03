package file

import (
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

type CopyStat struct {
	// current number of file being copied
	N int // 0: Err occurred before copying
	// error encountered while copying the file
	Err error
}

type CopyStatChan <-chan CopyStat

// CopyFilesToDir copies the specified files to the target directory.
// It returns a CopyStatChan that reports progress and errors during the operation.
//
// The returned channel receives a CopyStat after each file operation, containing:
// - N: The number of files processed so far (1-indexed)
// - Err: Any error that occurred during the operation
//
// The channel will be closed when either:
// - All files have been copied successfully
// - An error occurs (copying stops at first error)
// - The Server's shutdown context is canceled
//
// If the shutdown context is canceled during copying, all copied files
// will be deleted from the target directory.
func CopyFilesToDir(dir string, files ...string) CopyStatChan {
	ch := make(chan CopyStat)
	bgtask.New().Run(func(shutdownCtx context.Context) {
		defer close(ch)
		for i, f := range files {
			select {
			case <-shutdownCtx.Done():
				// Once canceled during copying, delete all the files that are copied
				for _, file := range files {
					copiedFilepath := filepath.Join(dir, filepath.Base(file))
					_ = os.Remove(copiedFilepath) // ignore any error
				}
				return
			default:
				n := i + 1
				if _, err := os.Stat(dir); err != nil {
					if os.IsNotExist(err) {
						ch <- CopyStat{N: n, Err: fmt.Errorf("file does not exist: %v", f)}
					} else {
						ch <- CopyStat{N: n, Err: fmt.Errorf("cannot access file %q: %v", f, err.Error())}
					}
					return
				}
				destPath := filepath.Join(dir, filepath.Base(f))
				if err := copyFile(f, destPath); err != nil {
					ch <- CopyStat{N: n, Err: fmt.Errorf("copying file %q to %q: %v", f, destPath, err.Error())}
					return
				}
				ch <- CopyStat{N: n, Err: nil}
			}
		}
	})
	return ch
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %v", src, err)
	}
	defer func() {
		if err = s.Close(); err != nil {
			slog.Error("failed to close source file", "file", src, "Err", err)
		}
	}()
	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file %s: %v", dst, err)
	}
	defer func() {
		if err = d.Close(); err != nil {
			slog.Error("failed to close destination file", "file", dst, "Err", err)
		}
	}()
	if _, err = io.Copy(d, s); err != nil {
		return fmt.Errorf("copying file %s to %s: %v", src, dst, err)
	}
	return nil
}

// HumanizeSize converts filesize to user-friendly string
//
// Parameters:
//   - bytes: Filesize in bytes
//
// Returns:
//   - string: formated strings of filesize (KB, MB, GB)
func HumanizeSize(bytes uint64) string {
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

// CalculateSize calculates the total size in bytes of specified files/directories within a parent directory.
// If no filenames are provided, it returns the total size of the parent directory.
//
// Parameters:
//   - parentDir: The path to the parent directory.
//   - filenames: Optional list of file or directory names within the parent directory to calculate size for.
//
// Returns:
//   - int64: Total size in bytes.
//   - error: An error if any occurred during size calculation, such as
//   - File access errors
//   - File stat errors
//   - Directory traversal errors
func CalculateSize(parentDir string, filenames ...string) (int64, error) {
	if len(filenames) == 0 {
		size, err := calculateDirSize(parentDir)
		return size, err
	}
	size := int64(0)
	for _, name := range filenames {
		path := filepath.Join(parentDir, name)
		info, err := os.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("reading filestat for %q: %w", path, err)
		}
		if !info.IsDir() {
			size += info.Size()
			continue
		}
		dirSize, err := calculateDirSize(path)
		if err != nil {
			return 0, fmt.Errorf("calculating dir size for %q: %w", path, err)
		}
		size += dirSize
	}
	return size, nil
}

func calculateDirSize(path string) (int64, error) {
	size := int64(0)
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			var info fs.FileInfo
			info, err = d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}
