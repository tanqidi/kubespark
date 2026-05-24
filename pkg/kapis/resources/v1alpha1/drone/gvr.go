package drone

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kubespark/pkg/kapis"
	"kubespark/pkg/kapis/resources/v1alpha1/utils"
	"kubespark/pkg/simple/client/drone"

	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (h *Handler) HandleDroneList(req *restful.Request, resp *restful.Response, resource, namespace string) {
	if !h.EnsureConfigured(resp) {
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
		ns, repo, err := h.ResolveDroneRepo(namespace, req, map[string]any{})
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
		ns, repo, err := h.ResolveDroneRepo(namespace, req, map[string]any{})
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

func (h *Handler) HandleDroneGet(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.EnsureConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] get: resource=%s namespace=%s name=%s query=%s", resource, namespace, name, req.Request.URL.RawQuery)
	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "builds":
		ns, repo, err := h.ResolveDroneRepo(namespace, req, map[string]any{})
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		buildNumber, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "invalid build number")
			return
		}
		log.Printf("[drone-gvr] get build: namespace=%s repo=%s build=%d", ns, repo, buildNumber)
		result, err := h.droneClient.GetBuild(ctx, ns, repo, buildNumber)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadGateway, http.StatusBadGateway, err.Error())
			return
		}
		kapis.WriteSuccess(resp, result)
		return
	case "logs":
		ns, repo, err := h.ResolveDroneRepo(namespace, req, map[string]any{})
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
			return
		}
		buildNumber, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "invalid build number")
			return
		}

		stage := int64(-1)
		step := int64(-1)
		if stageStr := req.QueryParameter("stage"); stageStr != "" {
			if s, err := strconv.ParseInt(stageStr, 10, 64); err == nil {
				stage = s
			}
		}
		if stepStr := req.QueryParameter("step"); stepStr != "" {
			if s, err := strconv.ParseInt(stepStr, 10, 64); err == nil {
				step = s
			}
		}

		baseURL := h.droneClient.GetServerBaseURL()
		logsURL := drone.BuildBuildLogsURL(baseURL, ns, repo, buildNumber, stage, step)
		logsStreamURL := drone.BuildBuildLogsStreamURL(baseURL, ns, repo, buildNumber, stage, step)

		log.Printf("[drone-gvr] logs URLs: namespace=%s repo=%s build=%d stage=%d step=%d logsURL=%s streamURL=%s", ns, repo, buildNumber, stage, step, logsURL, logsStreamURL)

		kapis.WriteSuccess(resp, map[string]string{
			"logsURL":       logsURL,
			"logsStreamURL": logsStreamURL,
		})
		return
	default:
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+resource)
		return
	}
}

func (h *Handler) HandleDroneCreate(req *restful.Request, resp *restful.Response, resource, namespace string) {
	if !h.EnsureConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] create: resource=%s namespace=%s query=%s", resource, namespace, req.Request.URL.RawQuery)
	body, err := utils.ReadBodyAsMap(req)
	if err != nil {
		kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	ctx := req.Request.Context()
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "repos":
		ns, repo, err := h.ResolveDroneRepo(namespace, req, body)
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
		ns, repo, err := h.ResolveDroneRepo(namespace, req, body)
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

func (h *Handler) HandleDroneUpdate(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.EnsureConfigured(resp) {
		return
	}

	log.Printf("[drone-gvr] update: resource=%s namespace=%s name=%s query=%s", resource, namespace, name, req.Request.URL.RawQuery)
	kapis.WriteErrorWithCode(resp, http.StatusBadRequest, http.StatusBadRequest, "unsupported drone resource: "+strings.ToLower(strings.TrimSpace(resource)))
}

func (h *Handler) HandleDroneDelete(req *restful.Request, resp *restful.Response, resource, namespace, name string) {
	if !h.EnsureConfigured(resp) {
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

func (h *Handler) HandlePipelineRunCreated(created any) {
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
	merged := utils.MergeStringMaps(pipelineData, runData)

	repoNamespace := utils.PickFirstNonEmpty(
		merged["droneNamespace"],
		merged["namespace"],
		merged["repoNamespace"],
	)
	repoName := utils.PickFirstNonEmpty(
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
	if h.readBuildSpecString(buildSpec, "branch") == "" {
		if fallbackBranch := utils.ExtractBranchFromPipelineRunDroneYAML(createdObj); fallbackBranch != "" {
			buildSpec["branch"] = fallbackBranch
		}
	}
	h.normalizeDroneBuildSpec(buildSpec)
	if err := h.ensurePipelineRepoActive(repoNamespace, repoName); err != nil {
		log.Printf("[pipeline-run] ensure repo active failed: run=%s pipeline=%s namespace=%s repo=%s err=%v", runName, pipelineName, repoNamespace, repoName, err)
		return
	}

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer buildCancel()

	if payload, marshalErr := json.Marshal(buildSpec); marshalErr == nil {
		log.Printf("[pipeline-run] trigger build payload: run=%s pipeline=%s namespace=%s repo=%s payload=%s", runName, pipelineName, repoNamespace, repoName, string(payload))
	} else {
		log.Printf("[pipeline-run] trigger build payload marshal failed: run=%s pipeline=%s namespace=%s repo=%s err=%v payload=%v", runName, pipelineName, repoNamespace, repoName, marshalErr, buildSpec)
	}
	log.Printf("[pipeline-run] trigger build start: run=%s pipeline=%s namespace=%s repo=%s", runName, pipelineName, repoNamespace, repoName)
	result, err := h.droneClient.CreateBuild(buildCtx, repoNamespace, repoName, buildSpec)
	if err != nil {
		log.Printf("[pipeline-run] trigger build failed: run=%s pipeline=%s namespace=%s repo=%s err=%v", runName, pipelineName, repoNamespace, repoName, err)
		return
	}

	log.Printf("[pipeline-run] trigger build success: run=%s pipeline=%s namespace=%s repo=%s result=%v", runName, pipelineName, repoNamespace, repoName, result)
	if err := h.updatePipelineRunDroneAnnotations(createdObj, result); err != nil {
		log.Printf("[pipeline-run] update drone annotation failed: run=%s err=%v", runName, err)
	}
}

func (h *Handler) readBuildSpecString(spec map[string]any, key string) string {
	if spec == nil {
		return ""
	}
	value, ok := spec[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func (h *Handler) normalizeDroneBuildSpec(spec map[string]any) {
	if spec == nil {
		return
	}

	branch := h.readBuildSpecString(spec, "branch")
	if branch == "" {
		return
	}

	if h.readBuildSpecString(spec, "target") == "" {
		spec["target"] = branch
	}
	if h.readBuildSpecString(spec, "source") == "" {
		spec["source"] = branch
	}
	if h.readBuildSpecString(spec, "ref") == "" {
		spec["ref"] = "refs/heads/" + branch
	}
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

func (h *Handler) updatePipelineRunDroneAnnotations(
	runObj *unstructured.Unstructured,
	buildResult any,
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
	} else if !strings.Contains(err.Error(), "conflict") {
		return err
	}

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
