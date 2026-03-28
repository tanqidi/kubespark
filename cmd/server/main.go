package main

import (
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
	authHandler := auth.NewHandler(k8sClient)

	// 5. Create server
	server := apiserver.NewServer(*port)

	// 6. Register auth routes (must be registered before resource routes)
	authHandler.AddToContainer(server.Container())

	// 7. Register resource routes
	handler.AddToContainer(server.Container())

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
					Description: "JWT Bearer Token authentication. Enter 'Bearer {token}' (include 'Bearer' prefix). Get token from POST /kapis/auth/v1/login",
				},
			}

			// Add security requirements to all paths except login
			if swo.Paths != nil && swo.Paths.Paths != nil {
				for path, pathItem := range swo.Paths.Paths {
					// Skip login endpoint (health check also requires auth)
					if path == "/kapis/auth/v1/login" {
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
								authNotice.innerHTML = '<strong>🔐 认证说明：</strong>大部分 API 接口需要 JWT Token 认证。请先调用 <code>POST /kapis/auth/v1/login</code> 接口获取 Token，然后在右上角点击 "Authorize" 按钮，输入 <code>Bearer {your-token}</code>（包含 "Bearer" 前缀）。';
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<meta charset="UTF-8">
				<title>Kubespark API</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
					.container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
					h1 { color: #333; margin-top: 0; }
					h2 { margin: 24px 0 12px; color: #222; }
					p { line-height: 1.6; }
					.api-list { background: #f9f9f9; padding: 20px; border-radius: 5px; margin-top: 16px; }
					.api-item { margin: 10px 0; padding: 8px; background: white; border-left: 3px solid #4CAF50; }
					code { background: #eee; padding: 2px 6px; border-radius: 3px; font-family: 'Courier New', monospace; word-break: break-all; }
					.method { color: #4CAF50; font-weight: bold; }
					.endpoint { color: #2196F3; }
					.note { margin-top: 20px; padding: 15px; background: #e3f2fd; border-radius: 5px; }
					.warn { margin-top: 20px; padding: 15px; background: #fff8e1; border-radius: 5px; border: 1px solid #ffe082; }
					a { color: #2196F3; text-decoration: none; }
					a:hover { text-decoration: underline; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>Kubespark API Server</h1>
					<p>统一资源接口（GVR）入口：<code>/kapis/v1alpha1/resources/{group}/{version}/{resource}</code></p>

					<div class="api-list">
						<h2>文档与鉴权</h2>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/apidocs.json</code>
							- OpenAPI 规范
						</div>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/swagger</code>
							- Swagger UI
						</div>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/kapis/auth/v1/healthz</code>
							- 服务健康检查
						</div>
					</div>

					<div class="api-list">
						<h2>通用接口用法</h2>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/kapis/v1alpha1/resources/{group}/{version}/{resource}?namespace=bb</code>
							- 列表查询（<code>namespace</code> 仅对命名空间级资源生效）
						</div>
						<div class="api-item">
							<span class="method">GET</span>
							<code class="endpoint">/kapis/v1alpha1/resources/{group}/{version}/{resource}/{name}?namespace=bb</code>
							- 单条详情
						</div>
						<div class="api-item">
							<span class="method">POST</span>
							<code class="endpoint">/kapis/v1alpha1/resources/{group}/{version}/{resource}?namespace=bb</code>
							- 创建资源（Body 为 Kubernetes 资源 JSON）
						</div>
						<div class="api-item">
							<span class="method">PUT</span>
							<code class="endpoint">/kapis/v1alpha1/resources/{group}/{version}/{resource}/{name}?namespace=bb</code>
							- 更新资源
						</div>
						<div class="api-item">
							<span class="method">DELETE</span>
							<code class="endpoint">/kapis/v1alpha1/resources/{group}/{version}/{resource}/{name}?namespace=bb</code>
							- 删除资源
						</div>
					</div>

					<div class="note">
						<p><strong>认证说明：</strong>大部分 API 接口需要 JWT Token。先调用 <code>/kapis/auth/v1/login</code> 获取 Token，请求头带上 <code>Authorization: Bearer {token}</code>。</p>
						<p><strong>常用筛选参数：</strong></p>
						<ul>
							<li><code>?namespace=bb</code> - 命名空间过滤</li>
							<li><code>?labelSelector=app=nginx</code> - 标签过滤</li>
							<li><code>?fieldSelector=metadata.name=nginx-service</code> - 字段过滤</li>
						</ul>
					</div>

					<div class="warn">
						<p><strong>GVR 说明：</strong>核心组资源使用 <code>group=core</code>（例如 Service: <code>/resources/core/v1/services</code>）；非核心组示例：Deployment 使用 <code>/resources/apps/v1/deployments</code>。</p>
						<p><a href="/kapis/auth/v1/healthz">Health Check</a></p>
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
