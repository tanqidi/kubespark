package v1alpha1

import (
	restful "github.com/emicklei/go-restful/v3"
)

// GetDroneYaml resolves drone pipeline yaml from PipelineRun annotations.
func (h *Handler) GetDroneYaml(req *restful.Request, resp *restful.Response) {
	if h.droneHandler == nil {
		return
	}
	h.droneHandler.GetDroneYAML(req, resp)
}
