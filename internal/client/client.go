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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const IncompleteDownloadKey = ".incd"

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
	// finalName of the file after download is complete
	finalName string
	// d: downloaded bytes, t: total bytes
	d, t atomic.Int64
	// s: speed per second in bytes
	s                     atomic.Int32
	pch                   chan Progress
	at                    time.Time
	isTracking, firstSend bool
	ctx                   context.Context
	cancel                context.CancelFunc
}

// NewDownloadTracker prepares a file for download and returns a DownloadTracker.
func NewDownloadTracker(f string, ch chan Progress) (*DownloadTracker, error) {
	file, size, err := prepareFileForDownload(f)
	if err != nil {
		return nil, fmt.Errorf("preparing file %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	dt := &DownloadTracker{
		f:         file,
		pch:       ch,
		at:        time.Now(),
		firstSend: true,
		ctx:       ctx,
		cancel:    cancel,
	}
	dt.d.Store(size) // how much is it already downloaded
	return dt, nil
}

// Filename returns the name of the file (downloading or downloaded),
// use this before dereferencing the DownloadTracker.
func (dt *DownloadTracker) Filename() string {
	if dt.finalName != "" {
		return dt.finalName
	}
	return dt.f.Name()
}

func (dt *DownloadTracker) Write(p []byte) (n int, err error) {
	if dt.ctx.Err() != nil {
		return 0, dt.ctx.Err() //// return early if context is cancelled
	}

	// start tracking on first write
	if !dt.isTracking {
		go dt.trackPerSec()
		dt.isTracking = true
	}

	n, err = dt.f.Write(p)
	if err != nil {
		return
	}
	dt.d.Add(int64(n))

	if time.Since(dt.at).Milliseconds() > 250 || dt.firstSend {
		if dt.firstSend {
			dt.firstSend = false
		}
		dt.at = time.Now()
		dt.trySend(false)
	}

	return
}

func (dt *DownloadTracker) Close() error {
	dt.cancel()
	// if total size is set, send the last progress update
	// or else it may be a failed resume attempt, if t is not set
	if dt.t.Load() > 0 {
		dt.trySend(true) // send the last progress update
	}
	close(dt.pch)
	if err := dt.f.Close(); err != nil {
		return err
	}

	// if file is fully downloaded
	total := dt.t.Load()
	if dt.d.Load() == total && total > 0 {
		// rename it to remove the incomplete download key
		final := strings.TrimSuffix(dt.f.Name(), IncompleteDownloadKey)
		// check if the final file already exists, then generate a unique name
		if _, err := os.Stat(final); err == nil { // is nil
			// if such file exists, generate a unique name
			final = generateUniqueFileName(final)
		}
		if err := os.Rename(dt.f.Name(), final); err != nil {
			return fmt.Errorf("removing %q suffix from dowloaded file: %w", IncompleteDownloadKey, err)
		}
		dt.finalName = final
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

func (dt *DownloadTracker) trySend(force bool) bool {
	p := Progress{
		D: dt.d.Load(),
		T: dt.t.Load(),
		S: dt.s.Load(),
	}
	if force {
		select {
		case dt.pch <- p:
			return true
		case <-dt.ctx.Done():
			return false // context is cancelled, do not send
		}
	}
	select {
	case dt.pch <- p:
		return true
	case <-dt.ctx.Done():
		return false
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
		var proto http.Protocols
		proto.SetUnencryptedHTTP2(true)
		client = &Client{
			mdns: mdns.Get(),
			c: http.Client{Transport: &http.Transport{
				DisableCompression: true,
				Protocols:          &proto,
			}},
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

func (c *Client) DownloadFile(dst *DownloadTracker, instance string, accessID uint32) (int, error) {
	path := fmt.Sprintf("/%d", accessID)
	statusCode, size, err := c.getFileSize(instance, path)
	if err != nil {
		return -1, unwrapErr(err)
	}
	if statusCode != http.StatusOK {
		return statusCode, nil
	}
	dst.t.Store(size) // set total size of the file

	var status int
	status, err = c.downloadFile(dst, instance, path)
	if err != nil {
		s := "server sent GOAWAY and closed the connection"
		if strings.Contains(err.Error(), s) {
			return -1, errors.New(s)
		}
		return -1, unwrapErr(err)
	}
	return status, nil
}

func (c *Client) getFileSize(instance, path string) (statusCode int, size int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := c.newRequest(ctx, instance, http.MethodHead, path, nil)
	if err != nil {
		return -1, -1, fmt.Errorf("creating request: %w", err)
	}
	resp, err := c.c.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return http.StatusRequestTimeout, -1, nil
		}
		return -1, -1, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.ContentLength, nil
}

func (c *Client) downloadFile(dst *DownloadTracker, instance, path string) (int, error) {
	req, err := c.newRequest(context.Background(), instance, http.MethodGet, path, nil)
	if err != nil {
		return -1, fmt.Errorf("creating request: %w", err)
	}

	// in case of resume
	startRange := dst.d.Load() // how much is already downloaded
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startRange))

	resp, err := c.c.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return http.StatusRequestTimeout, nil
		}
		return -1, err
	}
	defer resp.Body.Close()

	// Check status code, and early return if not successful
	if resp.StatusCode >= 400 {
		return resp.StatusCode, nil
	}

	b := make([]byte, 1<<20) // 1 MiB buffer
	// Read the response body and write to the tracker
	if _, err = io.CopyBuffer(dst, resp.Body, b); err != nil {
		return -1, fmt.Errorf("copying resp body to file: %w", err)
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
	entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return nil, fmt.Errorf("instance %q is currently offline", instance)
	}
	addr := fmt.Sprintf("http://%s:%d%s", entry.IP, entry.Port, path)
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
	f += IncompleteDownloadKey
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

func unwrapErr(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}
