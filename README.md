# KubeSpark

KubeSpark 是一个基于 Go + client-go + go-restful 的 Kubernetes 资源管理 API 服务。

## API 路径约定

- 认证接口前缀：`/kapis/auth/v1`
- 资源接口前缀：`/kapis/v1alpha1`

## 快速启动

### 1. 本地启动

```bash
go run ./cmd/server --port=8080
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

## 部署

参考 `manifests/`：

- `manifests/rbac.yaml`
- `manifests/deployment.yaml`
- `manifests/service.yaml`

## 相关文档

- `docs/authentication.md`
- `docs/frontend-request-guide.md`
- `docs/project-structure.md`
- `docs/core.md`
- `docs/in-cluster-config-explained.md`
