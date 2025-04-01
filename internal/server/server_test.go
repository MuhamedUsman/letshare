package server

import (
	"context"
	"encoding/json"
	"github.com/MuhamedUsman/letshare/internal/util"
	"github.com/grandcat/zeroconf"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

// The first step will be to take any numbers of files and srv them
func TestCopyFilesToDir(t *testing.T) {
	tempDir := t.TempDir()
	files := createTempFiles(t, 3, tempDir)
	s := &Server{
		BT: util.NewBgTask(),
	}
	var ch CopyStatChan
	t.Log("Copying files to temp dir", "files", len(files), "tempDir", tempDir)
	ch = s.CopyFilesToDir(tempDir, files...)
	for {
		stat, ok := <-ch
		if !ok {
			break
		}
		if stat.Err != nil {
			t.Fatal(stat.Err.Error())
		}
		t.Log("Copied File: ", "N", stat.N)
	}
}

func TestSrvDir(t *testing.T) {
	tempDir := t.TempDir()
	s := &Server{
		BT: util.NewBgTask(),
	}
	tempFiles := createTempFiles(t, 3, tempDir)
	ch := s.CopyFilesToDir(tempDir, tempFiles...)
	for {
		stat, ok := <-ch
		if !ok {
			break
		}
		if stat.Err != nil {
			t.Fatal(stat.Err.Error())
		}
	}
	server := httptest.NewServer(s.routes(tempDir))
	defer server.Close()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	readBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(readBytes))
}

func TestPublishEntry(t *testing.T) {
	ctx, cancel1 := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	defer cancel1()
	instance := "Letshare"
	service := "_http._tcp"
	domain := "local."
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := PublishEntry(ctx, instance, "Testing..."); err != nil {
			cancel1()
			t.Error(err)
			t.Fail()
		}
	}()
	time.Sleep(50 * time.Millisecond) // wait for it to publish
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		t.Fatal(err)
	}
	entriesCh := make(chan *zeroconf.ServiceEntry)
	entries := make([]*zeroconf.ServiceEntry, 0)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-entriesCh:
				if !ok {
					cancel1()
					return
				}
				if entry != nil {
					entries = append(entries, entry)
				}
			}
		}
	}()
	// service & domain are hardcoded in the Publish function
	tctx, cancel2 := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel2()
	if err = resolver.Lookup(tctx, instance, service, domain, entriesCh); err != nil {
		cancel1()
		cancel2()
		t.Fatal(err)
	}
	wg.Wait()
	if slices.ContainsFunc(entries, func(entry *zeroconf.ServiceEntry) bool {
		if entry.Instance == instance &&
			entry.Domain == domain &&
			entry.Service == service &&
			entry.Port == 80 {
			return true
		}
		return false
	}) {
		t.Log("Entry Found")
	} else {
		t.Error("No entry found for ", instance, service, domain, entries)
		t.Fail()
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
			handler := http.HandlerFunc(server.Stop)
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
					t.Errorf("Stop did not cancel the context")
				}

				// Check response status
				expected := "Shutdown initiated, it may take maximum of 10 seconds to shutdown the server."
				// Read the response body
				responseBody, err := io.ReadAll(rr.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Parse the JSON response
				var response map[string]interface{}
				if err := json.Unmarshal(responseBody, &response); err != nil {
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

func createTempFiles(t *testing.T, n int, tempDir string) []string {
	files := make([]string, 3)
	for i := range n {
		f, err := os.CreateTemp(tempDir, "temp-*.txt")
		if err != nil {
			t.Errorf("failed to create temp file: %v", err)
		}
		if _, err = f.Write([]byte("test content")); err != nil {
			t.Errorf("failed to write to temp file: %v", err)
		}
		_ = f.Close()
		files[i] = f.Name()
	}
	return files
}
