package drone

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"kubespark/pkg/kapis"
	"kubespark/pkg/kapis/resources/v1alpha1/utils"
	"kubespark/pkg/models/resources"
	droneclient "kubespark/pkg/simple/client/drone"

	"github.com/99designs/httpsignatures-go"
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

// GetDroneYAML resolves drone pipeline yaml from PipelineRun annotations.
// Protocol:
// - 200 + raw yaml text when found.
// - 204 when not found, so Drone can fallback to repository .drone.yml.
func (h *Handler) GetDroneYAML(req *restful.Request, resp *restful.Response) {
	if err := h.verifyDroneYAMLRequest(req); err != nil {
		log.Printf(
			"[drone-yaml] unauthorized: err=%v method=%s uri=%s host=%s date=%s digest=%s signature=%s",
			err,
			req.Request.Method,
			req.Request.RequestURI,
			req.Request.Host,
			req.Request.Header.Get("Date"),
			req.Request.Header.Get("Digest"),
			utils.TruncateLogString(req.Request.Header.Get("Signature"), 280),
		)
		resp.AddHeader("Content-Type", "text/plain; charset=utf-8")
		resp.WriteHeader(http.StatusUnauthorized)
		_, _ = resp.Write([]byte("invalid drone yaml signature"))
		return
	}

	body, rawBody := utils.ReadBodyAsMapLoose(req)
	contentType := strings.TrimSpace(req.Request.Header.Get("Content-Type"))
	accept := strings.TrimSpace(req.Request.Header.Get("Accept"))
	log.Printf(
		"[drone-yaml] request: method=%s contentType=%s accept=%s query=%s body=%s",
		req.Request.Method,
		contentType,
		accept,
		req.Request.URL.RawQuery,
		rawBody,
	)

	owner, repo := utils.ResolveDroneYamlRepo(req, body)
	if owner == "" || repo == "" {
		log.Printf("[drone-yaml] skip: unresolved owner/repo query=%s body=%s", req.Request.URL.RawQuery, rawBody)
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, cancel := context.WithTimeout(req.Request.Context(), 10*time.Second)
	defer cancel()

	yamlText, matchedRun, matchedBy, err := h.readDroneYamlFromPipelineRunAnnotations(ctx, owner, repo, body)
	if err != nil {
		log.Printf("[drone-yaml] miss: owner=%s repo=%s err=%v", owner, repo, err)
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	log.Printf("[drone-yaml] hit: owner=%s repo=%s run=%s by=%s bytes=%d", owner, repo, matchedRun, matchedBy, len(yamlText))
	if utils.AcceptWantsJSON(accept) {
		log.Printf("[drone-yaml] response: format=json")
		resp.AddHeader("Content-Type", "application/json; charset=utf-8")
		resp.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(resp).Encode(map[string]string{
			"data": yamlText,
		})
		return
	}
	log.Printf("[drone-yaml] response: format=text")
	resp.AddHeader("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(yamlText))
}

func (h *Handler) verifyDroneYAMLRequest(req *restful.Request) error {
	if req == nil || req.Request == nil {
		return fmt.Errorf("nil request")
	}
	// Keep the secret bytes exactly as provided in kubespark/kubespark-secret
	// to match Drone's signer behavior.
	secret, err := droneclient.ReadDroneYAMLSecret()
	if err != nil || secret == "" {
		if err != nil {
			return fmt.Errorf("read DRONE_YAML_SECRET from secret failed: %w", err)
		}
		return fmt.Errorf("missing DRONE_YAML_SECRET in secret")
	}
	if strings.TrimSpace(req.Request.Header.Get("Signature")) == "" {
		return fmt.Errorf("missing Signature header")
	}

	signature, err := httpsignatures.FromRequest(req.Request)
	if err != nil {
		return fmt.Errorf("read signature failed: %w", err)
	}
	if !signature.IsValid(secret, req.Request) {
		return fmt.Errorf("signature validation failed")
	}
	return nil
}

func (h *Handler) readDroneYamlFromPipelineRunAnnotations(
	ctx context.Context,
	owner,
	repo string,
	body map[string]any,
) (string, string, string, error) {
	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelineruns",
	}

	result, err := h.resourcesOperator.ListResourcesByGVR(ctx, pipelineRunGVR, metav1.NamespaceAll, metav1.ListOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("list pipelineruns failed: %w", err)
	}

	list, ok := result.(*unstructured.UnstructuredList)
	if !ok {
		return "", "", "", fmt.Errorf("unexpected pipelineruns list type: %T", result)
	}

	targetOwner := strings.TrimSpace(owner)
	targetRepo := strings.TrimSpace(repo)
	if targetOwner == "" || targetRepo == "" {
		return "", "", "", fmt.Errorf("missing owner/repo")
	}
	targetRepoID, hasRepoID := utils.ReadInt64FromMap(body, "build", "repo_id")
	if !hasRepoID || targetRepoID <= 0 {
		return "", "", "", fmt.Errorf("missing build.repo_id")
	}

	var (
		bestRunName string
		bestYaml    string
		bestTime    time.Time
	)

	for i := range list.Items {
		item := &list.Items[i]
		annotations := item.GetAnnotations()
		if annotations == nil {
			continue
		}
		droneYaml := strings.TrimSpace(annotations[DroneYamlAnnotationKey])
		if droneYaml == "" {
			continue
		}
		// Only consider runs that haven't been bound to a Drone build yet.
		if strings.TrimSpace(annotations["tanqidi.com/drone"]) != "" {
			continue
		}

		data, _, _ := unstructured.NestedStringMap(item.Object, "spec", "data")
		runOwner := utils.PickFirstNonEmpty(data["namespace"], data["droneNamespace"], data["repoNamespace"])
		runRepo := utils.PickFirstNonEmpty(data["repo"], data["droneRepo"])
		if runOwner != targetOwner || runRepo != targetRepo {
			continue
		}

		createdAt := item.GetCreationTimestamp().Time
		if bestRunName == "" || createdAt.After(bestTime) {
			bestRunName = strings.TrimSpace(item.GetName())
			bestYaml = droneYaml
			bestTime = createdAt
		}
	}

	if bestRunName == "" || bestYaml == "" {
		return "", "", "", fmt.Errorf(
			"no matched unbound pipelinerun annotation for owner=%s repo=%s repo_id=%d",
			targetOwner,
			targetRepo,
			targetRepoID,
		)
	}
	return bestYaml, bestRunName, "owner/repo+repo_id+latest-unbound", nil
}

func (h *Handler) ResolveDroneRepo(namespace string, req *restful.Request, body map[string]any) (string, string, error) {
	ns := strings.TrimSpace(namespace)
	repo := strings.TrimSpace(req.QueryParameter("repo"))

	// 先直接获取 repo 信息
	if repo == "" {
		repo = utils.ParseNameFromFieldSelector(req.QueryParameter("fieldSelector"))
	}
	if repo == "" {
		repo = utils.ReadStringFromMap(body, "metadata", "name")
	}
	if ns == "" {
		ns = utils.ReadStringFromMap(body, "metadata", "namespace")
	}

	// 如果已经有 namespace 和 repo 了，直接返回
	if ns != "" && repo != "" {
		// 检查是否是 pipeline，如果是 pipeline 则需要查找对应的 drone namespace/repo
		pipelineName := strings.TrimSpace(req.QueryParameter("fieldSelector"))
		if pipelineName != "" && strings.Contains(pipelineName, "metadata.name=") {
			pipelineName = strings.TrimPrefix(pipelineName, "metadata.name=")
			pipelineName = strings.TrimSpace(pipelineName)
			if pipelineName != "" {
				droneNs, droneRepo := h.loadDroneInfoFromPipeline(ns, pipelineName)
				if droneNs != "" && droneRepo != "" {
					return droneNs, droneRepo, nil
				}
			}
		}
		return ns, repo, nil
	}

	if ns == "" || repo == "" {
		return "", "", fmt.Errorf("query parameter namespace and repo are required")
	}
	return ns, repo, nil
}

func (h *Handler) loadDroneInfoFromPipeline(namespace, pipelineName string) (string, string) {
	pipelineGVR := schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelines",
	}

	result, err := h.resourcesOperator.ListResourcesByGVR(context.Background(), pipelineGVR, namespace, metav1.ListOptions{
		FieldSelector: "metadata.name=" + pipelineName,
	})
	if err != nil {
		log.Printf("[drone] load pipeline failed: pipeline=%s err=%v", pipelineName, err)
		return "", ""
	}

	list, ok := result.(*unstructured.UnstructuredList)
	if !ok || len(list.Items) == 0 {
		log.Printf("[drone] pipeline not found: pipeline=%s", pipelineName)
		return "", ""
	}

	data, _, _ := unstructured.NestedStringMap(list.Items[0].Object, "spec", "data")

	droneNs := utils.PickFirstNonEmpty(
		data["droneNamespace"],
		data["namespace"],
		data["repoNamespace"],
		namespace,
	)
	droneRepo := utils.PickFirstNonEmpty(
		data["droneRepo"],
		data["repo"],
		pipelineName,
	)

	return droneNs, droneRepo
}

func (h *Handler) EnsureConfigured(resp *restful.Response) bool {
	if h.droneClient == nil {
		kapis.WriteErrorWithCode(resp, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "drone client not configured")
		return false
	}
	return true
}

// IsDroneGVR checks if the request is for a drone GVR
func (h *Handler) IsDroneGVR(req *restful.Request) bool {
	group := req.PathParameter("group")
	version := req.PathParameter("version")
	return utils.IsDroneGVR(group, version)
}

// HandleGVRList handles GVR list requests for drone resources
func (h *Handler) HandleGVRList(req *restful.Request, resp *restful.Response) {
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	h.HandleDroneList(req, resp, resource, namespace)
}

// HandleGVRCreate handles GVR create requests for drone resources
func (h *Handler) HandleGVRCreate(req *restful.Request, resp *restful.Response) {
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	h.HandleDroneCreate(req, resp, resource, namespace)
}

// HandleGVRUpdate handles GVR update requests for drone resources
func (h *Handler) HandleGVRUpdate(req *restful.Request, resp *restful.Response) {
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")
	h.HandleDroneUpdate(req, resp, resource, namespace, name)
}

// HandleGVRDelete handles GVR delete requests for drone resources
func (h *Handler) HandleGVRDelete(req *restful.Request, resp *restful.Response) {
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")
	h.HandleDroneDelete(req, resp, resource, namespace, name)
}

// HandleGVRGet handles GVR get requests for drone resources
func (h *Handler) HandleGVRGet(req *restful.Request, resp *restful.Response) {
	resource := req.PathParameter("resource")
	namespace := req.QueryParameter("namespace")
	name := req.PathParameter("name")
	h.HandleDroneGet(req, resp, resource, namespace, name)
}
