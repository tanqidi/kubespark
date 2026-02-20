package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/go-openapi/spec"

	"kubespark/pkg/apiserver"
	"kubespark/pkg/kapis/auth"
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

	// 4. Create auth handler
	authHandler := auth.NewHandler()

	// 5. Create server
	server := apiserver.NewServer(*port)

	// 6. Register auth routes (must be registered before resource routes)
	authHandler.AddToContainer(server.Container())

	// 7. Register resource routes
	handler.AddToContainer(server.Container())

	// 8. Health check endpoint
	server.Container().Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := k8sClient.Kubernetes().Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.Background()).Error(); err != nil {
			http.Error(w, "K8s API not healthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 9. Register OpenAPI (Swagger) documentation endpoint.
	// This will scan all registered WebServices and expose the spec at /apidocs.json
	openAPIConfig := restfulspec.Config{
		WebServices: server.Container().RegisteredWebServices(),
		APIPath:     "/apidocs.json",
		PostBuildSwaggerObjectHandler: func(swo *spec.Swagger) {
			// Add security definitions for JWT Bearer Token
			if swo.SecurityDefinitions == nil {
				swo.SecurityDefinitions = make(map[string]*spec.SecurityScheme)
			}
			swo.SecurityDefinitions["Bearer"] = &spec.SecurityScheme{
				SecuritySchemeProps: spec.SecuritySchemeProps{
					Type:        "apiKey",
					Name:        "Authorization",
					In:          "header",
					Description: "JWT Bearer Token authentication. Enter 'Bearer {token}' (include 'Bearer' prefix). Get token from POST /kapis/auth.kubespark.io/v1/login",
				},
			}

			// Add security requirements to all paths except login and health check
			if swo.Paths != nil && swo.Paths.Paths != nil {
				for path, pathItem := range swo.Paths.Paths {
					// Skip login endpoint and health check
					if path == "/kapis/auth.kubespark.io/v1/login" || path == "/healthz" {
						continue
					}

					// Add security requirement to all operations in this path
					securityReq := map[string][]string{"Bearer": {}}

					if pathItem.Get != nil {
						pathItem.Get.Security = []map[string][]string{securityReq}
					}
					if pathItem.Post != nil {
						pathItem.Post.Security = []map[string][]string{securityReq}
					}
					if pathItem.Put != nil {
						pathItem.Put.Security = []map[string][]string{securityReq}
					}
					if pathItem.Delete != nil {
						pathItem.Delete.Security = []map[string][]string{securityReq}
					}
					if pathItem.Patch != nil {
						pathItem.Patch.Security = []map[string][]string{securityReq}
					}
				}
			}
		},
	}
	server.Container().Add(restfulspec.NewOpenAPIService(openAPIConfig))

	// 10. Swagger UI page (like Java-style Swagger page)
	server.Container().Handle("/swagger", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<title>Kubespark Swagger UI</title>
				<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
				<style>
					html, body { margin: 0; padding: 0; height: 100%; }
					body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif; }
					.topbar { display: none; }
				</style>
			</head>
			<body>
				<div id="swagger-ui"></div>
				<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
				<script>
					window.onload = function() {
						window.ui = SwaggerUIBundle({
							url: '/apidocs.json',
							dom_id: '#swagger-ui',
							presets: [
								SwaggerUIBundle.presets.apis,
								SwaggerUIBundle.SwaggerUIStandalonePreset
							],
							layout: "BaseLayout",
							onComplete: function() {
								// Add authentication notice
								var authNotice = document.createElement('div');
								authNotice.style.cssText = 'background: #fff3cd; border: 1px solid #ffc107; border-radius: 4px; padding: 15px; margin: 20px; color: #856404;';
								authNotice.innerHTML = '<strong>🔐 认证说明：</strong>大部分 API 接口需要 JWT Token 认证。请先调用 <code>POST /kapis/auth.kubespark.io/v1/login</code> 接口获取 Token，然后在右上角点击 "Authorize" 按钮，输入 <code>Bearer {your-token}</code>（包含 "Bearer" 前缀）。';
								var swaggerContainer = document.querySelector('#swagger-ui');
								if (swaggerContainer) {
									swaggerContainer.insertBefore(authNotice, swaggerContainer.firstChild);
								}
							}
						});
					};
				</script>
			</body>
			</html>
		`))
	}))

	// 11. Welcome page
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
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/apidocs.json</code>
							- OpenAPI (Swagger) specification for all registered endpoints.
						</div>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/kapis/resources.kubespark.io/v1alpha1/*</code>
							- Kubernetes resource proxy (pods, deployments, services, etc.).
						</div>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/healthz</code>
							- Server and Kubernetes API health check.
						</div>
						<p style="margin-top:16px;">
							You can open the OpenAPI spec in any Swagger UI instance by pointing it to
							<code class="endpoint">/apidocs.json</code>.
						</p>
					</div>
					
					<div class="note">
						<p><strong>🔐 认证说明：</strong>大部分 API 接口需要 JWT Token 认证。请先调用 <code>/kapis/auth.kubespark.io/v1/login</code> 接口获取 Token，然后在请求头中添加 <code>Authorization: Bearer {token}</code>。</p>
						<p><strong>Note:</strong> All endpoints support query parameters like:</p>
						<ul>
							<li><code>?namespace=default</code> - Filter by namespace</li>
							<li><code>?labelSelector=app=nginx</code> - Filter by labels</li>
							<li><code>?fieldSelector=status.phase=Running</code> - Filter by fields</li>
						</ul>
						<p><a href="/healthz">Health Check</a></p>
						<p><a href="/swagger">Swagger UI</a></p>
					</div>
				</div>
			</body>
			</html>
		`))
	}))

	// 12. Start server with graceful shutdown
	server.StartWithGracefulShutdown()
}
