package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/99designs/httpsignatures-go"
	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ReadBodyAsMap(req *restful.Request) (map[string]any, error) {
	payload := map[string]any{}
	if req.Request.ContentLength == 0 {
		return payload, nil
	}
	if err := req.ReadEntity(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func ReadBodyAsMapLoose(req *restful.Request) (map[string]any, string) {
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

func ReadStringFromMap(m map[string]any, path ...string) string {
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

func ReadInt64FromMap(m map[string]any, path ...string) (int64, bool) {
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

func GetMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func GetMapInt64(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func GetMapInt64OrZero(m map[string]any, key string) int64 {
	value, ok := GetMapInt64(m, key)
	if !ok {
		return 0
	}
	return value
}

func MergeStringMaps(base, override map[string]string) map[string]string {
	merged := map[string]string{}
	for k, v := range base {
		merged[k] = strings.TrimSpace(v)
	}
	for k, v := range override {
		merged[k] = strings.TrimSpace(v)
	}
	return merged
}

func PickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func CountLeadingSpaces(value string) int {
	for i, ch := range value {
		if ch != ' ' {
			return i
		}
	}
	return len(value)
}

func TruncateLogString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "...(truncated)"
}

func AcceptWantsJSON(accept string) bool {
	normalized := strings.ToLower(strings.TrimSpace(accept))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "json")
}

func IsDroneGVR(group, version string) bool {
	return strings.EqualFold(strings.TrimSpace(group), "drone") && strings.EqualFold(strings.TrimSpace(version), "v1")
}

func IsPipelineRunGVR(group, version, resource string) bool {
	return strings.EqualFold(strings.TrimSpace(group), "tanqidi.com") &&
		strings.EqualFold(strings.TrimSpace(version), "v1alpha1") &&
		strings.EqualFold(strings.TrimSpace(resource), "pipelineruns")
}

func ParseNameFromFieldSelector(fieldSelector string) string {
	const prefix = "metadata.name="
	raw := strings.TrimSpace(fieldSelector)
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
}

func TranslateCoreGroup(group string) string {
	if group == "core" {
		return ""
	}
	return group
}

func ParseListOptions(req *restful.Request) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: req.QueryParameter("labelSelector"),
		FieldSelector: req.QueryParameter("fieldSelector"),
	}
}

func ExtractBranchFromPipelineRunDroneYAML(runObj *unstructured.Unstructured) string {
	if runObj == nil {
		return ""
	}
	annotations := runObj.GetAnnotations()
	if annotations == nil {
		return ""
	}
	yamlText := strings.TrimSpace(annotations["tanqidi.com/drone-yaml"])
	if yamlText == "" {
		return ""
	}

	lines := strings.Split(yamlText, "\n")
	triggerIndex := -1
	triggerIndent := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "trigger:") {
			triggerIndex = i
			triggerIndent = CountLeadingSpaces(line)
			break
		}
	}
	if triggerIndex < 0 {
		return ""
	}

	triggerEnd := len(lines)
	for i := triggerIndex + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if CountLeadingSpaces(line) <= triggerIndent {
			triggerEnd = i
			break
		}
	}

	branchIndex := -1
	branchIndent := 0
	for i := triggerIndex + 1; i < triggerEnd; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "branch:") {
			branchIndex = i
			branchIndent = CountLeadingSpaces(line)
			break
		}
	}
	if branchIndex < 0 {
		return ""
	}

	for i := branchIndex + 1; i < triggerEnd; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if CountLeadingSpaces(line) <= branchIndent {
			break
		}
		if strings.HasPrefix(trimmed, "-") {
			candidate := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			candidate = strings.Trim(candidate, "\"'")
			if candidate != "" {
				return candidate
			}
		}
	}

	return ""
}

func ResolveDroneYamlRepo(req *restful.Request, body map[string]any) (string, string) {
	owner := PickFirstNonEmpty(
		req.QueryParameter("owner"),
		req.QueryParameter("namespace"),
		req.QueryParameter("repoNamespace"),
		ReadStringFromMap(body, "repo", "namespace"),
		ReadStringFromMap(body, "repo", "owner"),
		ReadStringFromMap(body, "build", "namespace"),
		ReadStringFromMap(body, "build", "author_login"),
		ReadStringFromMap(body, "build", "sender"),
	)
	repo := PickFirstNonEmpty(
		req.QueryParameter("repo"),
		ReadStringFromMap(body, "repo", "name"),
		ReadStringFromMap(body, "build", "repo"),
	)

	if owner == "" || repo == "" {
		slug := PickFirstNonEmpty(
			req.QueryParameter("slug"),
			ReadStringFromMap(body, "repo", "slug"),
			ReadStringFromMap(body, "build", "repo"),
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

func VerifyDroneYAMLRequest(r *restful.Request, readSecret func() (string, error)) error {
	if r == nil || r.Request == nil {
		return fmt.Errorf("nil request")
	}
	secret, err := readSecret()
	if err != nil || secret == "" {
		if err != nil {
			return fmt.Errorf("read DRONE_YAML_SECRET from secret failed: %w", err)
		}
		return fmt.Errorf("missing DRONE_YAML_SECRET in secret")
	}
	if strings.TrimSpace(r.Request.Header.Get("Signature")) == "" {
		return fmt.Errorf("missing Signature header")
	}

	signature, err := httpsignatures.FromRequest(r.Request)
	if err != nil {
		return fmt.Errorf("read signature failed: %w", err)
	}
	if !signature.IsValid(secret, r.Request) {
		return fmt.Errorf("signature validation failed")
	}
	return nil
}
