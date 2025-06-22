package server

import (
	"context"
	"encoding/json"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/util"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// The first step will be to take any numbers of files and srv them
func TestCopyFilesToDir(t *testing.T) {
	tempDir1 := t.TempDir()
	fileContents := "test content"
	files := createTempFiles(t, 3, tempDir1, fileContents)
	s := New()

	// Test normal copying
	t.Log("Copying files to temp dir", "files", len(files), "tempDir1", tempDir1)
	ch := s.CopyFilesToDir(tempDir1, files...)
	var count int
	for stat := range ch {
		if stat.Err != nil {
			t.Fatal(stat.Err.Error())
		}
		count++
	}

	// Verify all files were copied
	if count != len(files) {
		t.Fatalf("Expected %d files to be copied, got %d", len(files), count)
	}

	// Verify files exist in destination
	for _, f := range files {
		destPath := filepath.Join(tempDir1, filepath.Base(f))
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Fatalf("File %s was not copied to destination", f)
		}
	}

	// Test cancellation during copy (cleanup)
	tempDir2 := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	s2 := &Server{
		BT:            util.NewBgTask(),
		StopCtx:       ctx,
		StopCtxCancel: cancel,
		mu:            new(sync.Mutex),
	}

	// Cancel after first file
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	ch = s2.CopyFilesToDir(tempDir2, files...)
	for range ch {
		// Just drain the channel
	}
}

func TestSrvDir(t *testing.T) {
	tempDir := t.TempDir()
	s := New()

	fileContents := "test content"
	tempFiles := createTempFiles(t, 3, tempDir, fileContents)

	ch := s.CopyFilesToDir(tempDir, tempFiles...)
	for stat := range ch {
		if stat.Err != nil {
			t.Fatal(stat.Err)
		}
	}

	server := httptest.NewServer(s.routes(tempDir))
	defer server.Close()

	// Test directory listing
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string][]domain.FileInfo
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	files, ok := response["directoryIndex"]
	if !ok {
		t.Fatal("Expected 'directoryIndex' in response")
	}

	if len(files) != len(tempFiles) {
		t.Fatalf("Expected %d files, got %d", len(tempFiles), len(files))
	}

}

func TestServer_Stop(t *testing.T) {
	bt := util.NewBgTask()
	ctx1, cancel1 := context.WithCancel(t.Context())
	ctx2, cancel2 := context.WithCancel(t.Context())
	ctx3, cancel3 := context.WithCancel(t.Context())
	cases := map[string]Server{
		"unstoppable": {false, ctx1, cancel1, bt, &sync.Mutex{}, 0},
		"stoppable":   {true, ctx2, cancel2, bt, &sync.Mutex{}, 0},
		"active":      {true, ctx3, cancel3, bt, &sync.Mutex{}, 10},
	}
	for name, server := range cases {
		t.Run(name, func(t *testing.T) {
			// Create a request to pass to our handler
			req, err := http.NewRequest("GET", "/stop", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(server.stopHandler)
			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check status code and response body based on the case
			switch name {
			case "unstoppable":
				// Expect not stoppable error
				if rr.Code != http.StatusForbidden && rr.Code != http.StatusMethodNotAllowed {
					t.Errorf("handler returned wrong status code: got %v want %v or %v",
						rr.Code, http.StatusConflict, http.StatusMethodNotAllowed)
				}

			case "active":
				// Expect not idle error
				if rr.Code != http.StatusConflict && rr.Code != http.StatusServiceUnavailable {
					t.Errorf("handler returned wrong status code: got %v want %v or %v",
						rr.Code, http.StatusConflict, http.StatusServiceUnavailable)
				}

			case "stoppable":
				// Expect success
				if rr.Code != http.StatusAccepted {
					t.Errorf("handler returned wrong status code: got %v want %v",
						rr.Code, http.StatusAccepted)
				}

				// Check if cancel function was called
				select {
				case <-ctx2.Done():
					// Context was canceled, which is expected
				default:
					t.Errorf("stopHandler did not cancel the context")
				}

				// Check response status
				expected := "Shutdown initiated, it may take maximum of 10 seconds to shutdown the server."
				// Read the response body
				var responseBody []byte
				responseBody, err = io.ReadAll(rr.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Parse the JSON response
				var response map[string]any
				if err = json.Unmarshal(responseBody, &response); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}

				// Check if message matches expected
				message, exists := response["status"]
				if !exists {
					t.Errorf("Response does not contain 's' field: %s", string(responseBody))
				} else if msg, ok := message.(string); !ok || msg != expected {
					t.Errorf("Expected message '%s', got '%v'", expected, message)
				}
			}
		})
	}
}

func createTempFiles(t *testing.T, n int, tempDir, content string) []string {
	files := make([]string, 3)
	for i := range n {
		f, err := os.CreateTemp(tempDir, "temp-*.txt")
		if err != nil {
			t.Errorf("failed to create temp file: %v", err)
		}
		if _, err = f.Write([]byte(content)); err != nil {
			t.Errorf("failed to write to temp file: %v", err)
		}
		_ = f.Close()
		files[i] = f.Name()
	}
	return files
}
