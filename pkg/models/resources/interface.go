package resources

import (
	"context"

	"kubespark/pkg/simple/client/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Interface defines the interface for resource operations
type Interface interface {
	ListResources(ctx context.Context, namespace string, resource string, options metav1.ListOptions) (runtime.Object, error)
	GetResource(ctx context.Context, namespace string, resource string, name string, options metav1.GetOptions) (runtime.Object, error)
	ListResourcesByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace string, options metav1.ListOptions) (runtime.Object, error)
	GetClusterInfo(ctx context.Context) (map[string]interface{}, error)
	GetPodLogs(ctx context.Context, namespace string, name string, container string, tailLines *int64) ([]byte, error)
}

// ResourcesOperator provides methods to manipulate resources
type ResourcesOperator struct {
	k8sClient k8s.Interface
}

// NewResourcesOperator creates a resources operator with the given k8s client
func NewResourcesOperator(k8sClient k8s.Interface) Interface {
	return &ResourcesOperator{
		k8sClient: k8sClient,
	}
}

