package auth

import (
	"net/http"
	"os"

	"kubespark/pkg/kapis"

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

// Handler handles authentication requests
type Handler struct{}

// NewHandler creates a new auth handler
func NewHandler() *Handler {
	return &Handler{}
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

	// Generate JWT token (include password hash in token for additional security)
	// This ensures that even if attackers obtain JWT secret and username,
	// they cannot generate valid tokens without knowing the password
	token, err := GenerateToken(loginReq.Username, loginReq.Password)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusInternalServerError, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	kapis.WriteSuccess(resp, LoginResponse{
		Token: token,
	})
}
