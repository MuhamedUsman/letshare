package file

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Zipper tracks and reports progress of zip operations.
//
// Zipper is designed to be thread-safe and can be used concurrently from multiple
// goroutines. It maintains a cumulative count of bytes processed across all
// read operations and periodically reports progress through the provided channel.
type Zipper struct {
	progressCh chan uint64
	read       *atomic.Uint64
	lrMu       *sync.RWMutex
	lastRead   time.Time
}

// NewZipper creates a new Zipper instance.
//
// The progressCh parameter is a channel that receives progress updates
// during the zip operation. The first value sent to the channel will be
// the total size of all files to be zipped, and subsequent values will
// be the number of bytes processed so far.
//
// Example:
//
//	progressCh := make(chan uint64, 10)
//	zipper := ziputil.NewZipper(progressCh)
func NewZipper(progressCh chan uint64) *Zipper {
	return &Zipper{
		progressCh: progressCh,
		read:       new(atomic.Uint64),
		lrMu:       new(sync.RWMutex),
		lastRead:   time.Now(),
	}
}

// ZipArchive creates a zip archive of the specified files/directories.
//
// If no filenames are provided, it creates a zip archive of the entire root directory.
// The first write to progressChan will be the total size of the archive.
//
// Parameters:
//   - path: The directory where the zip archive will be created
//   - archiveName: The name of the zip archive to create (should include .zip extension)
//   - root: The path to the root directory to zip
//   - files: Optional list of file or directory names within the root directory to zip
//
// Returns:
//   - The full path to the created zip archive
//   - An error if the operation fails
//
// Example:
//
//	// Zip an entire directory
//	path, err := zipper.ZipArchive("/tmp", "backup.zip", "/home/user/documents")
//
//	// Zip specific files within a directory
//	path, err := zipper.ZipArchive("/tmp", "partial.zip", "/home/user/documents", "file1.txt", "folder1")
func (z *Zipper) ZipArchive(path, archiveName, root string, files ...string) (string, error) {
	archivePath := filepath.Join(path, archiveName)
	archive, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating empty zip archive: %w", err)
	}
	defer archive.Close()
	size, err := CalculateSize(root, files...)
	if err != nil {
		return "", fmt.Errorf("retrieving filesize: %w", err)
	}

	z.progressCh <- uint64(size) // report total size

	zw := zip.NewWriter(archive)
	defer zw.Close()

	// zip the whole dir if no files are specified
	if len(files) == 0 {
		if err = z.writeDir(zw, root, root); err != nil {
			return "", fmt.Errorf("zipping whole dir %q: %w", root, err)
		}
	} else { // zip the specified files
		for _, file := range files {
			filePath := filepath.Join(root, file)
			var stat fs.FileInfo
			if stat, err = os.Stat(filePath); err != nil {
				return "", fmt.Errorf("statting file %q: %w", file, err)
			}
			if stat.IsDir() {
				err = z.writeDir(zw, root, filePath)
			} else {
				err = z.writeFile(zw, root, filePath)
			}
			if err != nil {
				return "", fmt.Errorf("zipping %q: %w", filePath, err)
			}
		}
	}
	return archivePath, nil
}

// ZipArchives concurrently creates zip archives for multiple directories.
//
// This method creates a separate zip file for each directory specified in the dirs parameter.
// All zipping operations run concurrently using a worker pool for maximum efficiency.
//
// Parameters:
//   - path: The directory where the zip archives will be created
//   - root: The base path containing the directories to be zipped
//   - dirs: List of directory names within the root to zip (each becomes a separate archive)
//
// Returns:
//   - A slice of paths to the created zip archives, in the same order as the input dirs
//   - An error if any zipping operation fails
//
// Example:
//
//	// Zip multiple directories concurrently
//	paths, err := zipper.ZipArchives("/tmp", "/home/user", "documents", "pictures", "music")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, path := range paths {
//	    fmt.Println("Created archive:", path)
//	}
func (z *Zipper) ZipArchives(path, root string, dirs ...string) ([]string, error) {
	if err := checkValidDirs(root, dirs...); err != nil {
		return nil, err
	}
	size, err := CalculateSize(root, dirs...)
	if err != nil {
		return nil, fmt.Errorf("retrieving total size of dirs: %w", err)
	}
	z.progressCh <- uint64(size) // report total size

	zippedDirs := make([]string, len(dirs))
	wp := bgtask.NewWorkerPool(context.TODO())

main:
	for i, dir := range dirs {
		select {
		case <-wp.Ctx.Done(): // do the cleanup
			for _, zd := range zippedDirs {
				_ = os.Remove(zd) // ignore any error
			}
			break main
		default:
			wp.Spawn(func() error {
				dirToZip := filepath.Join(root, dir)
				archiveName := dir + ".zip"
				var archivePath string
				if archivePath, err = z.ZipArchive(path, archiveName, dirToZip); err != nil {
					return err
				}
				zippedDirs[i] = archivePath
				return nil
			})
		}
	}

	if err = wp.Wait(); err != nil {
		return nil, fmt.Errorf("blocking for all zipping operations: %w", err)
	}

	return zippedDirs, nil
}

// Close sends a final progress update and closes the progress channel,
// and sets the accumulative read bytes to 0.
//
// This should be called when all zip operations are complete to ensure
// the final progress is reported and to allow any goroutines reading
// from the progress channel to terminate properly.
//
// Example:
//
//	defer zipper.Close()
func (z *Zipper) Close() {
	z.progressCh <- z.read.Load()
	close(z.progressCh)
	z.read.Store(0)
}

// newReader creates a new progressReader that wraps the provided io.Reader.
//
// This is an internal method used to create reader instances that track
// progress during file operations.
func (z *Zipper) newReader(base io.Reader) io.Reader {
	return &progressReader{
		r:      base,
		Zipper: z,
	}
}

// writeDir zips the contents of a directory into the zip archive.
//
// This is an internal method used by ZipArchive to recursively process directories.
//
// Parameters:
//   - w: The zip writer to write to
//   - root: The base path for calculating relative paths
//   - dirPath: The absolute path of the directory to zip
//
// Returns:
//   - An error if the operation fails
func (z *Zipper) writeDir(w *zip.Writer, root, dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		return z.writeFile(w, root, path)
	})
}

// writeFile adds a single file to the zip archive.
//
// This is an internal method used by both ZipArchive and writeDir to add files to the archive.
// It maintains the directory structure relative to the root directory.
//
// Parameters:
//   - w: The zip writer to write to
//   - basePath: The base path for calculating relative paths
//   - filePath: The absolute path of the file to add
//
// Returns:
//   - An error if the operation fails
func (z *Zipper) writeFile(w *zip.Writer, basePath, filePath string) error {
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
	if ioW, err = w.CreateHeader(fh); err != nil {
		return fmt.Errorf("creating file header in archive: %w", err)
	}

	r := z.newReader(f)
	buf := make([]byte, 1024*1024) // 1MB buffer
	if _, err = io.CopyBuffer(ioW, r, buf); err != nil {
		return fmt.Errorf("copying %q to archive: %w", f.Name(), err)
	}
	return nil
}

// progressReader implements io.Reader and tracks read progress.
//
// progressReader wraps an underlying io.Reader and counts bytes read,
// sending progress updates through the parent Zipper's progress channel
// at regular intervals.
type progressReader struct {
	r       io.Reader // Underlying reader
	*Zipper           // Embedded Zipper for progress tracking
}

// Read implements io.Reader interface for progressReader.
//
// This method reads data from the underlying reader while tracking
// progress. It updates the cumulative byte counter and sends progress
// updates to the progress channel at regular intervals (500ms).
//
// The progress updates are sent in a non-blocking way to prevent
// deadlocks if the channel receiver is not ready.
func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.read.Add(uint64(n))
	pr.lrMu.RLock()
	isReportTime := time.Since(pr.lastRead) > 500*time.Millisecond
	pr.lrMu.RUnlock()
	if isReportTime {
		select { // non-blocking send
		case pr.progressCh <- pr.read.Load():
		default:
		}
		pr.lrMu.Lock()
		pr.lastRead = time.Now()
		pr.lrMu.Unlock()
	}
	return
}

// checkValidDirs verifies that all specified paths are valid directories.
//
// This is an internal utility function used by ZipArchives to validate input directories.
//
// Parameters:
//   - root: The base path containing the directories
//   - dirs: The directories to check
//
// Returns:
//   - An error if any path is not a valid directory
func checkValidDirs(root string, dirs ...string) error {
	for _, dir := range dirs {
		info, err := os.Lstat(filepath.Join(root, dir))
		if err != nil {
			return fmt.Errorf("statting dir %q: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %q is not a directory", dir)
		}
	}
	return nil
}
