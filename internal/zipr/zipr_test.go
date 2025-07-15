package zipr

import (
	"archive/zip"
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestZipr_CreateArchives(t *testing.T) {

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	p, l := make(chan uint64, 1), make(chan string, 1)
	z := New(ctx, p, l, Store)
	parent, dirs, expSize, expCount := createDirWithFiles(t, 5, 3)

	// Calculate expected paths for verification
	expectedPaths := getExpectedPaths(t, dirs, parent)

	// Zipr.CreateArchives() also tests Zipr.CreateArchive() implicitly
	archives, err := z.CreateArchives(t.TempDir(), parent, dirs...)
	assert.NoError(t, err, "creating archives should not return an error")
	assert.NoError(t, z.Close(), "closing zipper should not return an error")

	t.Run("Test Archive Contents", func(t *testing.T) {
		//  check if expCount of archives same as passed
		assert.Equal(t, len(dirs), len(archives), "expected no of dirs must match no of archives created")

		var size uint64
		var count int
		var paths []string
		for _, a := range archives {
			// check if archive exists
			assert.FileExists(t, a, "archive %s does not exist", a)
			// check if archive is valid zip
			r, err := zip.OpenReader(a)
			assert.NoError(t, err, "opening archive should not return an error")

			count += len(r.File)
			for _, f := range r.File {
				size += f.FileHeader.UncompressedSize64
				// append only files
				if !strings.HasSuffix(f.Name, "/") {
					paths = append(paths, f.Name)
				}
			}
			assert.NoError(t, r.Close(), "closing archive reader should not return an error")
		}
		slices.Sort(paths)

		assert.Exactly(t, expSize, size)
		assert.Exactly(t, expCount, count)
		assert.Exactly(t, expectedPaths, paths)
	})

	// test the progress to report the total size at first read
	// and to report the total size at the end when chan is closed
	//
	// for log channel, we just check if it read once, that is zipper is reporting logs
	t.Run("Test progress channel", func(t *testing.T) {
		isFirst := true
		var lastProg uint64
		for prog := range p {
			if isFirst {
				assert.Equalf(t, expSize, prog, "expected first progress %d, got %d", expSize, prog)
				isFirst = false
			}
			lastProg = prog
		}
		if isFirst {
			assert.Fail(t, "progress channel did not report any progress")
			return
		}
		assert.Equalf(t, expSize, lastProg, "expected last progress %d updates, got %d", expSize, lastProg)
	})

	t.Run("Test log channel", func(t *testing.T) {
		var read bool
		for range l {
			read = true
			break
		}
		assert.True(t, read, "log channel did not report any logs")
	})

}

func getExpectedPaths(t *testing.T, dirs []string, parent string) []string {
	expectedPaths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = filepath.Join(parent, dir)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				var rel string
				rel, err = filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				expectedPaths = append(expectedPaths, rel)
			}
			return nil
		})
		assert.NoError(t, err, "walking directory should not return an error")
	}
	slices.Sort(expectedPaths)
	return expectedPaths
}

// createDirWithFiles create the no of files specified in each dir
// it also nests the same dir inside each odd dir.
// Returns:
//   - parent: the parent directory where all dirs are created
//   - subPaths: the names of the created directories
//   - size: total size of all files created
//   - count: total number of files created
func createDirWithFiles(t *testing.T, dirs, files int) (parent string, subPaths []string, size uint64, count int) {
	parent = t.TempDir()
	subPaths = make([]string, dirs)
	// creating dirs
	for i := range dirs {
		p, err := os.MkdirTemp(parent, "dir-*")
		assert.NoError(t, err, "creating temp directory should not return an error")

		subPaths[i] = filepath.Base(p)
		// create files in the dir
		for range files {
			size += uint64(createTempFile(t, p))
			count++
		}
		// create nested dir if odd
		if i%2 != 0 {
			p, err = os.MkdirTemp(p, "nested-*")
			assert.NoError(t, err, "creating nested directory should not return an error")
			size += uint64(createTempFile(t, p))
			count++
		}
	}
	return
}

func createTempFile(t *testing.T, p string) int {
	f, err := os.CreateTemp(p, "file-*.txt")
	assert.NoError(t, err, "creating temp file should not return an error")
	defer func() {
		assert.NoError(t, f.Close(), "closing temp file should not return an error")
	}()
	// write name of the file to it
	n, err := f.WriteString(f.Name())
	assert.NoError(t, err, "writing to temp file should not return an error")
	return n
}
