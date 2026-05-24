package drone

import (
	restful "github.com/emicklei/go-restful/v3"
)

// AddToContainer adds drone routes to the container
func (h *Handler) AddToContainer(container *restful.Container) {
	h.addDroneYAMLRoutes(container)
}

func (h *Handler) addDroneYAMLRoutes(container *restful.Container) {
	droneWS := new(restful.WebService)
	droneWS.Path("/kapis/v1alpha1/drone").
		Consumes("*/*").
		Produces("*/*")

	droneWS.Route(droneWS.GET("/yaml").To(h.GetDroneYAML).
		Param(droneWS.QueryParameter("owner", "Repository owner/namespace").DataType("string")).
		Param(droneWS.QueryParameter("repo", "Repository name").DataType("string")).
		Doc("Resolve Drone pipeline YAML from PipelineRun annotation"))

	droneWS.Route(droneWS.POST("/yaml").To(h.GetDroneYAML).
		Doc("Resolve Drone pipeline YAML from PipelineRun annotation"))

	container.Add(droneWS)
}
