package client

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/util"
	"io"
	"net/http"
)

const (
	base = "/"
	stop = base + "stop"
)

type Client struct {
	BT *util.BackgroundTask
}

func New() *Client {
	return &Client{BT: util.NewBgTask()}
}

func (c *Client) IndexDirectory() ([]*domain.FileInfo, error) {
	resp, err := http.Get()
	if err != nil {
		return nil, fmt.Errorf("creating request while indexing directory: %v", err)
	}
	io.ReadAll(resp.Body)
}
