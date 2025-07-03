package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	incompleteDownloadKey = ".incd"
)

var (
	once   sync.Once
	client *Client
)

type Progress struct {
	// D: downloaded bytes, T: total bytes
	D, T int64
	// S: speed per sec in bytes
	S int32
}

type DownloadTracker struct {
	// f: underlying writer
	f *os.File
	// d: downloaded bytes, t: total bytes
	d, t atomic.Int64
	// s: speed per second in bytes
	s      atomic.Int32
	pch    chan Progress
	at     time.Time
	ctx    context.Context
	cancel context.CancelFunc
}

func NewDownloadTracker(f string, ch chan Progress) (*DownloadTracker, error) {
	file, size, err := prepareFileForDownload(f)
	if err != nil {
		return nil, fmt.Errorf("preparing file %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	dt := &DownloadTracker{
		f:      file,
		pch:    ch,
		at:     time.Now(),
		ctx:    ctx,
		cancel: cancel,
	}

	dt.d.Store(size) // how much is it already downloaded
	go dt.trackPerSec()

	return dt, nil
}

func (dt *DownloadTracker) Write(p []byte) (n int, err error) {
	n, err = dt.f.Write(p)
	if err != nil {
		return
	}
	dt.d.Add(int64(n))

	if time.Since(dt.at).Milliseconds() > 500 {
		dt.at = time.Now()
		dt.trySend()
	}

	return
}

func (dt *DownloadTracker) Close() error {
	dt.cancel()
	dt.trySend() // send the last progress update
	close(dt.pch)
	if err := dt.f.Close(); err != nil {
		return err
	}

	total := dt.t.Load()
	if total > 0 && dt.d.Load() != total {
		return nil
	}

	// if the file is fully downloaded, rename it to remove the incomplete download key
	final := strings.TrimSuffix(dt.f.Name(), incompleteDownloadKey)
	// check if the final file already exists
	if _, err := os.Stat(final); err == nil { // is nil
		// if such file exists, generate a unique name
		final = generateUniqueFileName(final)
	}
	if err := os.Rename(dt.f.Name(), final); err != nil {
		return fmt.Errorf("removing %q suffix from dowloaded file: %w", incompleteDownloadKey, err)
	}

	return nil
}

func (dt *DownloadTracker) trackPerSec() {
	t := time.NewTicker(time.Second)
	for {
		prev := dt.d.Load()
		select {
		case <-dt.ctx.Done():
			return
		case <-t.C:
			curr := dt.d.Load()
			ps := max(0, curr-prev)
			dt.s.Store(int32(ps))
		}
	}
}

func (dt *DownloadTracker) trySend() bool {
	p := Progress{
		D: dt.d.Load(),
		T: dt.t.Load(),
		S: dt.s.Load(),
	}
	select {
	case dt.pch <- p:
		return true
	default:
		return false
	}
}

type Client struct {
	mdns *mdns.MDNS
	c    http.Client
}

func Get() *Client {
	once.Do(func() {
		client = &Client{
			mdns: mdns.Get(),
			c:    http.Client{}, // same network, low latency expected
		}
	})
	return client
}

func (c *Client) IndexFiles(instance string) ([]*domain.FileInfo, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := client.newRequest(ctx, instance, http.MethodGet, "", nil)
	if err != nil {
		return nil, -1, fmt.Errorf("creating request: %v", err)
	}

	resp, err := c.c.Do(req)
	var urlErr *url.Error
	if err != nil {
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return nil, http.StatusRequestTimeout, fmt.Errorf("indexing files: request timed out")
		}
		return nil, -1, fmt.Errorf("indexing files: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("server returned status %d while indexing directory", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, fmt.Errorf("reading directory index response: %v", err)
	}

	var r map[string][]*domain.FileInfo
	if err = json.Unmarshal(b, &r); err != nil {
		return nil, -1, fmt.Errorf("parsing directory index JSON: %v", err)
	}

	return r["fileIndexes"], resp.StatusCode, nil
}

func (c *Client) DownloadFile(instance string, accessID uint32, dst *DownloadTracker) (int, error) {
	path := fmt.Sprintf("/%d", accessID)
	var status int
	b := make([]byte, 1<<20) // 1MB buffer size

	size, err := c.getFileSize(instance, path)
	if err != nil {
		return -1, fmt.Errorf("getting file size: %v", err)
	}
	dst.t.Store(size) // set total size of the file

	for i := 1; i <= 5; i++ {
		var req *http.Request
		req, err = c.newRequest(context.Background(), instance, http.MethodGet, path, nil)
		if err != nil {
			return -1, fmt.Errorf("creating request: %v", err)
		}

		status, err = c.downloadFile(req, dst, b)
		if err != nil {
			return status, err
		}

		if status == http.StatusRequestTimeout {
			exponentialBackoff(i, 30*time.Second) // exponential backoff
			continue
		}
		break
	}
	return status, nil
}

func (c *Client) getFileSize(instance, path string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := c.newRequest(ctx, instance, http.MethodHead, path, nil)
	if err != nil {
		return -1, fmt.Errorf("creating request: %v", err)
	}
	resp, err := c.c.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return -1, fmt.Errorf("request timed out")
		}
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return resp.ContentLength, nil
}

func (c *Client) downloadFile(req *http.Request, dst *DownloadTracker, buffer []byte) (int, error) {
	// in case of resume
	startRange := dst.d.Load() // how much is already downloaded
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startRange))
	resp, err := c.c.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return http.StatusRequestTimeout, nil
		}
		return -1, fmt.Errorf("downloading file: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("server returned status %d while downloading file", resp.StatusCode)
	}

	// Read the response body and write to the tracker
	if _, err = io.CopyBuffer(dst, resp.Body, buffer); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe) { // retryable errors
			return http.StatusRequestTimeout, nil
		}
		return -1, fmt.Errorf("copying resp body to file: %v", err)
	}

	return resp.StatusCode, nil
}

func (c *Client) StopServer(instance string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := c.newRequest(ctx, instance, http.MethodPost, "/stop", nil)
	if err != nil {
		return -1, fmt.Errorf("creating request: %v", err)
	}

	resp, err := c.c.Do(req)
	var urlErr *url.Error
	if err != nil {
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return http.StatusRequestTimeout, nil
		}
		return -1, fmt.Errorf("stopping server: %v", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	var response struct {
		Status string `json:"status"`
	}
	if err = json.Unmarshal(b, &response); err != nil {
		return -1, fmt.Errorf("parsing server response while stopping: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusForbidden &&
		resp.StatusCode != http.StatusConflict {
		return resp.StatusCode, fmt.Errorf("server returned status %d while stopping server", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (c *Client) newRequest(ctx context.Context, instance, method, path string, body io.Reader) (*http.Request, error) {
	/*entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}
	addr := fmt.Sprintf("http://%s:%d%s", entry.IP, entry.Port, path)*/
	addr := fmt.Sprintf("http://%s:%d%s", "192.168.100.39", 80, path)
	uname, err := c.getClientUsername()
	if err != nil {
		return nil, fmt.Errorf("retrieving client username: %v", err)
	}

	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Requested-By", uname)
	return req, nil
}

func (c *Client) getClientUsername() (string, error) {
	cfg, err := config.Get()
	if err != nil {
		cfg, _ = config.Load()
	}
	return cfg.Personal.Username, nil
}

func prepareFileForDownload(f string) (*os.File, int64, error) {
	// with incomplete download key, it ensures either it's a new download or a resume
	f += incompleteDownloadKey
	file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, 0, err
	}
	// Get the current size of the file to determine the starting point for download
	fi, err := file.Stat()
	if err != nil {
		cErr := file.Close()
		return nil, 0, fmt.Errorf("retriving filesize: %file", errors.Join(err, cErr))
	}
	return file, fi.Size(), nil
}

func generateUniqueFileName(name string) string {
	ext := filepath.Ext(name)
	if filepath.Base(name) == ext {
		ext = ""
	}
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s-%06d%s", base, time.Now().UnixNano()%1_000_000, ext)
}

func exponentialBackoff(attempt int, maxDelay time.Duration) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	jitter := time.Duration(rand.IntN(int(time.Second)))
	delay += jitter
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}
