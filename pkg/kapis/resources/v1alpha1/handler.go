package v1alpha1

import (
	"net/http"
	"strconv"

	"kubespark/pkg/kapis"
	"kubespark/pkg/models/resources"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Handler handles API requests for resources
type Handler struct {
	resourcesOperator resources.Interface
}

// NewHandler creates a new API handler
func NewHandler(resourcesOperator resources.Interface) *Handler {
	return &Handler{
		resourcesOperator: resourcesOperator,
	}
}

// ListPods lists all pods
func (h *Handler) ListPods(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "pods")
}

// ListDeployments lists all deployments
func (h *Handler) ListDeployments(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "deployments")
}

// ListServices lists all services
func (h *Handler) ListServices(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "services")
}

// ListNamespaces lists all namespaces
func (h *Handler) ListNamespaces(req *restful.Request, resp *restful.Response) {
	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResources(req.Request.Context(), "", "namespaces", opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// ListNodes lists all nodes
func (h *Handler) ListNodes(req *restful.Request, resp *restful.Response) {
	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResources(req.Request.Context(), "", "nodes", opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// ListConfigMaps lists all configmaps
func (h *Handler) ListConfigMaps(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "configmaps")
}

// ListSecrets lists all secrets
func (h *Handler) ListSecrets(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "secrets")
}

// ListEvents lists all events
func (h *Handler) ListEvents(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "events")
}

// ListPersistentVolumes lists all persistent volumes
func (h *Handler) ListPersistentVolumes(req *restful.Request, resp *restful.Response) {
	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResources(req.Request.Context(), "", "persistentvolumes", opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// ListPersistentVolumeClaims lists all persistent volume claims
func (h *Handler) ListPersistentVolumeClaims(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "persistentvolumeclaims")
}

// ListStorageClasses lists all storage classes
func (h *Handler) ListStorageClasses(req *restful.Request, resp *restful.Response) {
	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResources(req.Request.Context(), "", "storageclasses", opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// ListIngresses lists all ingresses
func (h *Handler) ListIngresses(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "ingresses")
}

// ListDaemonSets lists all daemonsets
func (h *Handler) ListDaemonSets(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "daemonsets")
}

// ListStatefulSets lists all statefulsets
func (h *Handler) ListStatefulSets(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "statefulsets")
}

// ListJobs lists all jobs
func (h *Handler) ListJobs(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "jobs")
}

// ListCronJobs lists all cronjobs
func (h *Handler) ListCronJobs(req *restful.Request, resp *restful.Response) {
	h.handleListResource(req, resp, metav1.NamespaceAll, "cronjobs")
}

// GetClusterInfo returns cluster information
func (h *Handler) GetClusterInfo(req *restful.Request, resp *restful.Response) {
	result, err := h.resourcesOperator.GetClusterInfo(req.Request.Context())
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// GetResourceByGVR lists resources by Group Version Resource
func (h *Handler) GetResourceByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")

	// In Kubernetes, core resources (like pods, services, etc.) live in the
	// "core" API group, which is represented by an empty string "" in the
	// GroupVersionResource. However, our HTTP route requires a non-empty
	// "{group}" path segment, so the API is designed to accept the literal
	// string "core" and then translate it to "" when constructing the GVR.
	if group == "core" {
		group = ""
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResourcesByGVR(req.Request.Context(), gvr, namespace, opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}

	kapis.WriteSuccess(resp, result)
}

// CreateResourceByGVR creates a resource by Group Version Resource
func (h *Handler) CreateResourceByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")

	// Translate "core" group from the HTTP path into the empty string that
	// Kubernetes expects for core resources.
	if group == "core" {
		group = ""
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	obj := &unstructured.Unstructured{}
	if err := req.ReadEntity(obj); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}

	created, err := h.resourcesOperator.CreateResourceByGVR(req.Request.Context(), gvr, namespace, obj, metav1.CreateOptions{})
	if err != nil {
		h.handleError(resp, err)
		return
	}

	kapis.WriteCreated(resp, created)
}

// UpdateResourceByGVR updates a resource by Group Version Resource
func (h *Handler) UpdateResourceByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")

	// Translate "core" group from the HTTP path into the empty string that
	// Kubernetes expects for core resources.
	if group == "core" {
		group = ""
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	obj := &unstructured.Unstructured{}
	if err := req.ReadEntity(obj); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}

	// Ensure name in metadata matches path parameter to avoid accidental rename
	if obj.GetName() == "" {
		obj.SetName(name)
	}

	updated, err := h.resourcesOperator.UpdateResourceByGVR(req.Request.Context(), gvr, namespace, obj, metav1.UpdateOptions{})
	if err != nil {
		h.handleError(resp, err)
		return
	}

	kapis.WriteSuccess(resp, updated)
}

// DeleteResourceByGVR deletes a resource by Group Version Resource
func (h *Handler) DeleteResourceByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")

	// Translate "core" group from the HTTP path into the empty string that
	// Kubernetes expects for core resources.
	if group == "core" {
		group = ""
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	if err := h.resourcesOperator.DeleteResourceByGVR(req.Request.Context(), gvr, namespace, name, metav1.DeleteOptions{}); err != nil {
		h.handleError(resp, err)
		return
	}

	// For delete we still return a JSON body wrapped in the standard APIResponse,
	// so that clients always receive a consistent structure.
	kapis.WriteSuccess(resp, map[string]string{
		"namespace": namespace,
		"resource":  resource,
		"name":      name,
		"status":    "deleted",
	})
}

// GetNamespaceResources lists resources in a specific namespace
func (h *Handler) GetNamespaceResources(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")
	resource := req.PathParameter("resource")
	opts := h.parseListOptions(req)

	result, err := h.resourcesOperator.ListResources(req.Request.Context(), namespace, resource, opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}

	kapis.WriteSuccess(resp, result)
}

// GetResourceDetail gets a specific resource in a namespace
func (h *Handler) GetResourceDetail(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")
	resource := req.PathParameter("resource")
	name := req.PathParameter("name")

	result, err := h.resourcesOperator.GetResource(req.Request.Context(), namespace, resource, name, metav1.GetOptions{})
	if err != nil {
		h.handleError(resp, err)
		return
	}

	kapis.WriteSuccess(resp, result)
}

// GetPodLogs gets logs from a pod
func (h *Handler) GetPodLogs(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")
	name := req.PathParameter("name")
	container := req.QueryParameter("container")
	tailLinesStr := req.QueryParameter("tailLines")

	var tailLines *int64
	if tailLinesStr != "" {
		if parsed, err := strconv.ParseInt(tailLinesStr, 10, 64); err == nil {
			tailLines = &parsed
		}
	}

	logs, err := h.resourcesOperator.GetPodLogs(req.Request.Context(), namespace, name, container, tailLines)
	if err != nil {
		h.handleError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.Write(logs)
}

// handleListResource is a helper to handle list resource requests
func (h *Handler) handleListResource(req *restful.Request, resp *restful.Response, namespace string, resource string) {
	opts := h.parseListOptions(req)
	result, err := h.resourcesOperator.ListResources(req.Request.Context(), namespace, resource, opts)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	kapis.WriteSuccess(resp, result)
}

// parseListOptions parses query parameters into ListOptions
func (h *Handler) parseListOptions(req *restful.Request) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: req.QueryParameter("labelSelector"),
		FieldSelector: req.QueryParameter("fieldSelector"),
	}
}

// handleError handles errors and converts them to HTTP responses
func (h *Handler) handleError(resp *restful.Response, err error) {
	if statusErr, ok := err.(*errors.StatusError); ok {
		// Preserve the Kubernetes Status as the Data field, and use its Code/message
		kapis.WriteErrorWithCode(resp, int(statusErr.ErrStatus.Code), int(statusErr.ErrStatus.Code), statusErr.Error())
		return
	}
	kapis.WriteErrorWithCode(resp, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
}
