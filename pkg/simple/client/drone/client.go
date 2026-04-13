package drone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envDroneServer = "DRONE_SERVER"
	envDroneToken  = "DRONE_TOKEN"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClientFromEnv() *Client {
	baseURL := strings.TrimSpace(os.Getenv(envDroneServer))
	token := strings.TrimSpace(os.Getenv(envDroneToken))
	if baseURL == "" || token == "" {
		return nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any) (any, error) {
	if c == nil {
		return nil, fmt.Errorf("drone client not configured")
	}

	endpoint := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("drone request failed (%d): %s", resp.StatusCode, msg)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{"status": "ok"}, nil
	}

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return string(data), nil
	}
	return decoded, nil
}

func (c *Client) ListRepos(ctx context.Context) (any, error) {
	return c.request(ctx, http.MethodGet, "/api/user/repos", nil, nil)
}

func (c *Client) ActivateRepo(ctx context.Context, namespace, name string) (any, error) {
	return c.request(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(name), nil, nil)
}

func (c *Client) UpdateRepo(ctx context.Context, namespace, name string, body map[string]any) (any, error) {
	return c.request(ctx, http.MethodPatch, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(name), nil, body)
}

func (c *Client) DeleteRepo(ctx context.Context, namespace, name string) (any, error) {
	return c.request(ctx, http.MethodDelete, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(name), nil, nil)
}

func (c *Client) ListBuilds(ctx context.Context, namespace, repo string) (any, error) {
	return c.request(ctx, http.MethodGet, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(repo)+"/builds", nil, nil)
}

func (c *Client) CreateBuild(ctx context.Context, namespace, repo string, body map[string]any) (any, error) {
	return c.request(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(repo)+"/builds", nil, body)
}

func (c *Client) RestartBuild(ctx context.Context, namespace, repo string, buildNumber int64) (any, error) {
	return c.request(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(repo)+"/builds/"+strconv.FormatInt(buildNumber, 10), nil, nil)
}

func (c *Client) StopBuild(ctx context.Context, namespace, repo string, buildNumber int64) (any, error) {
	return c.request(ctx, http.MethodDelete, "/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(repo)+"/builds/"+strconv.FormatInt(buildNumber, 10), nil, nil)
}
