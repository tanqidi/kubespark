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
	"time"

	"kubespark/pkg/kapis"
	"kubespark/pkg/simple/client/drone"

	"github.com/99designs/httpsignatures-go"
	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetDroneYaml resolves drone pipeline yaml from PipelineRun annotations.
// Protocol:
// - 200 + raw yaml text when found.
// - 204 when not found, so Drone can fallback to repository .drone.yml.
func (h *Handler) GetDroneYaml(req *restful.Request, resp *restful.Response) {
	if err := verifyDroneYAMLRequest(req.Request); err != nil {
		log.Printf(
			"[drone-yaml] unauthorized: err=%v method=%s uri=%s host=%s date=%s digest=%s signature=%s",
			err,
			req.Request.Method,
			req.Request.RequestURI,
			req.Request.Host,
			req.Request.Header.Get("Date"),
			req.Request.Header.Get("Digest"),
			truncateLogString(req.Request.Header.Get("Signature"), 280),
		)
		resp.AddHeader("Content-Type", "text/plain; charset=utf-8")
		resp.WriteHeader(http.StatusUnauthorized)
		_, _ = resp.Write([]byte("invalid drone yaml signature"))
		return
	}

	body, rawBody := readBodyAsMapLoose(req)
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

	owner, repo := resolveDroneYamlRepo(req, body)
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
	if acceptWantsJSON(accept) {
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

func verifyDroneYAMLRequest(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("nil request")
	}
	// Keep the secret bytes exactly as provided in kubespark/kubespark-secret
	// to match Drone's signer behavior.
	secret, err := drone.ReadDroneYAMLSecret()
	if err != nil || secret == "" {
		if err != nil {
			return fmt.Errorf("read DRONE_YAML_SECRET from secret failed: %w", err)
		}
		return fmt.Errorf("missing DRONE_YAML_SECRET in secret")
	}
	if strings.TrimSpace(r.Header.Get("Signature")) == "" {
		return fmt.Errorf("missing Signature header")
	}

	signature, err := httpsignatures.FromRequest(r)
	if err != nil {
		return fmt.Errorf("read signature failed: %w", err)
	}
	if !signature.IsValid(secret, r) {
		return fmt.Errorf("signature validation failed")
	}
	return nil
}

func readBodyAsMapLoose(req *restful.Request) (map[string]any, string) {
	payload := map[string]any{}
	if req == nil || req.Request == nil || req.Request.Body == nil {
		return payload, ""
	}

	data, err := io.ReadAll(req.Request.Body)
	if err != nil {
		return payload, ""
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return payload, ""
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return map[string]any{}, raw
	}
	return payload, raw
}

func truncateLogString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "...(truncated)"
}

func acceptWantsJSON(accept string) bool {
	normalized := strings.ToLower(strings.TrimSpace(accept))
	if normalized == "" {
		return false
	}
	// Drone sends vendor media type like: application/vnd.drone.config.v1+json
	return strings.Contains(normalized, "json")
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
		"drone integration is not configured, please configure DRONE_SERVER/DRONE_TOKEN in secret kubespark/kubespark-secret",
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

func readInt64FromMap(m map[string]any, path ...string) (int64, bool) {
	var current any = m
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return 0, false
		}
		current, ok = obj[key]
		if !ok {
			return 0, false
		}
	}
	switch value := current.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		return int64(value), true
	case json.Number:
		n, err := value.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func resolveDroneYamlRepo(req *restful.Request, body map[string]any) (string, string) {
	owner := pickFirstNonEmpty(
		req.QueryParameter("owner"),
		req.QueryParameter("namespace"),
		req.QueryParameter("repoNamespace"),
		readStringFromMap(body, "repo", "namespace"),
		readStringFromMap(body, "repo", "owner"),
		readStringFromMap(body, "build", "namespace"),
		readStringFromMap(body, "build", "author_login"),
		readStringFromMap(body, "build", "sender"),
		readStringFromMap(body, "build", "trigger"),
	)
	repo := pickFirstNonEmpty(
		req.QueryParameter("repo"),
		readStringFromMap(body, "repo", "name"),
		readStringFromMap(body, "build", "repo"),
	)

	if owner == "" || repo == "" {
		slug := pickFirstNonEmpty(
			req.QueryParameter("slug"),
			readStringFromMap(body, "repo", "slug"),
			readStringFromMap(body, "build", "repo"),
		)
		if strings.Contains(slug, "/") {
			parts := strings.SplitN(slug, "/", 2)
			if owner == "" {
				owner = strings.TrimSpace(parts[0])
			}
			if repo == "" {
				repo = strings.TrimSpace(parts[1])
			}
		}
	}

	return strings.TrimSpace(owner), strings.TrimSpace(repo)
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
	targetRepoID, hasRepoID := readInt64FromMap(body, "build", "repo_id")
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
		droneYaml := strings.TrimSpace(annotations[droneYamlAnnotationKey])
		if droneYaml == "" {
			continue
		}
		// Only consider runs that haven't been bound to a Drone build yet.
		if strings.TrimSpace(annotations["tanqidi.com/drone"]) != "" {
			continue
		}

		data, _, _ := unstructured.NestedStringMap(item.Object, "spec", "data")
		runOwner := pickFirstNonEmpty(data["namespace"], data["droneNamespace"], data["repoNamespace"])
		runRepo := pickFirstNonEmpty(data["repo"], data["droneRepo"])
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
	case "secrets":
		ns, repo, err := h.resolveDroneRepo(namespace, req, map[string]any{})
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[drone-gvr] list secrets: namespace=%s repo=%s", ns, repo)
		result, err := h.droneClient.ListSecrets(ctx, ns, repo)
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
	case "reposync":
		log.Printf("[drone-gvr] sync repos")
		result, err := h.droneClient.SyncRepos(ctx)
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
	case "secrets":
		ns := strings.TrimSpace(namespace)
		repo := strings.TrimSpace(req.QueryParameter("repo"))
		if ns == "" || repo == "" {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace and repo are required")
			return
		}
		log.Printf("[drone-gvr] create secret: namespace=%s repo=%s", ns, repo)
		result, err := h.droneClient.CreateSecret(ctx, ns, repo, body)
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
	kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+strings.ToLower(strings.TrimSpace(resource)))
}

func (h *Handler) handleDroneDelete(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.ensureDroneConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] delete: resource=%s namespace=%s name=%s query=%s", resource, namespace, name, req.Request.URL.RawQuery)
	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "secrets":
		ns := strings.TrimSpace(namespace)
		repo := strings.TrimSpace(req.QueryParameter("repo"))
		secretName := strings.TrimSpace(name)
		if ns == "" || repo == "" || secretName == "" {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "query parameter namespace/repo and path name are required")
			return
		}
		log.Printf("[drone-gvr] delete secret: namespace=%s repo=%s secret=%s", ns, repo, secretName)
		result, err := h.droneClient.DeleteSecret(ctx, ns, repo, secretName)
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
