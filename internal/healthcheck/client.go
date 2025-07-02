package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/elissonalvesilva/releasy/pkg/httpclient"
	"github.com/elissonalvesilva/releasy/pkg/logger"
)

type (
	httpHealthChecker struct {
		client *httpclient.Client
	}

	HealthChecker interface {
		Ping(ctx context.Context, url string, port int, intervalSeconds int) error
	}
)

const (
	uri = "http://%s:%d/ping"
)

func NewHTTPHealthChecker(c *httpclient.Client) *httpHealthChecker {
	return &httpHealthChecker{
		client: c,
	}
}

func (h *httpHealthChecker) Ping(ctx context.Context, serviceName string, port int, intervalSeconds int) error {
	url := h.buildURL(serviceName, port)
	logger.WithField("url", url).Info("starting healthcheck")

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("healthcheck canceled or timed out: %w", ctx.Err())

		case <-ticker.C:
			resp, err := h.client.Get(url)
			if err != nil {
				logger.Warn("Error pinging service, retrying...", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				logger.WithField("status", resp.StatusCode).Info("Ping OK! Service is healthy.")
				return nil
			}

			logger.WithField("status", resp.StatusCode).Warn("Ping returned unexpected status, retrying...")
		}
	}
}

func (h *httpHealthChecker) buildURL(host string, port int) string {
	return fmt.Sprintf(uri, host, port)
}
