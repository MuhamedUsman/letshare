package file

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestZip(t *testing.T) {
	subDirs := 5
	filesPerDir := 1
	root, files := createFilesToZip(t, subDirs, filesPerDir)

	progressChan := make(chan int64)
	isTotalSize := true
	var total int64

	go func() {
		for i := range progressChan {
			if isTotalSize {
				total = i
				isTotalSize = false
			} else {
				fmt.Printf("\rprogress: %s/%s ", HumanizeSize(i), HumanizeSize(total))
			}
		}
	}()

	archiveZipPath, err := Zip(progressChan, "test.zip", root, files...)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(archiveZipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	r, err := zip.NewReader(f, stat.Size())
	if err != nil {
		t.Fatal(err)
	}

	// Collect expected files and their contents
	expected := make(map[string]string)
	for _, name := range files {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				filePath := filepath.Join(path, entry.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				relPath, err := filepath.Rel(root, filePath)
				if err != nil {
					t.Fatal(err)
				}
				expected[relPath] = string(content)
			}
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expected[name] = string(content)
		}
	}

	// Check archive structure and contents
	found := make(map[string]bool)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open file %q in archive: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read file %q in archive: %v", f.Name, err)
		}
		want, ok := expected[f.Name]
		if !ok {
			t.Errorf("unexpected file in archive: %q", f.Name)
			continue
		}
		if want != string(data) {
			t.Errorf("content mismatch for %q: want %q, got %q", f.Name, want, string(data))
		}
		found[f.Name] = true
	}

	// Ensure all expected files are present
	for name := range expected {
		if !found[name] {
			t.Errorf("expected file %q not found in archive", name)
		}
	}
}

// createFilesToZip first creates files in root and then creates files in subDirs
// and returns the root directory and the list of files created in root and subDirs
func createFilesToZip(t *testing.T, subDirs, filesPerDir int) (root string, files []string) {
	root = t.TempDir()
	// first, create files in root
	files = append(files, createFiles(t, filesPerDir, root)...)
	// create files in subDirs
	for range subDirs {
		dir, err := os.MkdirTemp(root, "*")
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, filepath.Base(dir))
		createFiles(t, filesPerDir, dir)
	}
	return
}

func createFiles(t *testing.T, n int, root string) []string {
	files := make([]string, n)
	for i := range n {
		f, err := os.CreateTemp(root, "*.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.WriteString(f.Name()); err != nil {
			t.Fatal(err)
		}
		files[i] = filepath.Base(f.Name())
		f.Close()
	}
	return files
}
