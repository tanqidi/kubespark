package resources

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ListResources lists resources by type
func (r *ResourcesOperator) ListResources(ctx context.Context, namespace string, resource string, options metav1.ListOptions) (runtime.Object, error) {
	switch strings.ToLower(resource) {
	case "pods":
		return r.k8sClient.Kubernetes().CoreV1().Pods(namespace).List(ctx, options)
	case "deployments":
		return r.k8sClient.Kubernetes().AppsV1().Deployments(namespace).List(ctx, options)
	case "services":
		return r.k8sClient.Kubernetes().CoreV1().Services(namespace).List(ctx, options)
	case "configmaps":
		return r.k8sClient.Kubernetes().CoreV1().ConfigMaps(namespace).List(ctx, options)
	case "secrets":
		return r.k8sClient.Kubernetes().CoreV1().Secrets(namespace).List(ctx, options)
	case "events":
		return r.k8sClient.Kubernetes().CoreV1().Events(namespace).List(ctx, options)
	case "ingresses":
		return r.k8sClient.Kubernetes().NetworkingV1().Ingresses(namespace).List(ctx, options)
	case "daemonsets":
		return r.k8sClient.Kubernetes().AppsV1().DaemonSets(namespace).List(ctx, options)
	case "statefulsets":
		return r.k8sClient.Kubernetes().AppsV1().StatefulSets(namespace).List(ctx, options)
	case "jobs":
		return r.k8sClient.Kubernetes().BatchV1().Jobs(namespace).List(ctx, options)
	case "cronjobs":
		return r.k8sClient.Kubernetes().BatchV1().CronJobs(namespace).List(ctx, options)
	case "replicasets":
		return r.k8sClient.Kubernetes().AppsV1().ReplicaSets(namespace).List(ctx, options)
	case "persistentvolumeclaims":
		return r.k8sClient.Kubernetes().CoreV1().PersistentVolumeClaims(namespace).List(ctx, options)
	case "namespaces":
		return r.k8sClient.Kubernetes().CoreV1().Namespaces().List(ctx, options)
	case "nodes":
		return r.k8sClient.Kubernetes().CoreV1().Nodes().List(ctx, options)
	case "persistentvolumes":
		return r.k8sClient.Kubernetes().CoreV1().PersistentVolumes().List(ctx, options)
	case "storageclasses":
		return r.k8sClient.Kubernetes().StorageV1().StorageClasses().List(ctx, options)
	default:
		return nil, errors.NewBadRequest(fmt.Sprintf("Unsupported resource type: %s", resource))
	}
}

// GetResource gets a specific resource
func (r *ResourcesOperator) GetResource(ctx context.Context, namespace string, resource string, name string, options metav1.GetOptions) (runtime.Object, error) {
	switch strings.ToLower(resource) {
	case "pods":
		return r.k8sClient.Kubernetes().CoreV1().Pods(namespace).Get(ctx, name, options)
	case "deployments":
		return r.k8sClient.Kubernetes().AppsV1().Deployments(namespace).Get(ctx, name, options)
	case "services":
		return r.k8sClient.Kubernetes().CoreV1().Services(namespace).Get(ctx, name, options)
	case "configmaps":
		return r.k8sClient.Kubernetes().CoreV1().ConfigMaps(namespace).Get(ctx, name, options)
	case "secrets":
		return r.k8sClient.Kubernetes().CoreV1().Secrets(namespace).Get(ctx, name, options)
	case "ingresses":
		return r.k8sClient.Kubernetes().NetworkingV1().Ingresses(namespace).Get(ctx, name, options)
	case "daemonsets":
		return r.k8sClient.Kubernetes().AppsV1().DaemonSets(namespace).Get(ctx, name, options)
	case "statefulsets":
		return r.k8sClient.Kubernetes().AppsV1().StatefulSets(namespace).Get(ctx, name, options)
	case "jobs":
		return r.k8sClient.Kubernetes().BatchV1().Jobs(namespace).Get(ctx, name, options)
	case "cronjobs":
		return r.k8sClient.Kubernetes().BatchV1().CronJobs(namespace).Get(ctx, name, options)
	default:
		return nil, errors.NewBadRequest(fmt.Sprintf("Unsupported resource type: %s", resource))
	}
}

// ListResourcesByGVR lists resources by Group Version Resource
func (r *ResourcesOperator) ListResourcesByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace string, options metav1.ListOptions) (runtime.Object, error) {
	if namespace == "" || namespace == metav1.NamespaceAll {
		return r.k8sClient.Dynamic().Resource(gvr).List(ctx, options)
	}
	return r.k8sClient.Dynamic().Resource(gvr).Namespace(namespace).List(ctx, options)
}

// GetClusterInfo returns cluster information
func (r *ResourcesOperator) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	version, err := r.k8sClient.Kubernetes().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	nodes, err := r.k8sClient.Kubernetes().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaces, err := r.k8sClient.Kubernetes().CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"kubernetesVersion": version,
		"platform":          version.Platform,
		"nodeCount":         len(nodes.Items),
		"namespaceCount":    len(namespaces.Items),
		"serverTime":        metav1.Now(),
	}, nil
}

// GetPodLogs gets logs from a pod
func (r *ResourcesOperator) GetPodLogs(ctx context.Context, namespace string, name string, container string, tailLines *int64) ([]byte, error) {
	opts := &corev1.PodLogOptions{
		Container:  container,
		TailLines:  tailLines,
		Timestamps: true,
	}
	return r.k8sClient.Kubernetes().CoreV1().Pods(namespace).GetLogs(name, opts).DoRaw(ctx)
}

