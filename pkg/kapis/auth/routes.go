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
	ws.Route(ws.POST("/login").To(h.Login).Doc("User login"))

	container.Add(ws)
}
