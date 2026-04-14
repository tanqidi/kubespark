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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kubespark/pkg/simple/client/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	envDroneServer          = "DRONE_SERVER"
	envDroneToken           = "DRONE_TOKEN"
	envDroneSecretNamespace = "DRONE_SECRET_NAMESPACE"
	envDroneSecretName      = "DRONE_SECRET_NAME"
	envPodNamespace         = "POD_NAMESPACE"

	defaultDroneSecretName      = "drone-secret"
	defaultDroneSecretNamespace = "kubespark"

	secretKeyDroneServer      = "DRONE_SERVER"
	secretKeyDroneServerHost  = "DRONE_SERVER_HOST"
	secretKeyDroneServerProto = "DRONE_SERVER_PROTO"
	secretKeyDroneToken       = "DRONE_TOKEN"
	secretKeyDroneRPCSecret   = "DRONE_RPC_SECRET"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClientFromEnv() *Client {
	baseURL := normalizeBaseURL(strings.TrimSpace(os.Getenv(envDroneServer)))
	token := strings.TrimSpace(os.Getenv(envDroneToken))
	if baseURL == "" || token == "" {
		baseURL, token = readConfigFromSecret()
		if baseURL == "" || token == "" {
			log.Printf("[drone] client not configured: missing DRONE_SERVER/DRONE_TOKEN and secret fallback")
			return nil
		}
		log.Printf("[drone] client initialized from secret, server=%s", baseURL)
	} else {
		log.Printf("[drone] client initialized from env, server=%s", baseURL)
	}

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
	secretName := strings.TrimSpace(os.Getenv(envDroneSecretName))
	if secretName == "" {
		secretName = defaultDroneSecretName
	}

	secretNamespace := resolveSecretNamespace()
	if secretNamespace == "" {
		secretNamespace = defaultDroneSecretNamespace
	}

	k8sClient, err := k8s.NewClient()
	if err != nil {
		return "", ""
	}

	secret, err := k8sClient.Kubernetes().CoreV1().Secrets(secretNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", ""
	}

	readSecretValue := func(key string) string {
		if secret == nil || secret.Data == nil {
			return ""
		}
		value, ok := secret.Data[key]
		if !ok {
			return ""
		}
		return strings.TrimSpace(string(value))
	}

	baseURL := normalizeBaseURL(readSecretValue(secretKeyDroneServer))
	if baseURL == "" {
		host := readSecretValue(secretKeyDroneServerHost)
		proto := readSecretValue(secretKeyDroneServerProto)
		if proto == "" {
			proto = "http"
		}
		if host != "" {
			baseURL = normalizeBaseURL(proto + "://" + host)
		}
	}

	token := readSecretValue(secretKeyDroneToken)
	if token == "" {
		token = readSecretValue(secretKeyDroneRPCSecret)
	}

	return baseURL, token
}

func resolveSecretNamespace() string {
	if value := strings.TrimSpace(os.Getenv(envDroneSecretNamespace)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(envPodNamespace)); value != "" {
		return value
	}

	namespaceFile := filepath.Join(string(filepath.Separator), "var", "run", "secrets", "kubernetes.io", "serviceaccount", "namespace")
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
