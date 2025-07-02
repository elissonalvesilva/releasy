package httpclient

import (
	"net/http"
	"time"
)

type Client struct {
	client *http.Client
}

func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) Get(url string) (*http.Response, error) {
	return c.client.Get(url)
}
