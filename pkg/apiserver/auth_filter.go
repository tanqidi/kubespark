package apiserver

import (
	"net/http"
	"strings"

	"kubespark/pkg/kapis"
	"kubespark/pkg/kapis/auth"

	restful "github.com/emicklei/go-restful/v3"
)

// AuthFilter is a filter that validates JWT tokens for all requests except login
func AuthFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// Allow OPTIONS requests (CORS preflight)
	if req.Request.Method == "OPTIONS" {
		chain.ProcessFilter(req, resp)
		return
	}

	// Allow health check and swagger endpoints
	path := req.Request.URL.Path
	if path == "/swagger" || path == "/" || path == "/apidocs.json" {
		chain.ProcessFilter(req, resp)
		return
	}

	// Allow login endpoint
	if strings.HasPrefix(path, "/kapis/auth/v1/login") {
		chain.ProcessFilter(req, resp)
		return
	}

	// Allow Drone YAML extension endpoint (Drone server cannot provide Kubespark JWT).
	if strings.HasPrefix(path, "/kapis/v1alpha1/drone/yaml") {
		chain.ProcessFilter(req, resp)
		return
	}

	// Extract token from Authorization header
	authHeader := req.HeaderParameter("Authorization")
	if authHeader == "" {
		// Query token fallback for websocket exec endpoint (browser WebSocket cannot set custom Authorization header).
		if strings.HasSuffix(path, "/exec") {
			token := req.QueryParameter("token")
			if token != "" {
				claims, err := auth.ValidateToken(token)
				if err != nil {
					kapis.WriteErrorWithCode(resp, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid or expired token")
					return
				}
				req.SetAttribute("username", claims.Username)
				chain.ProcessFilter(req, resp)
				return
			}
		}
		kapis.WriteErrorWithCode(resp, http.StatusUnauthorized, http.StatusUnauthorized, "Missing authorization token")
		return
	}

	// Parse Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		kapis.WriteErrorWithCode(resp, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid authorization header format")
		return
	}

	tokenString := parts[1]

	// Validate token
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Store username in request attribute for potential use in handlers
	req.SetAttribute("username", claims.Username)

	// Continue to next filter/handler
	chain.ProcessFilter(req, resp)
}
