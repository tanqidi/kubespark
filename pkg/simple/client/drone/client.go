package drone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kubespark/pkg/simple/client/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FixedDroneSecretName      = "kubespark-secret"
	FixedDroneSecretNamespace = "kubespark"

	secretKeyDroneServer     = "DRONE_SERVER"
	secretKeyDroneToken      = "DRONE_TOKEN"
	secretKeyDroneYAMLSecret = "DRONE_YAML_SECRET"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClientFromSecret() *Client {
	baseURL, token := readConfigFromSecret()
	if baseURL == "" || token == "" {
		log.Printf("[drone] client not configured: missing DRONE_SERVER/DRONE_TOKEN in secret %s/%s", FixedDroneSecretNamespace, FixedDroneSecretName)
		return nil
	}
	log.Printf("[drone] client initialized from secret %s/%s, server=%s", FixedDroneSecretNamespace, FixedDroneSecretName, baseURL)

	return newClient(baseURL, token)
}

func newClient(baseURL, token string) *Client {
	return &Client{
		baseURL: normalizeBaseURL(baseURL),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func normalizeBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}
	return strings.TrimRight(value, "/")
}

func readConfigFromSecret() (string, string) {
	secretData, err := readDroneSecretData()
	if err != nil {
		return "", ""
	}

	readSecretValue := func(key string) string {
		value, ok := secretData[key]
		if !ok {
			return ""
		}
		return strings.TrimSpace(string(value))
	}

	baseURL := normalizeBaseURL(readSecretValue(secretKeyDroneServer))
	token := readSecretValue(secretKeyDroneToken)

	return baseURL, token
}

func ReadDroneYAMLSecret() (string, error) {
	value, err := readDroneSecretValueRaw(secretKeyDroneYAMLSecret)
	if err != nil {
		return "", err
	}
	return value, nil
}

func readDroneSecretValueRaw(key string) (string, error) {
	secretData, err := readDroneSecretData()
	if err != nil {
		return "", err
	}
	value, ok := secretData[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", key, FixedDroneSecretNamespace, FixedDroneSecretName)
	}
	return string(value), nil
}

func readDroneSecretData() (map[string][]byte, error) {
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	secret, err := k8sClient.Kubernetes().CoreV1().Secrets(FixedDroneSecretNamespace).Get(context.Background(), FixedDroneSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret %s/%s has no data", FixedDroneSecretNamespace, FixedDroneSecretName)
	}
	return secret.Data, nil
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any) (any, error) {
	if c == nil {
		return nil, fmt.Errorf("drone client not configured")
	}

	endpoint := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	start := time.Now()
	log.Printf("[drone] request start: method=%s url=%s", method, endpoint)

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
		log.Printf("[drone] request error: method=%s url=%s err=%v cost=%v", method, endpoint, err, time.Since(start))
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
		log.Printf("[drone] request failed: method=%s url=%s status=%d cost=%v msg=%s", method, endpoint, resp.StatusCode, time.Since(start), msg)
		return nil, fmt.Errorf("drone request failed (%d): %s", resp.StatusCode, msg)
	}
	log.Printf("[drone] request done: method=%s url=%s status=%d cost=%v", method, endpoint, resp.StatusCode, time.Since(start))

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

func (c *Client) GetBuild(ctx context.Context, namespace, repo string, buildNumber int64) (any, error) {
	return c.request(
		ctx,
		http.MethodGet,
		"/api/repos/"+url.PathEscape(namespace)+"/"+url.PathEscape(repo)+"/builds/"+strconv.FormatInt(buildNumber, 10),
		nil,
		nil,
	)
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
