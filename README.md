# KubeSpark

KubeSpark 是一个基于 Go + client-go + go-restful 的 Kubernetes 资源管理 API 服务。

## API 路径约定

- 认证接口前缀：`/kapis/auth/v1`
- 资源接口前缀：`/kapis/v1alpha1`

## 快速启动

### 1. 本地启动

```bash
go run ./cmd/server/main.go
```

### 2. 打开文档

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI JSON: `http://localhost:8080/apidocs.json`

## 认证流程

### 1. 登录获取 Token

```bash
curl -X POST http://localhost:8080/kapis/auth/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

### 2. 携带 Token 访问资源接口

```bash
curl -X GET "http://localhost:8080/kapis/v1alpha1/resources/core/v1/namespaces" \
  -H "Authorization: Bearer <your-token>"
```

## 统一资源接口（GVR）

统一入口：

- `GET /kapis/v1alpha1/resources/{group}/{version}/{resource}`
- `POST /kapis/v1alpha1/resources/{group}/{version}/{resource}`
- `PUT /kapis/v1alpha1/resources/{group}/{version}/{resource}/{name}`
- `DELETE /kapis/v1alpha1/resources/{group}/{version}/{resource}/{name}`

示例：

- 查询 Pod 列表（core 组）：

```bash
curl -X GET "http://localhost:8080/kapis/v1alpha1/resources/core/v1/pods?namespace=default" \
  -H "Authorization: Bearer <your-token>"
```

说明：`core` 会在服务端转换为 Kubernetes 真正的空 group（`""`）。

## 健康检查

```bash
curl -X GET http://localhost:8080/kapis/auth/v1/healthz \
  -H "Authorization: Bearer <your-token>"
```

## 工作原理

1. Pod 启动时，Kubernetes 会自动将 ServiceAccount 的 token 挂载到：
   `/var/run/secrets/kubernetes.io/serviceaccount/`

2. `kubespark/pkg/simple/client/k8s/client.go` 中的 `NewClient()` 函数会：
   - 首先调用 `rest.InClusterConfig()` 读取挂载的 token
   - 如果成功，就使用集群内配置（不需要 kubeconfig）
   - 如果失败，才回退到读取 `~/.kube/config`

3. 因此，只要 Pod 中配置了 `serviceAccountName`，就会自动使用集群内配置。

## 相关文档

- `docs/authentication.md`
- `docs/frontend-request-guide.md`
- `docs/project-structure.md`
- `docs/core.md`
- `docs/in-cluster-config-explained.md`

