package fetcher

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	http   *http.Client
	sem    chan struct{}
	logger *slog.Logger
}

func New(maxConcurrent int) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		sem:    make(chan struct{}, maxConcurrent),
		logger: slog.Default(),
	}
}

func (c *Client) acquire() { c.sem <- struct{}{} }
func (c *Client) release() { <-c.sem }

// Do performs an HTTP request with semaphore-based concurrency limiting
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.acquire()
	defer c.release()
	return c.http.Do(req)
}

// DoWithRetry performs the request with up to maxRetries on transient errors
func (c *Client) DoWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i-1)) * 500 * time.Millisecond
			time.Sleep(backoff)
		}
		resp, err := c.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error %s", resp.Status)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
