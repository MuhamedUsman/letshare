package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	stop = "/stop"
)

type Client struct {
	http *http.Client
	mdns *mdns.MDNS
}

func New() *Client {
	return &Client{
		http: &http.Client{Timeout: 2 * time.Second}, // same network, low latency expected
		mdns: mdns.Get(),
	}
}

func (c *Client) IndexFiles(instance string) ([]*domain.FileInfo, error) {
	entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}

	addr := fmt.Sprintf("http://%s:%d", entry.IP, entry.Port)
	uname, err := c.getClientUsername()
	if err != nil {
		return nil, fmt.Errorf("getting client username: %v", err)
	}

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("X-Requested-By", uname)

	resp, err := c.http.Do(req)
	var urlErr *url.Error
	if err != nil {
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return nil, fmt.Errorf("indexing files: request timed out")
		}
		return nil, fmt.Errorf("indexing files: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d while indexing directory", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading directory index response: %v", err)
	}

	var r map[string][]*domain.FileInfo
	if err = json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parsing directory index JSON: %v", err)
	}

	return r["fileIndexes"], nil
}

func (c *Client) StopServer(instance string) (int, error) {
	entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return -1, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}

	addr := fmt.Sprintf("http://%s:%d%s", entry.IP, entry.Port, stop)
	uname, err := c.getClientUsername()
	if err != nil {
		return -1, fmt.Errorf("getting client username: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, addr, nil)
	if err != nil {
		return -1, fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("X-Requested-By", uname)
	resp, err := c.http.Do(req)

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

func (c *Client) getClientUsername() (string, error) {
	cfg, err := config.Get()
	if err != nil {
		cfg, _ = config.Load()
	}
	return cfg.Personal.Username, nil
}
