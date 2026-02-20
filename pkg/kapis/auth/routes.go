package auth

import (
	restful "github.com/emicklei/go-restful/v3"
)

// AddToContainer adds auth routes to the container
func (h *Handler) AddToContainer(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/kapis/auth.kubespark.io/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// Login endpoint
	ws.Route(ws.POST("/login").To(h.Login).
		Doc("User login - Get JWT Token for API authentication").
		Notes("Use the returned token in Authorization header as 'Bearer {token}' for subsequent API calls"))

	// Health check endpoint
	ws.Route(ws.GET("/healthz").To(h.Healthz).
		Doc("Health check - Check server and Kubernetes API health").
		Notes("Returns the health status of the server and Kubernetes API connection"))

	container.Add(ws)
}
