package file

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"
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
func HumanizeSize(bytes int64) string {
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
		return calculateDirSize(parentDir)
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

// progressWriter is a custom io.Writer that reports progress to a channel as bytes are written.
// It is used to monitor the progress of writing data, such as when creating a zip archive.
//
// Usage notes:
//   - Before starting the write operation, the total size should be sent to progressChan.
//   - During writing, progressWriter periodically sends the number of bytes written to progressChan.
//   - When the write operation is complete, set lastReport to true to ensure a final progress update is sent.
//   - If progressChan is closed, the writing process is considered done or errored out.
type progressWriter struct {
	// The underlying io.Writer to which data is written.
	w io.Writer
	// The total number of bytes written so far.
	written int64
	// progressChan: Channel for reporting progress (bytes written).
	progressChan chan<- int64
	// lastReportTime: Timestamp of the last progress report (used to throttle updates).
	lastReportTime time.Time
	// lastReport:  If true, forces a progress report regardless of lastReportTime.
	lastReport bool
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.w.Write(p)
	pw.written += int64(n)
	isReportTime := time.Since(pw.lastReportTime) > 500*time.Millisecond
	if (isReportTime && pw.written > 0) || pw.lastReport {
		pw.progressChan <- pw.written
		pw.lastReportTime = time.Now()
	}
	return n, err
}

// Zip creates a zip archive of the specified files/directories within a parent directory.
// If no filenames are provided, it creates a zip archive of the root directory.
// The first write to progressChan will be the total size of the archive.
//
// Parameters:
//   - progressChan: A channel to report progress to, must be closed by the caller.
//   - archiveName: The name of the zip archive to create.
//   - root: The path to the parent directory.
//   - filenames: Optional list of file or directory names within the parent directory to zip.
func Zip(progressChan chan<- int64, archiveName, root string, files ...string) (string, error) {
	archivePath := filepath.Join(os.TempDir(), archiveName)
	archive, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating empty zip archive: %w", err)
	}
	defer archive.Close()

	size, err := CalculateSize(root, files...)
	if err != nil {
		return "", fmt.Errorf("retrieving filesize: %w", err)
	}

	progressChan <- size // report total size

	pw := &progressWriter{w: archive, progressChan: progressChan}
	w := zip.NewWriter(pw)
	defer w.Close()

	// zip the whole dir if no files are specified
	if len(files) == 0 {
		if err = zipDir(w, root, root); err != nil {
			return "", fmt.Errorf("zipping whole dir %q: %w", root, err)
		}
	} else { // zip the specified files
		for _, file := range files {
			filePath := filepath.Join(root, file)
			var stat os.FileInfo
			if stat, err = os.Stat(filePath); err != nil {
				return "", fmt.Errorf("statting file %q: %w", file, err)
			}
			if stat.IsDir() {
				err = zipDir(w, root, filePath)
			} else {
				err = zipFile(w, root, filePath)
			}
			if err != nil {
				return "", fmt.Errorf("zipping %q: %w", filePath, err)
			}
		}
	}

	pw.lastReport = true // signal that we are done

	return archivePath, nil
}

// zipDir zips the contents of a directory into the zip archive.
func zipDir(w *zip.Writer, root, dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		return zipFile(w, root, path) // caller has all the context, no need for fmt.Errorf
	})
}

func zipFile(w *zip.Writer, basePath, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", filePath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("statting file: %w", err)
	}
	if info.IsDir() {
		panic("filePath is a directory, dev needs a coffee!")
	}

	fh, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("creating file header: %w", err)
	}
	relativeName, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return fmt.Errorf("determining relative path for fileheader name: %w", err)
	}
	fh.Name = relativeName
	fh.Method = zip.Deflate

	var ioW io.Writer
	ioW, err = w.CreateHeader(fh)

	if _, err = io.Copy(ioW, f); err != nil {
		return fmt.Errorf("copying %q to archive: %w", f.Name(), err)
	}

	return nil
}

func Process(cfg client.ShareConfig, progressChan chan<- int64, root string, files ...string) ([]string, error) {
	if cfg.ZipFiles {
		path, err := Zip(progressChan, cfg.SharedZipName, root, files...)
		return []string{path}, err
	}
	// get list of directories that needs to be zipped
	dirsToZip, err := getDirsToZip(files, root)
	if err != nil {
		return nil, err
	}
	// zip the directories
	paths := make([]string, len(dirsToZip))
	for i, dir := range dirsToZip {
		var path string
		if path, err = Zip(progressChan, cfg.SharedZipName, dir); err != nil {
			return nil, fmt.Errorf("zipping directory %q: %w", dir, err)
		}
		paths[i] = path
	}
	return nil, nil
}

func getDirsToZip(files []string, root string) ([]string, error) {
	dirsToZip := make([]string, 0)
	for _, file := range files {
		path := filepath.Join(root, file)
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("determining dirs to zip %q: %w", path, err)
		}
		if info.IsDir() {
			dirsToZip = append(dirsToZip, path)
		}
	}
	return dirsToZip, nil
}
