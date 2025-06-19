package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	mdns *mdns.MDNS
	http *http.Client
}

func New() *Client {
	return &Client{
		mdns: mdns.Get(),
		http: &http.Client{Timeout: 2 * time.Second}, // same network, low latency expected
	}
}

func (c *Client) IndexFiles(instance string) ([]*domain.FileInfo, error) {
	entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}
	resp, err := c.http.Get("http://" + entry.IP)
	var urlErr *url.Error
	if err != nil {
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return nil, fmt.Errorf("directory indexing: request timed out")
		}
		return nil, fmt.Errorf("requesting directory index: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d while indexing directory", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading directory index response: %v", err)
	}

	var dir struct {
		Indexes []*domain.FileInfo `json:"directoryIndex"`
	}
	if err = json.Unmarshal(bytes, &dir); err != nil {
		return nil, fmt.Errorf("parsing directory index JSON: %v", err)
	}

	return dir.Indexes, nil
}

func (c *Client) StopServer(instance string) (int, error) {
	entries := c.mdns.Entries()
	entry, ok := entries[instance]
	if !ok {
		return -1, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}
	resp, err := c.http.Post("http://"+entry.Hostname+stop, "application/json", new(bytes.Buffer))
	var urlErr *url.Error
	if err != nil {
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return http.StatusRequestTimeout, nil
		}
		return -1, fmt.Errorf("stopping server: %v", err)
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	var response struct {
		Status string `json:"status"`
	}
	if err = json.Unmarshal(bytes, &response); err != nil {
		return -1, fmt.Errorf("parsing server response while stopping: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusForbidden &&
		resp.StatusCode != http.StatusConflict {
		return resp.StatusCode, fmt.Errorf("server returned status %d while stopping server", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
