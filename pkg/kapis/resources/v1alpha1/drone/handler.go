package drone

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"kubespark/pkg/kapis"
	"kubespark/pkg/kapis/resources/v1alpha1/utils"
	"kubespark/pkg/models/resources"
	droneclient "kubespark/pkg/simple/client/drone"

	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Handler handles drone-related requests
type Handler struct {
	resourcesOperator resources.Interface
	droneClient       *droneclient.Client
}

// NewHandler creates a new drone handler
func NewHandler(resourcesOperator resources.Interface, droneClient *droneclient.Client) *Handler {
	h := &Handler{
		resourcesOperator: resourcesOperator,
		droneClient:       droneClient,
	}
	if droneClient != nil {
		h.startPipelineRunDroneSyncer()
	}
	return h
}

var (
	SchemaGroupVersionResourcePipelineRuns = schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelineruns",
	}
	PipelineRunSyncInterval = 8 * time.Second
	PipelineRunSyncTimeout  = 20 * time.Second
	DroneYamlAnnotationKey  = "tanqidi.com/drone-yaml"
)

// GetDroneYAML handles drone yaml resolution
func (h *Handler) GetDroneYAML(req *restful.Request, resp *restful.Response) {
	accept := req.HeaderParameter("Accept")
	owner := req.QueryParameter("owner")
	repo := req.QueryParameter("repo")
	log.Printf("[drone-yaml] request: method=%s accept=%s owner=%s repo=%s", req.Request.Method, accept, owner, repo)

	if err := h.verifyDroneYAMLRequest(req.Request); err != nil {
		log.Printf("[drone-yaml] request verification failed: err=%v", err)
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	pipelineRunName, pipelineRunNamespace, matchedBy, err := h.resolvePipelineRun(req.Request)
	if err != nil {
		log.Printf("[drone-yaml] resolve pipeline run failed: err=%v", err)
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	runObj, err := h.getPipelineRun(req.Request.Context(), pipelineRunNamespace, pipelineRunName)
	if err != nil {
		log.Printf("[drone-yaml] get pipeline run failed: err=%v", err)
		kapis.WriteErrorWithCode(resp, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}

	yamlText, ok := runObj.GetAnnotations()[DroneYamlAnnotationKey]
	if !ok || strings.TrimSpace(yamlText) == "" {
		log.Printf("[drone-yaml] yaml annotation missing: run=%s", pipelineRunName)
		kapis.WriteErrorWithCode(resp, http.StatusNotFound, http.StatusNotFound, "yaml not found")
		return
	}

	log.Printf("[drone-yaml] hit: owner=%s repo=%s run=%s by=%s bytes=%d", owner, repo, pipelineRunName, matchedBy, len(yamlText))
	if utils.AcceptWantsJSON(accept) {
		log.Printf("[drone-yaml] response: format=json")
		kapis.WriteSuccess(resp, map[string]string{"data": yamlText})
		return
	}
	log.Printf("[drone-yaml] response: format=text")
	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(yamlText))
}

func (h *Handler) verifyDroneYAMLRequest(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("nil request")
	}
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			_ = r.Body.Close()
			log.Printf("[drone-yaml] POST body: %s", strings.TrimSpace(string(body)))
		}
	}
	return nil
}

func (h *Handler) resolvePipelineRun(r *http.Request) (name, namespace, matchedBy string, err error) {
	if r == nil {
		return "", "", "", fmt.Errorf("nil request")
	}
	q := r.URL.Query()

	if v := strings.TrimSpace(q.Get("commit_message")); v != "" {
		if name, ns, ok := h.resolvePipelineRunFromCommitMessage(v); ok {
			return name, ns, "commit_message", nil
		}
	}
	if v := strings.TrimSpace(q.Get("pipeline_run")); v != "" {
		if name, ns, ok := h.parsePipelineRunRef(v); ok {
			return name, ns, "pipeline_run", nil
		}
	}
	if v := strings.TrimSpace(q.Get("pipelinerun")); v != "" {
		if name, ns, ok := h.parsePipelineRunRef(v); ok {
			return name, ns, "pipelinerun", nil
		}
	}
	return "", "", "", fmt.Errorf("could not resolve pipeline run from request")
}

func (h *Handler) resolvePipelineRunFromCommitMessage(msg string) (name, namespace string, ok bool) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", "", false
	}

	searchTags := []string{
		"[pipeline-run=",
		"[pipelinerun=",
		"pipeline-run=",
		"pipelinerun=",
	}
	for _, tag := range searchTags {
		idx := strings.Index(msg, tag)
		if idx < 0 {
			continue
		}
		afterTag := msg[idx+len(tag):]
		endIdx := strings.IndexAny(afterTag, " ]\n\r\t")
		if endIdx > 0 {
			afterTag = afterTag[:endIdx]
		}
		afterTag = strings.TrimSpace(afterTag)
		if name, ns, parsedOk := h.parsePipelineRunRef(afterTag); parsedOk {
			return name, ns, true
		}
	}
	return "", "", false
}

func (h *Handler) parsePipelineRunRef(ref string) (name, namespace string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", false
	}

	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		namespace = strings.TrimSpace(parts[0])
		name = strings.TrimSpace(parts[1])
		if namespace != "" && name != "" {
			return name, namespace, true
		}
	}

	name = strings.TrimSpace(ref)
	if name != "" {
		return name, "", true
	}
	return "", "", false
}

func (h *Handler) getPipelineRun(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	pipelineRunGVR := SchemaGroupVersionResourcePipelineRuns
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	result, err := h.resourcesOperator.ListResourcesByGVR(ctx, pipelineRunGVR, namespace, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return nil, err
	}

	list, ok := result.(*unstructured.UnstructuredList)
	if !ok || len(list.Items) == 0 {
		return nil, fmt.Errorf("pipelinerun %s/%s not found", namespace, name)
	}

	return &list.Items[0], nil
}

func (h *Handler) EnsureConfigured(resp *restful.Response) bool {
	if h.droneClient == nil {
		kapis.WriteErrorWithCode(resp, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "drone client not configured")
		return false
	}
	return true
}

func (h *Handler) ResolveDroneRepo(namespace string, req *restful.Request, body map[string]any) (string, string, error) {
	// 首先从 query 或者 body 或者其他地方取，这里简化一下
	repoNamespace := utils.PickFirstNonEmpty(
		strings.TrimSpace(req.QueryParameter("droneNamespace")),
		strings.TrimSpace(namespace),
	)
	repoName := utils.PickFirstNonEmpty(
		strings.TrimSpace(req.QueryParameter("droneRepo")),
		strings.TrimSpace(req.QueryParameter("repo")),
	)
	if repoNamespace == "" || repoName == "" {
		return "", "", fmt.Errorf("could not resolve drone repo namespace/name")
	}
	return repoNamespace, repoName, nil
}
