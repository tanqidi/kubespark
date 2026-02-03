package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"kubespark/pkg/apiserver"
	"kubespark/pkg/kapis/resources/v1alpha1"
	"kubespark/pkg/models/resources"
	"kubespark/pkg/simple/client/k8s"
)

var (
	port = flag.String("port", "8080", "Server port")
)

func main() {
	flag.Parse()

	// 1. Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// 2. Create resources operator
	resourcesOperator := resources.NewResourcesOperator(k8sClient)

	// 3. Create API handler
	handler := v1alpha1.NewHandler(resourcesOperator)

	// 4. Create server
	server := apiserver.NewServer(*port)

	// 5. Register routes
	handler.AddToContainer(server.Container())

	// 6. Health check endpoint
	server.Container().Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := k8sClient.Kubernetes().Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.Background()).Error(); err != nil {
			http.Error(w, "K8s API not healthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 7. Welcome page
	server.Container().Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Kubespark - Kubernetes API Explorer</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
					.container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
					h1 { color: #333; margin-top: 0; }
					.api-list { background: #f9f9f9; padding: 20px; border-radius: 5px; margin-top: 20px; }
					.api-item { margin: 10px 0; padding: 8px; background: white; border-left: 3px solid #4CAF50; }
					code { background: #eee; padding: 2px 6px; border-radius: 3px; font-family: 'Courier New', monospace; }
					.method { color: #4CAF50; font-weight: bold; }
					.endpoint { color: #2196F3; }
					.note { margin-top: 20px; padding: 15px; background: #e3f2fd; border-radius: 5px; }
					a { color: #2196F3; text-decoration: none; }
					a:hover { text-decoration: underline; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>Kubespark API Server</h1>
					<p>A simple Kubernetes API proxy built with Go</p>
					
					<div class="api-list">
						<h3>Available APIs:</h3>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/pods</code> - List all pods</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/deployments</code> - List all deployments</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/services</code> - List all services</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/namespaces</code> - List all namespaces</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/nodes</code> - List all nodes</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/configmaps</code> - List all configmaps</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/secrets</code> - List all secrets</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/persistentvolumes</code> - List all PVs</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/persistentvolumeclaims</code> - List all PVCs</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/ingresses</code> - List all ingresses</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/daemonsets</code> - List all daemonsets</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/statefulsets</code> - List all statefulsets</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/jobs</code> - List all jobs</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/cronjobs</code> - List all cronjobs</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/events</code> - List all events</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/storageclasses</code> - List all storage classes</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/cluster-info</code> - Get cluster info</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/resources/{group}/{version}/{resource}</code> - List resources by GVR</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/namespaces/{namespace}/{resource}</code> - List resources in namespace</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/namespaces/{namespace}/{resource}/{name}</code> - Get resource detail</div>
						<div class="api-item"><span class="method">GET</span> <code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/namespaces/{namespace}/pods/{name}/logs</code> - Get pod logs</div>
					</div>
					
					<div class="note">
						<p><strong>Note:</strong> All endpoints support query parameters like:</p>
						<ul>
							<li><code>?namespace=default</code> - Filter by namespace</li>
							<li><code>?labelSelector=app=nginx</code> - Filter by labels</li>
							<li><code>?fieldSelector=status.phase=Running</code> - Filter by fields</li>
						</ul>
						<p><a href="/healthz">Health Check</a></p>
					</div>
				</div>
			</body>
			</html>
		`))
	}))

	// 8. Start server with graceful shutdown
	server.StartWithGracefulShutdown()
}
