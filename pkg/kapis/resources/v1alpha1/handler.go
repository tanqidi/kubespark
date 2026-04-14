package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"kubespark/pkg/kapis"
	"kubespark/pkg/models/resources"
	"kubespark/pkg/simple/client/drone"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/remotecommand"
)

// Handler handles API requests for resources
type Handler struct {
	resourcesOperator resources.Interface
	droneClient       *drone.Client
}

// NewHandler creates a new API handler
func NewHandler(resourcesOperator resources.Interface) *Handler {
	return &Handler{
		resourcesOperator: resourcesOperator,
		droneClient:       drone.NewClientFromEnv(),
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

	if h.isDroneGVR(group, version) {
		h.handleDroneList(req, resp, resource, namespace)
		return
	}

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

	if h.isDroneGVR(group, version) {
		h.handleDroneCreate(req, resp, resource, namespace)
		return
	}

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

	if h.isPipelineRunGVR(group, version, resource) {
		if createdObj, ok := created.(*unstructured.Unstructured); ok && createdObj != nil {
			createdCopy := createdObj.DeepCopy()
			go h.handlePipelineRunCreated(createdCopy)
		} else {
			go h.handlePipelineRunCreated(created)
		}
	}
}

// UpdateResourceByGVR updates a resource by Group Version Resource
func (h *Handler) UpdateResourceByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")

	if h.isDroneGVR(group, version) {
		h.handleDroneUpdate(req, resp, resource, namespace, name)
		return
	}

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

	if h.isDroneGVR(group, version) {
		h.handleDroneDelete(req, resp, resource, namespace, name)
		return
	}

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
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	name := req.PathParameter("name")
	namespace := req.QueryParameter("namespace")
	container := req.QueryParameter("container")
	tailLinesStr := req.QueryParameter("tailLines")
	followStr := req.QueryParameter("follow")

	// Translate "core" from HTTP path to Kubernetes core group.
	if group == "core" {
		group = ""
	}

	// Keep this endpoint aligned with GVR semantics while limiting scope
	// to the pod log subresource for now.
	if group != "" || version != "v1" || !strings.EqualFold(resource, "pods") {
		kapis.WriteErrorWithCode(
			resp,
			http.StatusBadRequest,
			http.StatusBadRequest,
			"logs subresource is currently supported only for core/v1 pods",
		)
		return
	}

	if namespace == "" {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace is required")
		return
	}

	var tailLines *int64
	if tailLinesStr != "" {
		parsed, err := strconv.ParseInt(tailLinesStr, 10, 64)
		if err != nil || parsed < 1 {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "tailLines must be a positive integer")
			return
		}
		tailLines = &parsed
	}

	follow := false
	if followStr != "" {
		parsed, err := strconv.ParseBool(followStr)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "follow must be a boolean")
			return
		}
		follow = parsed
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.Header().Set("Cache-Control", "no-cache")

	if follow {
		stream, err := h.resourcesOperator.StreamPodLogs(req.Request.Context(), namespace, name, container, tailLines)
		if err != nil {
			h.handleError(resp, err)
			return
		}
		defer stream.Close()

		flusher, _ := resp.ResponseWriter.(http.Flusher)
		buf := make([]byte, 32*1024)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if _, wErr := resp.ResponseWriter.Write(buf[:n]); wErr != nil {
					err = wErr
				}
				if flusher != nil {
					flusher.Flush()
				}
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				// Client disconnects are expected when user closes dialog / stops follow.
				if !strings.Contains(strings.ToLower(err.Error()), "broken pipe") &&
					!strings.Contains(strings.ToLower(err.Error()), "connection reset by peer") &&
					!strings.Contains(strings.ToLower(err.Error()), "context canceled") {
					h.handleError(resp, err)
				}
				break
			}
		}
		return
	}

	logs, err := h.resourcesOperator.GetPodLogs(req.Request.Context(), namespace, name, container, tailLines)
	if err != nil {
		h.handleError(resp, err)
		return
	}

	resp.Write(logs)
}

// GetResourceDescribeByGVR gets describe-like details from a resource by GVR.
func (h *Handler) GetResourceDescribeByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	name := req.PathParameter("name")
	namespace := req.QueryParameter("namespace")

	if group == "core" {
		group = ""
	}
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	describeText, err := h.resourcesOperator.DescribeResourceByGVR(req.Request.Context(), gvr, namespace, name)
	if err != nil {
		h.handleError(resp, err)
		return
	}
	resp.Header().Set("Content-Type", "text/plain")
	resp.Header().Set("Cache-Control", "no-cache")
	_, _ = resp.Write([]byte(describeText))
}

type execClientMessage struct {
	Op   string `json:"op"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

type execServerMessage struct {
	Op      string `json:"op"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

type terminalSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func newTerminalSizeQueue() *terminalSizeQueue {
	return &terminalSizeQueue{ch: make(chan remotecommand.TerminalSize, 8)}
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

func (q *terminalSizeQueue) Push(cols, rows uint16) {
	select {
	case q.ch <- remotecommand.TerminalSize{Width: cols, Height: rows}:
	default:
	}
}

func (q *terminalSizeQueue) Close() {
	close(q.ch)
}

// ExecPodByGVR executes into pod container over websocket.
func (h *Handler) ExecPodByGVR(req *restful.Request, resp *restful.Response) {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	resource := req.PathParameter("resource")
	name := req.PathParameter("name")
	namespace := req.QueryParameter("namespace")
	container := req.QueryParameter("container")
	command := req.Request.URL.Query()["command"]
	tty := true
	if ttyRaw := req.QueryParameter("tty"); ttyRaw != "" {
		parsed, err := strconv.ParseBool(ttyRaw)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "tty must be a boolean")
			return
		}
		tty = parsed
	}
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}

	if group == "core" {
		group = ""
	}
	if group != "" || version != "v1" || !strings.EqualFold(resource, "pods") {
		kapis.WriteErrorWithCode(
			resp,
			http.StatusBadRequest,
			http.StatusBadRequest,
			"exec subresource is currently supported only for core/v1 pods",
		)
		return
	}
	if namespace == "" {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace is required")
		return
	}

	ctx, cancel := context.WithCancel(req.Request.Context())
	defer cancel()

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	sizeQueue := newTerminalSizeQueue()
	// Provide an initial terminal size to avoid shells exiting early in TTY mode.
	sizeQueue.Push(80, 24)
	defer func() {
		sizeQueue.Close()
		_ = stdinWriter.Close()
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		_ = stdinReader.Close()
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
	}()

	uprader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}
	conn, err := uprader.Upgrade(resp.ResponseWriter, req.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	send := func(msg execServerMessage) {
		payload, mErr := json.Marshal(msg)
		if mErr != nil {
			return
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}

	pumpOutput := func(op string, reader io.Reader) {
		buf := make([]byte, 32*1024)
		for {
			n, rErr := reader.Read(buf)
			if n > 0 {
				send(execServerMessage{Op: op, Data: string(buf[:n])})
			}
			if rErr != nil {
				return
			}
		}
	}

	go pumpOutput("stdout", stdoutReader)
	if !tty {
		go pumpOutput("stderr", stderrReader)
	}

	execDone := make(chan error, 1)
	go func() {
		err := h.resourcesOperator.ExecPod(
			ctx,
			namespace,
			name,
			container,
			command,
			tty,
			stdinReader,
			stdoutWriter,
			stderrWriter,
			sizeQueue,
		)
		execDone <- err
	}()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, payload, rErr := conn.ReadMessage()
			if rErr != nil {
				cancel()
				return
			}

			var msg execClientMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}

			switch msg.Op {
			case "stdin":
				if msg.Data != "" {
					if _, err := io.WriteString(stdinWriter, msg.Data); err != nil {
						cancel()
						return
					}
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					sizeQueue.Push(msg.Cols, msg.Rows)
				}
			case "close":
				cancel()
				return
			}
		}
	}()

	select {
	case err := <-execDone:
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "context canceled") {
			send(execServerMessage{Op: "error", Message: err.Error()})
		} else {
			send(execServerMessage{Op: "exit"})
		}
	case <-readDone:
		cancel()
	}
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

func (h *Handler) isPipelineRunGVR(group, version, resource string) bool {
	return strings.EqualFold(strings.TrimSpace(group), "tanqidi.com") &&
		strings.EqualFold(strings.TrimSpace(version), "v1alpha1") &&
		strings.EqualFold(strings.TrimSpace(resource), "pipelineruns")
}

func (h *Handler) handlePipelineRunCreated(created any) {
	createdObj, ok := created.(*unstructured.Unstructured)
	if !ok || createdObj == nil {
		log.Printf("[pipeline-run] skip trigger: created object is not unstructured")
		return
	}
	runName := strings.TrimSpace(createdObj.GetName())
	log.Printf("[pipeline-run] created: name=%s", runName)

	if h.droneClient == nil {
		log.Printf("[pipeline-run] skip trigger: drone client not configured")
		return
	}

	pipelineName, _, _ := unstructured.NestedString(createdObj.Object, "spec", "pipelineRef", "name")
	pipelineName = strings.TrimSpace(pipelineName)
	if pipelineName == "" {
		log.Printf("[pipeline-run] skip trigger: missing spec.pipelineRef.name run=%s", runName)
		return
	}

	runData, _, _ := unstructured.NestedStringMap(createdObj.Object, "spec", "data")
	pipelineData := h.loadPipelineData(pipelineName)
	merged := mergeStringMaps(pipelineData, runData)

	repoNamespace := pickFirstNonEmpty(
		merged["droneNamespace"],
		merged["namespace"],
		merged["repoNamespace"],
	)
	repoName := pickFirstNonEmpty(
		merged["droneRepo"],
		merged["repo"],
		pipelineName,
	)

	if repoNamespace == "" || repoName == "" {
		log.Printf("[pipeline-run] skip trigger: namespace/repo unresolved run=%s pipeline=%s", runName, pipelineName)
		return
	}

	buildSpec := map[string]any{}
	for k, v := range merged {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		switch key {
		case "droneNamespace", "namespace", "repoNamespace", "droneRepo", "repo":
			continue
		default:
			buildSpec[key] = v
		}
	}

	if err := h.ensurePipelineRepoActive(repoNamespace, repoName); err != nil {
		log.Printf("[pipeline-run] ensure repo active failed: run=%s pipeline=%s namespace=%s repo=%s err=%v", runName, pipelineName, repoNamespace, repoName, err)
		return
	}

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer buildCancel()

	log.Printf("[pipeline-run] trigger build start: run=%s pipeline=%s namespace=%s repo=%s", runName, pipelineName, repoNamespace, repoName)
	result, err := h.droneClient.CreateBuild(buildCtx, repoNamespace, repoName, buildSpec)
	if err != nil {
		log.Printf("[pipeline-run] trigger build failed: run=%s pipeline=%s namespace=%s repo=%s err=%v", runName, pipelineName, repoNamespace, repoName, err)
		return
	}

	log.Printf("[pipeline-run] trigger build success: run=%s pipeline=%s namespace=%s repo=%s result=%v", runName, pipelineName, repoNamespace, repoName, result)
	if err := h.updatePipelineRunDroneAnnotations(createdObj, result, repoNamespace, repoName); err != nil {
		log.Printf("[pipeline-run] update drone annotation failed: run=%s err=%v", runName, err)
	}
}

func (h *Handler) updatePipelineRunDroneAnnotations(
	runObj *unstructured.Unstructured,
	buildResult any,
	repoNamespace,
	repoName string,
) error {
	if runObj == nil {
		return fmt.Errorf("pipeline run object is nil")
	}

	payloadBytes, err := json.Marshal(buildResult)
	if err != nil {
		return fmt.Errorf("marshal drone result failed: %w", err)
	}

	applyAnnotations := func(target *unstructured.Unstructured) {
		annotations := target.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}

		annotations["tanqidi.com/drone"] = string(payloadBytes)
		annotations["tanqidi.com/drone-namespace"] = strings.TrimSpace(repoNamespace)
		annotations["tanqidi.com/drone-repo"] = strings.TrimSpace(repoName)

		if obj, ok := buildResult.(map[string]any); ok {
			if number, ok := obj["number"]; ok {
				annotations["tanqidi.com/drone-build-number"] = strings.TrimSpace(fmt.Sprintf("%v", number))
			}
			if link, ok := obj["link"].(string); ok {
				annotations["tanqidi.com/drone-build-link"] = strings.TrimSpace(link)
			}
			if status, ok := obj["status"].(string); ok {
				annotations["tanqidi.com/drone-build-status"] = strings.TrimSpace(status)
			}
		}

		target.SetAnnotations(annotations)
	}

	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelineruns",
	}
	namespace := strings.TrimSpace(runObj.GetNamespace())

	applyAnnotations(runObj)
	if _, err := h.resourcesOperator.UpdateResourceByGVR(
		context.Background(),
		pipelineRunGVR,
		namespace,
		runObj,
		metav1.UpdateOptions{},
	); err == nil {
		log.Printf("[pipeline-run] updated drone annotation: run=%s", strings.TrimSpace(runObj.GetName()))
		return nil
	} else if !errors.IsConflict(err) {
		return err
	}

	// Retry once on conflict by reloading latest object.
	name := strings.TrimSpace(runObj.GetName())
	result, err := h.resourcesOperator.ListResourcesByGVR(context.Background(), pipelineRunGVR, namespace, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return fmt.Errorf("reload run on conflict failed: %w", err)
	}
	list, ok := result.(*unstructured.UnstructuredList)
	if !ok || len(list.Items) == 0 {
		return fmt.Errorf("reload run on conflict returned empty")
	}
	latest := list.Items[0].DeepCopy()
	applyAnnotations(latest)
	_, err = h.resourcesOperator.UpdateResourceByGVR(
		context.Background(),
		pipelineRunGVR,
		namespace,
		latest,
		metav1.UpdateOptions{},
	)
	if err != nil {
		return fmt.Errorf("update after conflict failed: %w", err)
	}

	log.Printf("[pipeline-run] updated drone annotation after retry: run=%s", name)
	return nil
}

func (h *Handler) ensurePipelineRepoActive(namespace, repo string) error {
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer checkCancel()

	result, err := h.droneClient.ListRepos(checkCtx)
	if err != nil {
		return fmt.Errorf("list repos failed: %w", err)
	}

	found, active := findDroneRepoStatus(result, namespace, repo)
	if found && active {
		log.Printf("[pipeline-run] repo already active: namespace=%s repo=%s", namespace, repo)
		return nil
	}

	if found && !active {
		log.Printf("[pipeline-run] repo found but inactive, activating: namespace=%s repo=%s", namespace, repo)
	} else {
		log.Printf("[pipeline-run] repo not found in list, try activate/import: namespace=%s repo=%s", namespace, repo)
	}

	activateCtx, activateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer activateCancel()
	if _, err := h.droneClient.ActivateRepo(activateCtx, namespace, repo); err != nil {
		return fmt.Errorf("activate repo failed: %w", err)
	}
	log.Printf("[pipeline-run] repo activate success: namespace=%s repo=%s", namespace, repo)
	return nil
}

func findDroneRepoStatus(payload any, namespace, repo string) (bool, bool) {
	items, ok := payload.([]any)
	if !ok {
		return false, false
	}
	targetNamespace := strings.TrimSpace(namespace)
	targetRepo := strings.TrimSpace(repo)
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ns, _ := obj["namespace"].(string)
		name, _ := obj["name"].(string)
		if strings.TrimSpace(ns) != targetNamespace || strings.TrimSpace(name) != targetRepo {
			continue
		}
		active, _ := obj["active"].(bool)
		return true, active
	}
	return false, false
}

func (h *Handler) loadPipelineData(pipelineName string) map[string]string {
	pipelineGVR := schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelines",
	}

	result, err := h.resourcesOperator.ListResourcesByGVR(context.Background(), pipelineGVR, "", metav1.ListOptions{
		FieldSelector: "metadata.name=" + pipelineName,
	})
	if err != nil {
		log.Printf("[pipeline-run] load pipeline failed: pipeline=%s err=%v", pipelineName, err)
		return map[string]string{}
	}

	list, ok := result.(*unstructured.UnstructuredList)
	if !ok || len(list.Items) == 0 {
		log.Printf("[pipeline-run] pipeline not found: pipeline=%s", pipelineName)
		return map[string]string{}
	}

	data, _, _ := unstructured.NestedStringMap(list.Items[0].Object, "spec", "data")
	return data
}

func mergeStringMaps(base, override map[string]string) map[string]string {
	merged := map[string]string{}
	for k, v := range base {
		merged[k] = strings.TrimSpace(v)
	}
	for k, v := range override {
		merged[k] = strings.TrimSpace(v)
	}
	return merged
}

func pickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *Handler) isDroneGVR(group, version string) bool {
	return strings.EqualFold(strings.TrimSpace(group), "drone") && strings.EqualFold(strings.TrimSpace(version), "v1")
}

func (h *Handler) parseNameFromFieldSelector(fieldSelector string) string {
	const prefix = "metadata.name="
	raw := strings.TrimSpace(fieldSelector)
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
}

func (h *Handler) ensureDroneConfigured(resp *restful.Response) bool {
	if h.droneClient != nil {
		return true
	}
	kapis.WriteErrorWithCode(
		resp,
		http.StatusServiceUnavailable,
		http.StatusServiceUnavailable,
		"drone integration is not configured, please set DRONE_SERVER/DRONE_TOKEN or provide secret drone-secret with DRONE_SERVER_HOST and DRONE_RPC_SECRET",
	)
	return false
}

func readBodyAsMap(req *restful.Request) (map[string]any, error) {
	payload := map[string]any{}
	if req.Request.ContentLength == 0 {
		return payload, nil
	}
	if err := req.ReadEntity(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func readStringFromMap(m map[string]any, path ...string) string {
	var current any = m
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = obj[key]
		if !ok {
			return ""
		}
	}
	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func (h *Handler) resolveDroneRepo(namespace string, req *restful.Request, body map[string]any) (string, string, error) {
	ns := strings.TrimSpace(namespace)
	repo := strings.TrimSpace(req.QueryParameter("repo"))
	if repo == "" {
		repo = h.parseNameFromFieldSelector(req.QueryParameter("fieldSelector"))
	}
	if repo == "" {
		repo = readStringFromMap(body, "metadata", "name")
	}
	if ns == "" {
		ns = readStringFromMap(body, "metadata", "namespace")
	}
	if ns == "" || repo == "" {
		return "", "", fmt.Errorf("query parameter namespace and repo are required")
	}
	return ns, repo, nil
}

func (h *Handler) handleDroneList(req *restful.Request, resp *restful.Response, resource, namespace string) {
	if !h.ensureDroneConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] list: resource=%s namespace=%s query=%s", resource, namespace, req.Request.URL.RawQuery)
	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "repos":
		result, err := h.droneClient.ListRepos(ctx)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, map[string]any{"items": result})
		return
	case "builds":
		ns, repo, err := h.resolveDroneRepo(namespace, req, map[string]any{})
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[drone-gvr] list builds: namespace=%s repo=%s", ns, repo)
		result, err := h.droneClient.ListBuilds(ctx, ns, repo)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, map[string]any{"items": result})
		return
	default:
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+resource)
		return
	}
}

func (h *Handler) handleDroneCreate(req *restful.Request, resp *restful.Response, resource, namespace string) {
	if !h.ensureDroneConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] create: resource=%s namespace=%s query=%s", resource, namespace, req.Request.URL.RawQuery)
	body, err := readBodyAsMap(req)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "repos":
		ns, repo, err := h.resolveDroneRepo(namespace, req, body)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[drone-gvr] activate repo: namespace=%s repo=%s", ns, repo)
		result, err := h.droneClient.ActivateRepo(ctx, ns, repo)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteCreated(resp, result)
		return
	case "builds":
		ns, repo, err := h.resolveDroneRepo(namespace, req, body)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		spec, _ := body["spec"].(map[string]any)
		log.Printf("[drone-gvr] create build: namespace=%s repo=%s", ns, repo)
		result, err := h.droneClient.CreateBuild(ctx, ns, repo, spec)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteCreated(resp, result)
		return
	default:
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+resource)
		return
	}
}

func (h *Handler) handleDroneUpdate(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.ensureDroneConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] update: resource=%s namespace=%s name=%s query=%s", resource, namespace, name, req.Request.URL.RawQuery)
	body, err := readBodyAsMap(req)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "repos":
		ns := strings.TrimSpace(namespace)
		if ns == "" {
			ns = readStringFromMap(body, "metadata", "namespace")
		}
		if ns == "" {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace is required")
			return
		}
		spec, _ := body["spec"].(map[string]any)
		log.Printf("[drone-gvr] update repo: namespace=%s repo=%s", ns, strings.TrimSpace(name))
		result, err := h.droneClient.UpdateRepo(ctx, ns, strings.TrimSpace(name), spec)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, result)
		return
	case "builds":
		ns, repo, err := h.resolveDroneRepo(namespace, req, body)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		number, err := strconv.ParseInt(strings.TrimSpace(name), 10, 64)
		if err != nil || number <= 0 {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "path parameter name must be build number")
			return
		}
		log.Printf("[drone-gvr] restart build: namespace=%s repo=%s build=%d", ns, repo, number)
		result, err := h.droneClient.RestartBuild(ctx, ns, repo, number)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, result)
		return
	default:
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+resource)
		return
	}
}

func (h *Handler) handleDroneDelete(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.ensureDroneConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] delete: resource=%s namespace=%s name=%s query=%s", resource, namespace, name, req.Request.URL.RawQuery)
	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "repos":
		ns := strings.TrimSpace(namespace)
		if ns == "" {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace is required")
			return
		}
		log.Printf("[drone-gvr] delete repo: namespace=%s repo=%s", ns, strings.TrimSpace(name))
		result, err := h.droneClient.DeleteRepo(ctx, ns, strings.TrimSpace(name))
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, result)
		return
	case "builds":
		ns := strings.TrimSpace(namespace)
		repo := strings.TrimSpace(req.QueryParameter("repo"))
		if ns == "" || repo == "" {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace and repo are required")
			return
		}
		number, err := strconv.ParseInt(strings.TrimSpace(name), 10, 64)
		if err != nil || number <= 0 {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "path parameter name must be build number")
			return
		}
		log.Printf("[drone-gvr] stop build: namespace=%s repo=%s build=%d", ns, repo, number)
		result, err := h.droneClient.StopBuild(ctx, ns, repo, number)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, result)
		return
	default:
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+resource)
		return
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
