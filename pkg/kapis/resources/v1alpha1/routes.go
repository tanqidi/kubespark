package v1alpha1

import (
	restful "github.com/emicklei/go-restful/v3"
)

// AddToContainer adds routes to the container
func (h *Handler) AddToContainer(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/kapis/resources.kubespark.io/v1alpha1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// Core resources
	ws.Route(ws.GET("/pods").To(h.ListPods).Doc("List all pods"))
	ws.Route(ws.GET("/deployments").To(h.ListDeployments).Doc("List all deployments"))
	ws.Route(ws.GET("/services").To(h.ListServices).Doc("List all services"))
	ws.Route(ws.GET("/namespaces").To(h.ListNamespaces).Doc("List all namespaces"))
	ws.Route(ws.GET("/nodes").To(h.ListNodes).Doc("List all nodes"))
	ws.Route(ws.GET("/configmaps").To(h.ListConfigMaps).Doc("List all configmaps"))
	ws.Route(ws.GET("/secrets").To(h.ListSecrets).Doc("List all secrets"))
	ws.Route(ws.GET("/events").To(h.ListEvents).Doc("List all events"))

	// Storage resources
	ws.Route(ws.GET("/persistentvolumes").To(h.ListPersistentVolumes).Doc("List all persistent volumes"))
	ws.Route(ws.GET("/persistentvolumeclaims").To(h.ListPersistentVolumeClaims).Doc("List all persistent volume claims"))
	ws.Route(ws.GET("/storageclasses").To(h.ListStorageClasses).Doc("List all storage classes"))

	// Networking resources
	ws.Route(ws.GET("/ingresses").To(h.ListIngresses).Doc("List all ingresses"))

	// Workload resources
	ws.Route(ws.GET("/daemonsets").To(h.ListDaemonSets).Doc("List all daemonsets"))
	ws.Route(ws.GET("/statefulsets").To(h.ListStatefulSets).Doc("List all statefulsets"))
	ws.Route(ws.GET("/jobs").To(h.ListJobs).Doc("List all jobs"))
	ws.Route(ws.GET("/cronjobs").To(h.ListCronJobs).Doc("List all cronjobs"))

	// Cluster info
	ws.Route(ws.GET("/cluster-info").To(h.GetClusterInfo).Doc("Get cluster information"))

	// Dynamic resource by GVR
	ws.Route(ws.GET("/resources/{group}/{version}/{resource}").To(h.GetResourceByGVR).
		Param(ws.PathParameter("group", "API group").DataType("string")).
		Param(ws.PathParameter("version", "API version").DataType("string")).
		Param(ws.PathParameter("resource", "Resource name").DataType("string")).
		Param(ws.QueryParameter("namespace", "Namespace name").DataType("string")).
		Doc("List resources by GVR (Group Version Resource)"))

	// Namespace-level resources
	ws.Route(ws.GET("/namespaces/{namespace}/{resource}").To(h.GetNamespaceResources).
		Param(ws.PathParameter("namespace", "Namespace name").DataType("string")).
		Param(ws.PathParameter("resource", "Resource type").DataType("string")).
		Param(ws.QueryParameter("labelSelector", "Label selector").DataType("string")).
		Param(ws.QueryParameter("fieldSelector", "Field selector").DataType("string")).
		Doc("List resources in specific namespace"))

	// Get specific resource detail
	ws.Route(ws.GET("/namespaces/{namespace}/{resource}/{name}").To(h.GetResourceDetail).
		Param(ws.PathParameter("namespace", "Namespace name").DataType("string")).
		Param(ws.PathParameter("resource", "Resource type").DataType("string")).
		Param(ws.PathParameter("name", "Resource name").DataType("string")).
		Doc("Get specific resource in namespace"))

	// Pod logs
	ws.Route(ws.GET("/namespaces/{namespace}/pods/{name}/logs").To(h.GetPodLogs).
		Param(ws.PathParameter("namespace", "Namespace name").DataType("string")).
		Param(ws.PathParameter("name", "Pod name").DataType("string")).
		Param(ws.QueryParameter("container", "Container name").DataType("string")).
		Param(ws.QueryParameter("tailLines", "Number of lines to tail").DataType("integer")).
		Doc("Get pod logs").
		Produces("text/plain"))

	container.Add(ws)
}

