package server

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/common"
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
		BT: common.NewBackgroundTask(),
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
		BT: common.NewBackgroundTask(),
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
	server := httptest.NewServer(s.createSrvDirHandler(tempDir))
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
	// service & domain are hardcoded in the PublishEntry function
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
