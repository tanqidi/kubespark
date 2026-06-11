package auth

import (
	"context"
	"net/http"
	"os"

	"kubespark/pkg/kapis"
	"kubespark/pkg/simple/client/k8s"

	restful "github.com/emicklei/go-restful/v3"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token"`
}

// HealthzResponse represents the health check response
type HealthzResponse struct {
	Status string `json:"status"`
}

// Handler handles authentication requests
type Handler struct {
	k8sClient k8s.Interface
}

// NewHandler creates a new auth handler
func NewHandler(k8sClient k8s.Interface) *Handler {
	return &Handler{
		k8sClient: k8sClient,
	}
}

// getCredentials returns the configured username and password from environment variables,
// or defaults to "admin" and "123456" if not set
func getCredentials() (username, password string) {
	username = os.Getenv("KUBESPARK_USERNAME")
	if username == "" {
		username = "admin"
	}

	password = os.Getenv("KUBESPARK_PASSWORD")
	if password == "" {
		password = "123456"
	}

	return username, password
}

// Login handles user login
func (h *Handler) Login(req *restful.Request, resp *restful.Response) {
	var loginReq LoginRequest
	if err := req.ReadEntity(&loginReq); err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get credentials from environment variables or use defaults
	expectedUsername, expectedPassword := getCredentials()

	// Validate credentials
	if loginReq.Username != expectedUsername || loginReq.Password != expectedPassword {
		kapis.WriteErrorWithCode(resp, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Generate JWT token
	token, err := GenerateToken(loginReq.Username)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusInternalServerError, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	kapis.WriteSuccess(resp, LoginResponse{
		Token: token,
	})
}

// Healthz handles health check requests
func (h *Handler) Healthz(req *restful.Request, resp *restful.Response) {
	if h.k8sClient == nil {
		kapis.WriteErrorWithCode(resp, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "K8s client not available")
		return
	}

	if err := h.k8sClient.Kubernetes().Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.Background()).Error(); err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "K8s API not healthy")
		return
	}

	kapis.WriteSuccess(resp, HealthzResponse{
		Status: "OK",
	})
}
