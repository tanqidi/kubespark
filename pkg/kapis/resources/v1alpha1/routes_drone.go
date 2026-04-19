package v1alpha1

import (
	restful "github.com/emicklei/go-restful/v3"
)

func (h *Handler) addDroneRoutes(container *restful.Container) {
	droneWS := new(restful.WebService)
	droneWS.Path("/kapis/v1alpha1/drone").
		Consumes("*/*").
		Produces("*/*")

	// Drone YAML extension endpoint (use loose consumes for Drone server compatibility).
	droneWS.Route(droneWS.GET("/yaml").To(h.GetDroneYaml).
		Param(droneWS.QueryParameter("owner", "Repository owner/namespace").DataType("string")).
		Param(droneWS.QueryParameter("repo", "Repository name").DataType("string")).
		Produces("*/*").
		Doc("Resolve Drone pipeline YAML from pipeline run annotation"))
	droneWS.Route(droneWS.POST("/yaml").To(h.GetDroneYaml).
		Produces("*/*").
		Doc("Resolve Drone pipeline YAML from pipeline run annotation"))

	container.Add(droneWS)
}
