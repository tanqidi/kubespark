package k8s

import (
	restful "github.com/emicklei/go-restful/v3"
)

// AddToContainer adds kubernetes routes to the container
func (h *Handler) AddToContainer(container *restful.Container) {
	h.addKubernetesRoutes(container)
}

func (h *Handler) addKubernetesRoutes(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/kapis/v1alpha1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/cluster-info").To(h.GetClusterInfo).
		Doc("Get cluster information"))

	ws.Route(ws.GET("/resources/{group}/{version}/{resource}").To(h.GetResourceByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("List resources by GVR (Group Version Resource)"))

	ws.Route(ws.POST("/resources/{group}/{version}/{resource}").To(h.CreateResourceByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("Create resource by GVR (Group Version Resource)"))

	ws.Route(ws.PUT("/resources/{group}/{version}/{resource}/{name}").To(h.UpdateResourceByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("Update resource by GVR (Group Version Resource)"))

	ws.Route(ws.DELETE("/resources/{group}/{version}/{resource}/{name}").To(h.DeleteResourceByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("Delete resource by GVR (Group Version Resource)"))

	ws.Route(ws.GET("/resources/{group}/{version}/{resource}/{name}").To(h.GetResourceDetail).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("Get resource by GVR (Group Version Resource)"))

	ws.Route(ws.GET("/resources/{group}/{version}/{resource}/{name}/log").To(h.GetPodLogs).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Param(ws.QueryParameter("container", "Container name").DataType("string")).
		Param(ws.QueryParameter("tailLines", "Number of log lines from the end").DataType("integer")).
		Param(ws.QueryParameter("follow", "Whether to stream logs continuously").DataType("boolean")).
		Produces("text/plain").
		Doc("Get logs by GVR (currently supports core/v1 pods only)"))

	ws.Route(ws.GET("/resources/{group}/{version}/{resource}/{name}/describe").To(h.GetResourceDescribeByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Produces("text/plain").
		Doc("Get describe details by GVR"))

	ws.Route(ws.GET("/resources/{group}/{version}/{resource}/{name}/exec").To(h.ExecPodByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Param(ws.QueryParameter("container", "Container name").DataType("string")).
		Param(ws.QueryParameter("command", "Command argument, can be repeated").DataType("string")).
		Param(ws.QueryParameter("tty", "Enable TTY").DataType("boolean")).
		Doc("Exec by GVR over Websocket (currently supports core/v1 pods only)"))

	container.Add(ws)
}
