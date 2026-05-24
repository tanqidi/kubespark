package v1alpha1

import (
	"kubespark/pkg/kapis/resources/v1alpha1/drone"
	"kubespark/pkg/kapis/resources/v1alpha1/k8s"
	"kubespark/pkg/models/resources"
	droneclient "kubespark/pkg/simple/client/drone"

	restful "github.com/emicklei/go-restful/v3"
)

// Handler handles API requests for resources
type Handler struct {
	k8sHandler   *k8s.Handler
	droneHandler *drone.Handler
}

// NewHandler creates a new API handler
func NewHandler(resourcesOperator resources.Interface) *Handler {
	droneClient := droneclient.NewClientFromSecret()

	k8sHandler := k8s.NewHandler(resourcesOperator)
	var droneHandler *drone.Handler
	if droneClient != nil {
		droneHandler = drone.NewHandler(resourcesOperator, droneClient)
	}

	handler := &Handler{
		k8sHandler:   k8sHandler,
		droneHandler: droneHandler,
	}

	// Set up the onResourceCreated callback to trigger drone pipeline run
	if droneHandler != nil {
		k8sHandler.SetOnResourceCreated(func(created any) {
			droneHandler.HandlePipelineRunCreated(created)
		})
	}

	return handler
}

// ListPods lists all pods
func (h *Handler) ListPods(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListPods(req, resp)
}

// ListDeployments lists all deployments
func (h *Handler) ListDeployments(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListDeployments(req, resp)
}

// ListServices lists all services
func (h *Handler) ListServices(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListServices(req, resp)
}

// ListNamespaces lists all namespaces
func (h *Handler) ListNamespaces(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListNamespaces(req, resp)
}

// ListNodes lists all nodes
func (h *Handler) ListNodes(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListNodes(req, resp)
}

// ListConfigMaps lists all configmaps
func (h *Handler) ListConfigMaps(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListConfigMaps(req, resp)
}

// ListSecrets lists all secrets
func (h *Handler) ListSecrets(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListSecrets(req, resp)
}

// ListEvents lists all events
func (h *Handler) ListEvents(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListEvents(req, resp)
}

// ListPersistentVolumes lists all persistent volumes
func (h *Handler) ListPersistentVolumes(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListPersistentVolumes(req, resp)
}

// ListPersistentVolumeClaims lists all persistent volume claims
func (h *Handler) ListPersistentVolumeClaims(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListPersistentVolumeClaims(req, resp)
}

// ListStorageClasses lists all storage classes
func (h *Handler) ListStorageClasses(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListStorageClasses(req, resp)
}

// ListIngresses lists all ingresses
func (h *Handler) ListIngresses(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListIngresses(req, resp)
}

// ListDaemonSets lists all daemonsets
func (h *Handler) ListDaemonSets(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListDaemonSets(req, resp)
}

// ListStatefulSets lists all statefulsets
func (h *Handler) ListStatefulSets(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListStatefulSets(req, resp)
}

// ListJobs lists all jobs
func (h *Handler) ListJobs(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListJobs(req, resp)
}

// ListCronJobs lists all cronjobs
func (h *Handler) ListCronJobs(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ListCronJobs(req, resp)
}

// GetClusterInfo returns cluster information
func (h *Handler) GetClusterInfo(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.GetClusterInfo(req, resp)
}

// GetResourceByGVR lists resources by Group Version Resource
func (h *Handler) GetResourceByGVR(req *restful.Request, resp *restful.Response) {
	// First check if it's a drone GVR
	if h.droneHandler != nil && h.droneHandler.IsDroneGVR(req) {
		h.droneHandler.HandleGVRList(req, resp)
		return
	}
	h.k8sHandler.GetResourceByGVR(req, resp)
}

// CreateResourceByGVR creates a resource by Group Version Resource
func (h *Handler) CreateResourceByGVR(req *restful.Request, resp *restful.Response) {
	// First check if it's a drone GVR
	if h.droneHandler != nil && h.droneHandler.IsDroneGVR(req) {
		h.droneHandler.HandleGVRCreate(req, resp)
		return
	}
	h.k8sHandler.CreateResourceByGVR(req, resp)
}

// UpdateResourceByGVR updates a resource by Group Version Resource
func (h *Handler) UpdateResourceByGVR(req *restful.Request, resp *restful.Response) {
	// First check if it's a drone GVR
	if h.droneHandler != nil && h.droneHandler.IsDroneGVR(req) {
		h.droneHandler.HandleGVRUpdate(req, resp)
		return
	}
	h.k8sHandler.UpdateResourceByGVR(req, resp)
}

// DeleteResourceByGVR deletes a resource by Group Version Resource
func (h *Handler) DeleteResourceByGVR(req *restful.Request, resp *restful.Response) {
	// First check if it's a drone GVR
	if h.droneHandler != nil && h.droneHandler.IsDroneGVR(req) {
		h.droneHandler.HandleGVRDelete(req, resp)
		return
	}
	h.k8sHandler.DeleteResourceByGVR(req, resp)
}

// GetNamespaceResources lists resources in a specific namespace
func (h *Handler) GetNamespaceResources(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.GetNamespaceResources(req, resp)
}

// GetResourceDetail gets a specific resource in a namespace
func (h *Handler) GetResourceDetail(req *restful.Request, resp *restful.Response) {
	// First check if it's a drone GVR
	if h.droneHandler != nil && h.droneHandler.IsDroneGVR(req) {
		h.droneHandler.HandleGVRGet(req, resp)
		return
	}
	h.k8sHandler.GetResourceDetail(req, resp)
}

// GetPodLogs gets logs from a pod
func (h *Handler) GetPodLogs(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.GetPodLogs(req, resp)
}

// GetResourceDescribeByGVR gets describe-like details from a resource by GVR
func (h *Handler) GetResourceDescribeByGVR(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.GetResourceDescribeByGVR(req, resp)
}

// ExecPodByGVR executes into pod container over websocket
func (h *Handler) ExecPodByGVR(req *restful.Request, resp *restful.Response) {
	h.k8sHandler.ExecPodByGVR(req, resp)
}

// IsDroneGVR checks if the request is for a drone GVR
func (h *Handler) IsDroneGVR(req *restful.Request) bool {
	if h.droneHandler == nil {
		return false
	}
	return h.droneHandler.IsDroneGVR(req)
}
