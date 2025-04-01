package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"io"
	"log"
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
		return nil, fmt.Errorf("IndexDirectory: instance %q not found in mDNS entries", instance)
	}
	resp, err := c.http.Get("http://" + hostname)
	var urlErr *url.Error
	if err != nil && errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return nil, fmt.Errorf("IndexDirectory: request timed out")
		}
	} else {
		return nil, fmt.Errorf("IndexDirectory: requesting directory index: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IndexDirectory: server returned status %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("IndexDirectory: reading response body: %v", err)
	}

	var dirIdx struct {
		Idx []*domain.FileInfo `json:"directoryIndex"`
	}
	if err = json.Unmarshal(bytes, &dirIdx); err != nil {
		return nil, fmt.Errorf("IndexDirectory: unmarshalling response body: %v", err)
	}

	for _, file := range dirIdx.Idx {
		log.Println(file)
	}

	return dirIdx.Idx, nil
}
