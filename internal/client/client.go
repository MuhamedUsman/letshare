package client

import (
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
		mdns: mdns.New(),
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) IndexDirectory(instance string) ([]*domain.FileInfo, error) {
	entries := c.mdns.Entries()
	hostname, ok := entries[instance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in mDNS entries", instance)
	}
	resp, err := c.http.Get("http://" + hostname)
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
