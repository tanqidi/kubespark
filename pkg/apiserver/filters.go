package apiserver

import (
	"log"
	"time"

	restful "github.com/emicklei/go-restful/v3"
)

// CORSFilter is a filter that adds CORS headers to all responses
func CORSFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	resp.Header().Set("Access-Control-Allow-Credentials", "true")

	if req.Request.Method == "OPTIONS" {
		return
	}

	chain.ProcessFilter(req, resp)
}

// LoggingFilter is a filter that logs all HTTP requests
func LoggingFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)

	log.Printf("[%s] %s %s - %d (%v)",
		time.Now().Format("2006-01-02 15:04:05"),
		req.Request.Method,
		req.Request.URL.Path,
		resp.StatusCode(),
		duration,
	)
}

