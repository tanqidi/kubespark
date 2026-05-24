package drone

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"kubespark/pkg/kapis/resources/v1alpha1/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (h *Handler) startPipelineRunDroneSyncer() {
	go func() {
		ticker := time.NewTicker(PipelineRunSyncInterval)
		defer ticker.Stop()
		log.Printf("[pipeline-run-sync] started: interval=%s", PipelineRunSyncInterval)

		for {
			h.syncPendingPipelineRuns()
			<-ticker.C
		}
	}()
}

func (h *Handler) syncPendingPipelineRuns() {
	if h.droneClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), PipelineRunSyncTimeout)
	defer cancel()

	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tanqidi.com",
		Version:  "v1alpha1",
		Resource: "pipelineruns",
	}

	result, err := h.resourcesOperator.ListResourcesByGVR(ctx, pipelineRunGVR, metav1.NamespaceAll, metav1.ListOptions{})
	if err != nil {
		log.Printf("[pipeline-run-sync] scan failed: err=%v", err)
		return
	}

	list, ok := result.(*unstructured.UnstructuredList)
	if !ok {
		log.Printf("[pipeline-run-sync] scan skipped: unexpected list type=%T", result)
		return
	}

	var (
		total        = len(list.Items)
		missingAnno  int
		parseFail    int
		terminal     int
		missingBuild int
		missingRepo  int
		unchanged    int
		updated      int
		failed       int
	)

	for i := range list.Items {
		runObj := list.Items[i].DeepCopy()
		runName := strings.TrimSpace(runObj.GetName())

		dronePayload := strings.TrimSpace(runObj.GetAnnotations()["tanqidi.com/drone"])
		if dronePayload == "" {
			missingAnno++
			continue
		}

		currentBuild, err := parseDroneBuildPayload(dronePayload)
		if err != nil {
			parseFail++
			log.Printf("[pipeline-run-sync] parse annotation failed: run=%s err=%v", runName, err)
			continue
		}

		currentStatus := normalizeBuildStatus(utils.GetMapString(currentBuild, "status"))
		if isTerminalBuildStatus(currentStatus) {
			terminal++
			continue
		}

		buildNumber, ok := utils.GetMapInt64(currentBuild, "number")
		if !ok || buildNumber <= 0 {
			missingBuild++
			log.Printf("[pipeline-run-sync] skip: missing build number run=%s status=%s", runName, currentStatus)
			continue
		}

		repoNamespace, repoName := h.resolvePipelineRunDroneRepo(runObj)
		if repoNamespace == "" || repoName == "" {
			missingRepo++
			log.Printf("[pipeline-run-sync] skip: namespace/repo unresolved run=%s", runName)
			continue
		}

		buildCtx, buildCancel := context.WithTimeout(context.Background(), 10*time.Second)
		latestBuildAny, err := h.droneClient.GetBuild(buildCtx, repoNamespace, repoName, buildNumber)
		buildCancel()
		if err != nil {
			failed++
			log.Printf("[pipeline-run-sync] get build failed: run=%s namespace=%s repo=%s build=%d err=%v", runName, repoNamespace, repoName, buildNumber, err)
			continue
		}

		latestBuild, ok := latestBuildAny.(map[string]any)
		if !ok {
			failed++
			log.Printf("[pipeline-run-sync] unexpected build payload: run=%s type=%T", runName, latestBuildAny)
			continue
		}

		latestStatus := normalizeBuildStatus(utils.GetMapString(latestBuild, "status"))
		if currentStatus == latestStatus && utils.GetMapInt64OrZero(currentBuild, "updated") == utils.GetMapInt64OrZero(latestBuild, "updated") {
			unchanged++
			continue
		}

		if err := h.updatePipelineRunDroneAnnotations(runObj, latestBuild); err != nil {
			failed++
			log.Printf("[pipeline-run-sync] update annotation failed: run=%s namespace=%s repo=%s build=%d err=%v", runName, repoNamespace, repoName, buildNumber, err)
			continue
		}

		updated++
		log.Printf("[pipeline-run-sync] updated: run=%s namespace=%s repo=%s build=%d status=%s->%s", runName, repoNamespace, repoName, buildNumber, currentStatus, latestStatus)
	}

	log.Printf(
		"[pipeline-run-sync] scan done: total=%d missingAnno=%d parseFail=%d terminal=%d missingBuild=%d missingRepo=%d unchanged=%d updated=%d failed=%d",
		total,
		missingAnno,
		parseFail,
		terminal,
		missingBuild,
		missingRepo,
		unchanged,
		updated,
		failed,
	)
}

func parseDroneBuildPayload(raw string) (map[string]any, error) {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func normalizeBuildStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isTerminalBuildStatus(status string) bool {
	switch normalizeBuildStatus(status) {
	case "success", "failure", "error", "killed":
		return true
	default:
		return false
	}
}

func (h *Handler) resolvePipelineRunDroneRepo(runObj *unstructured.Unstructured) (string, string) {
	if runObj == nil {
		return "", ""
	}

	runData, _, _ := unstructured.NestedStringMap(runObj.Object, "spec", "data")
	pipelineName, _, _ := unstructured.NestedString(runObj.Object, "spec", "pipelineRef", "name")
	pipelineName = strings.TrimSpace(pipelineName)

	pipelineData := map[string]string{}
	if pipelineName != "" {
		pipelineData = h.loadPipelineData(pipelineName)
	}
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
	return repoNamespace, repoName
}
