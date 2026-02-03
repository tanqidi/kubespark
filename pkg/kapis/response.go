package kapis

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
)

// APIResponse is the standard response wrapper for all API handlers.
// It is designed to be simple and flexible so that any kind of data
// (including raw Kubernetes objects or custom structs) can be placed
// into the Data field.
type APIResponse struct {
	Code    int         `json:"code"`              // Business status code, 0 means success
	Message string      `json:"message,omitempty"` // Human-readable message
	Data    interface{} `json:"data,omitempty"`    // Arbitrary payload
}

// WriteSuccess writes a successful response with the given data.
// It always uses HTTP 200 OK, with business Code = 0.
func WriteSuccess(resp *restful.Response, data interface{}) {
	_ = resp.WriteHeaderAndEntity(http.StatusOK, APIResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// WriteCreated writes a successful creation response with the given data.
// It uses HTTP 201 Created, with business Code = 0.
func WriteCreated(resp *restful.Response, data interface{}) {
	_ = resp.WriteHeaderAndEntity(http.StatusCreated, APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// WriteErrorWithCode writes an error response using the given HTTP status
// and business code.
func WriteErrorWithCode(resp *restful.Response, httpStatus int, code int, msg string) {
	_ = resp.WriteHeaderAndEntity(httpStatus, APIResponse{
		Code:    code,
		Message: msg,
	})
}
