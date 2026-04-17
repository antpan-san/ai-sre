// Package transport provides the HTTP/HTTPS communication layer for server interaction.
// It defines the ServerAPI interface and a concrete HTTPClient implementation.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"ft-client/internal/config"
	"ft-client/internal/model"
)

// ServerAPI defines the interface for all server communication.
// This interface makes the transport layer testable and swappable.
type ServerAPI interface {
	// SendHeartbeat posts heartbeat data and returns the server's response.
	SendHeartbeat(ctx context.Context, req *model.HeartbeatRequest) (*model.HeartbeatResponse, error)

	// ReportResult sends a command execution result back to the server.
	ReportResult(ctx context.Context, result *model.CommandResult) error

	// PostLog sends a single log line to the server while a task is executing.
	// Non-fatal: callers should log and continue if this returns an error.
	PostLog(ctx context.Context, entry *model.LogEntry) error
}

// HTTPClient implements ServerAPI over HTTP/HTTPS.
type HTTPClient struct {
	baseURL    string
	authToken  string // Bearer token for API authentication
	httpClient *http.Client
}

// NewHTTPClient creates a properly configured HTTP client with TLS support.
func NewHTTPClient(cfg *config.Config) (*HTTPClient, error) {
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build TLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &HTTPClient{
		baseURL:   cfg.Server.URL,
		authToken: cfg.Auth.Token,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

// SendHeartbeat posts heartbeat data and returns the server's response.
func (c *HTTPClient) SendHeartbeat(ctx context.Context, req *model.HeartbeatRequest) (*model.HeartbeatResponse, error) {
	var resp model.HeartbeatResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/heartbeats", req, &resp); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}
	return &resp, nil
}

// ReportResult sends a command execution result back to the server.
func (c *HTTPClient) ReportResult(ctx context.Context, result *model.CommandResult) error {
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/task/report", result, nil); err != nil {
		return fmt.Errorf("report result: %w", err)
	}
	return nil
}

// PostLog sends a single log line to the server while a task is executing.
func (c *HTTPClient) PostLog(ctx context.Context, entry *model.LogEntry) error {
	// Use a short per-call timeout so a slow server doesn't block task execution.
	logCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := c.doJSON(logCtx, http.MethodPost, "/api/v1/task/log", entry, nil); err != nil {
		return fmt.Errorf("post log: %w", err)
	}
	return nil
}

// doJSON is a generic helper that marshals the request body, sends the HTTP request,
// and optionally unmarshals the response into the provided target.
func (c *HTTPClient) doJSON(ctx context.Context, method, path string, body interface{}, target interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Inject authentication token if configured
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if target != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// buildTLSConfig constructs a *tls.Config based on the application configuration.
func buildTLSConfig(cfg *config.Config) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Skip verification for development environments
	if cfg.TLS.SkipVerify {
		tlsCfg.InsecureSkipVerify = true
		return tlsCfg, nil
	}

	// Load custom CA certificate if provided
	if cfg.TLS.CACert != "" {
		caCert, err := os.ReadFile(cfg.TLS.CACert)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate %s: %w", cfg.TLS.CACert, err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate %s", cfg.TLS.CACert)
		}
		tlsCfg.RootCAs = pool
	}

	return tlsCfg, nil
}
