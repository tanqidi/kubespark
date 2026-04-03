package resources

import (
	"context"
	"io"

	"kubespark/pkg/simple/client/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/remotecommand"
)

// Interface defines the interface for resource operations
type Interface interface {
	ListResources(ctx context.Context, namespace string, resource string, options metav1.ListOptions) (runtime.Object, error)
	GetResource(ctx context.Context, namespace string, resource string, name string, options metav1.GetOptions) (runtime.Object, error)
	ListResourcesByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace string, options metav1.ListOptions) (runtime.Object, error)
	GetClusterInfo(ctx context.Context) (map[string]interface{}, error)
	GetPodLogs(ctx context.Context, namespace string, name string, container string, tailLines *int64) ([]byte, error)
	StreamPodLogs(ctx context.Context, namespace string, name string, container string, tailLines *int64) (io.ReadCloser, error)
	ExecPod(ctx context.Context, namespace string, name string, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer, terminalSizeQueue remotecommand.TerminalSizeQueue) error

	// Generic CRUD based on GroupVersionResource, using dynamic client.
	CreateResourceByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, options metav1.CreateOptions) (runtime.Object, error)
	UpdateResourceByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, options metav1.UpdateOptions) (runtime.Object, error)
	DeleteResourceByGVR(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, options metav1.DeleteOptions) error
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
