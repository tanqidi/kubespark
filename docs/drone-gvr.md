# Drone GVR 接口说明

本文记录 KubeSpark 后端对 Drone 的统一 GVR 接入约定，供前后端联调与后续维护使用。

## 1. 设计目标

- 不新增独立 `/drone/*` 路由
- 复用现有统一资源入口 `/kapis/v1alpha1/resources/{group}/{version}/{resource}`
- 保持响应包装与现有资源接口一致

## 2. 路由映射

Drone 通过虚拟 GVR 暴露：

- `group=drone`
- `version=v1`

支持资源：

- `repos`
- `builds`

完整路径：

- `GET /kapis/v1alpha1/resources/drone/v1/repos`
- `POST /kapis/v1alpha1/resources/drone/v1/repos`
- `PUT /kapis/v1alpha1/resources/drone/v1/repos/{name}`
- `DELETE /kapis/v1alpha1/resources/drone/v1/repos/{name}`
- `GET /kapis/v1alpha1/resources/drone/v1/builds`
- `POST /kapis/v1alpha1/resources/drone/v1/builds`
- `PUT /kapis/v1alpha1/resources/drone/v1/builds/{name}`
- `DELETE /kapis/v1alpha1/resources/drone/v1/builds/{name}`

## 3. 环境变量

服务启动前需要配置：

- `DRONE_SERVER`：Drone 服务地址，如 `http://drone.example.com`
- `DRONE_TOKEN`：Drone API Token

若缺失任一配置，接口返回 `503 Service Unavailable`。

## 4. 参数约定

## 4.1 repos

- 列表：无必填参数
- 创建：需要 `namespace` + 仓库名
  - 仓库名可来自 `repo` 查询参数或 body `metadata.name`
- 更新：需要 `namespace` + path `{name}`
- 删除：需要 `namespace` + path `{name}`

## 4.2 builds

- 列表：需要 `namespace` + `repo`
- 创建：需要 `namespace` + `repo`（body `spec` 透传给 Drone）
- 更新：语义为“重启构建”，需要 `namespace` + `repo` + path `{name}`
  - `{name}` 必须为构建号（正整数）
- 删除：语义为“停止构建”，需要 `namespace` + `repo` + path `{name}`
  - `{name}` 必须为构建号（正整数）

## 5. 与 Drone 上游 API 对照

- `GET repos` -> `GET /api/user/repos`
- `POST repos` -> `POST /api/repos/{namespace}/{name}`（激活仓库）
- `PUT repos/{name}` -> `PATCH /api/repos/{namespace}/{name}`
- `DELETE repos/{name}` -> `DELETE /api/repos/{namespace}/{name}`
- `GET builds` -> `GET /api/repos/{namespace}/{repo}/builds`
- `POST builds` -> `POST /api/repos/{namespace}/{repo}/builds`
- `PUT builds/{name}` -> `POST /api/repos/{namespace}/{repo}/builds/{number}`（重启）
- `DELETE builds/{name}` -> `DELETE /api/repos/{namespace}/{repo}/builds/{number}`（停止）

## 6. 错误处理约定

- 参数缺失或非法：返回 `400`
- Drone 请求失败：返回 `502`
- Drone 未配置：返回 `503`

返回结构遵循现有 kapis 统一封装，不额外定义新格式。

## 7. 联调示例

```bash
# 查询仓库列表
curl -X GET "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/repos" \
  -H "Authorization: Bearer <token>"

# 激活仓库
curl -X POST "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/repos?namespace=org" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"metadata":{"name":"demo-repo"}}'

# 查询构建列表
curl -X GET "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/builds?namespace=org&repo=demo-repo" \
  -H "Authorization: Bearer <token>"

# 触发构建
curl -X POST "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/builds?namespace=org&repo=demo-repo" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"spec":{"branch":"main"}}'

# 重启构建（构建号 12）
curl -X PUT "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/builds/12?namespace=org&repo=demo-repo" \
  -H "Authorization: Bearer <token>"

# 停止构建（构建号 12）
curl -X DELETE "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/builds/12?namespace=org&repo=demo-repo" \
  -H "Authorization: Bearer <token>"
```

