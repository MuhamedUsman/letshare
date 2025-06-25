package zipr

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type compressionAlgo = uint16

const (
	Store   compressionAlgo = 0 // no compression
	Deflate compressionAlgo = 8 // max compression
)

// Zipr tracks and reports progress of zip operations.
//
// Zipr is designed to be thread-safe and can be used concurrently from multiple
// goroutines. It maintains a cumulative count of bytes processed across all
// read operations and periodically reports progress through the provided channel.
type Zipr struct {
	progressCh chan<- uint64
	logCh      chan<- string
	read       atomic.Uint64
	lrMu       sync.RWMutex // lrMu caps lastRead to avoid race conditions
	lastRead   time.Time
	algo       compressionAlgo
}

// New creates a new Zipr instance with the specified compression algorithm.
//
// Parameters:
//   - progressCh: A channel that receives progress updates during zip operations.
//     The first value sent is the total size of all files to be zipped,
//     and subsequent values report the number of bytes processed so far.
//     To avoid missing the initial total filesize report, ensure you're
//     already reading from this channel before calling Zipr.CreateArchive/Zipr.CreateArchives
//     or make this channel buffered with at least a capacity of 1.
//   - logCh: A channel that receives paths to the files under progress.
//   - algo: The compression algorithm to use for the zip operation.
//     Supported algorithms are defined in the compressionAlgo type.
//
// WARNING: All writes to channels are non-blocking.
//
// Example:
//
//	progressCh := make(chan uint64, 1) // Buffered to ensure total size is not missed
//	logCh := make(chan string) // Buffered/Unbuffered as needed
//	zipper := zipr.Get(progressCh, logCh, zipr.Deflate) // Using DEFLATE compression
//	noReportZipper := zipr.Get(nil, nil, zipr.Store) // No progress reporting
func New(progressCh chan<- uint64, logCh chan<- string, algo compressionAlgo) *Zipr {
	return &Zipr{
		progressCh: progressCh,
		logCh:      logCh,
		lastRead:   time.Now(),
		algo:       algo,
	}
}

// CreateArchive creates a zip archive of the specified files/directories.
//
// If no filenames are provided, it creates a zip archive of the entire root directory.
// The first write to progressChan will be the total size of the archive.
//
// Parameters:
//   - ctx: Context for cancelling the operation - if cancelled, any partially created archive will be deleted
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
//	path, err := zipper.CreateArchive(context.Background(), "/tmp", "backup.zip", "/home/user/documents")
//
//	// Zip specific files within a directory
//	path, err := zipper.CreateArchive(context.Background() ,"/tmp", "partial.zip", "/home/user/documents", "file1.txt", "folder1")
func (z *Zipr) CreateArchive(ctx context.Context, path, archiveName, root string, files ...string) (string, error) {
	archivePath := filepath.Join(path, archiveName)
	archive, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating empty zip archive: %w", err)
	}
	defer func() { _ = archive.Close() }()
	size, err := calculateSize(root, files...)
	if err != nil {
		return "", fmt.Errorf("retrieving filesize: %w", err)
	}

	_ = trySend(z.progressCh, uint64(size)) // report total size

	zw := zip.NewWriter(archive)
	defer func() { _ = zw.Close() }()

	// zip the whole dir if no files are specified
	if len(files) == 0 {
		if err = z.writeDir(ctx, zw, root, root); err != nil {
			_ = archive.Close()
			_ = os.Remove(archivePath) // delete partial written archive, ignore errors
			return "", fmt.Errorf("zipping whole dir: %w", err)
		}
	} else { // zip the specified files
		for _, file := range files {
			filePath := filepath.Join(root, file)
			var stat fs.FileInfo
			if stat, err = os.Stat(filePath); err != nil {
				return "", fmt.Errorf("statting file: %w", err)
			}
			if stat.IsDir() {
				err = z.writeDir(ctx, zw, root, filePath)
			} else {
				// once a file write is happening, we cannot cancel
				// so check for ctx.Done() before writing
				select {
				case <-ctx.Done():
					err = ctx.Err()
				default:
					err = z.writeFile(zw, root, filePath)
				}
			}
			if err != nil {
				_ = archive.Close()
				_ = os.Remove(archivePath) // delete partial written archive, ignore errors
				return "", fmt.Errorf("zipping %q: %w", filePath, err)
			}
		}
	}
	return archivePath, nil
}

// CreateArchives concurrently creates zip archives for multiple directories.
//
// This method creates a separate zip file for each directory specified in the dirs parameter.
// All zipping operations run concurrently using a worker pool for maximum efficiency.
//
// Parameters:
//   - ctx: Context for cancelling the operation - if cancelled, any partially created archive will be deleted
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
//	paths, err := zipper.CreateArchives(context.Background(), "/tmp", "/home/user", "documents", "pictures", "music")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, path := range paths {
//	    fmt.Println("Created archive:", path)
//	}
func (z *Zipr) CreateArchives(ctx context.Context, path, root string, dirs ...string) ([]string, error) {
	if err := checkValidDirs(root, dirs...); err != nil {
		return nil, err
	}
	size, err := calculateSize(root, dirs...)
	if err != nil {
		return nil, fmt.Errorf("retrieving total size of dirs: %w", err)
	}

	_ = trySend(z.progressCh, uint64(size)) // report total size

	zippedDirs := make([]string, len(dirs))
	wp := bgtask.NewWorkerPool(ctx)

main:
	for i, dir := range dirs {
		select {
		case <-wp.Ctx.Done(): // do the cleanup
			for _, zd := range zippedDirs {
				_ = os.Remove(zd) // delete partial written archive, ignore errors
			}
			break main
		default:
			wp.Spawn(func() error {
				dirToZip := filepath.Join(root, dir)
				archiveName := dir + ".zip"
				var archivePath string
				if archivePath, err = z.CreateArchive(ctx, path, archiveName, dirToZip); err != nil {
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
func (z *Zipr) Close() error {
	_ = trySend(z.progressCh, z.read.Load())
	z.read.Store(0)
	close(z.logCh)
	close(z.progressCh)
	return nil
}

// newReader creates a new progressReader that wraps the provided io.Reader.
//
// This is an internal method used to create reader instances that track
// progress during file operations.
func (z *Zipr) newReader(base io.Reader) io.Reader {
	return &progressReader{
		r:    base,
		Zipr: z,
	}
}

// writeDir zips the contents of a directory into the zip archive.
//
// This is an internal method used by CreateArchive to recursively process directories.
//
// Parameters:
//   - w: The zip writer to write to
//   - root: The base path for calculating relative paths
//   - dirPath: The absolute path of the directory to zip
//
// Returns:
//   - An error if the operation fails
func (z *Zipr) writeDir(ctx context.Context, w *zip.Writer, root, dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err != nil || d.IsDir() {
				return err
			}
			return z.writeFile(w, root, path)
		}
	})
}

// writeFile adds a single file to the zip archive.
//
// This is an internal method used by both CreateArchive and writeDir to add files to the archive.
// It maintains the directory structure relative to the root directory.
//
// Parameters:
//   - w: The zip writer to write to
//   - basePath: The base path for calculating relative paths
//   - filePath: The absolute path of the file to add
//
// Returns:
//   - An error if the operation fails
func (z *Zipr) writeFile(w *zip.Writer, basePath, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// report the file we're about to zip
	_ = trySend(z.logCh, filePath)

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("statting file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", filePath)
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
	fh.Method = z.algo

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
// sending progress updates through the parent Zipr's progress channel
// at regular intervals.
type progressReader struct {
	r     io.Reader // Underlying reader
	*Zipr           // Embedded Zipr for progress tracking
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
		_ = trySend(pr.progressCh, pr.read.Load()) // report progress
		pr.lrMu.Lock()
		pr.lastRead = time.Now()
		pr.lrMu.Unlock()
	}
	return
}

// checkValidDirs verifies that all specified paths are valid directories.
//
// This is an internal utility function used by CreateArchives to validate input directories.
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
			return fmt.Errorf("statting dir: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %q is not a directory", dir)
		}
	}
	return nil
}

func trySend[T any](ch chan<- T, v T) bool {
	select {
	case ch <- v:
		return true
	default:
		return false
	}
}

// calculateSize determines the total size in bytes of all specified files/directories.
// If no filenames are provided, it calculates the size of the entire parent directory.
func calculateSize(parentDir string, filenames ...string) (int64, error) {
	if len(filenames) == 0 {
		size, err := calculateDirSize(parentDir)
		return size, err
	}
	size := int64(0)
	for _, name := range filenames {
		path := filepath.Join(parentDir, name)
		info, err := os.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("reading filestat: %w", err)
		}
		if !info.IsDir() {
			size += info.Size()
			continue
		}
		dirSize, err := calculateDirSize(path)
		if err != nil {
			return 0, fmt.Errorf("calculating dir size: %w", err)
		}
		size += dirSize
	}
	return size, nil
}

// calculateDirSize recursively determines the total size of all files within a directory.
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
