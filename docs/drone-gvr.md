# Drone 集成说明（GVR + YAML 扩展）

本文记录 KubeSpark 后端当前对 Drone 的两类能力：

- Drone API 的 GVR 代理（统一资源入口）
- Drone Configuration Extension（动态 YAML，下发前读取平台配置）

## 1. 路由

### 1.1 Drone GVR 代理

统一入口：

- `/kapis/v1alpha1/resources/{group}/{version}/{resource}`
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

### 1.2 Drone YAML 扩展接口

- `POST /kapis/v1alpha1/drone/yaml`
- `GET /kapis/v1alpha1/drone/yaml`（用于手工调试）

说明：

- 该接口不是给前端调用，不走 JWT。
- 该接口使用 `DRONE_YAML_SECRET` 做 HTTP Signature 验签（Drone 官方扩展机制）。

## 2. 环境变量

### 2.1 Kubespark 后端

- `DRONE_SERVER`：Drone 地址，例如 `http://172.31.0.88:30001`
- `DRONE_TOKEN`：Drone API Token（触发构建）
- `DRONE_YAML_SECRET`：YAML 扩展验签密钥（必须和 Drone Server 一致）

### 2.2 Drone Server

- `DRONE_YAML_ENDPOINT`：例如 `http://172.31.0.88:8080/kapis/v1alpha1/drone/yaml`
- `DRONE_YAML_SECRET`：与 Kubespark 后端同值

## 3. YAML 来源（平台托管）

当前实现从 K8s Secret 读取 YAML：

- namespace：`kubespark`
- name：`kubespark-drone-yaml-secret`

按 owner/repo 匹配 key，优先级：

1. `owner__repo`
2. `owner_repo`
3. `owner-repo`
4. `owner.repo`
5. `repo`

命中后返回 YAML；未命中返回 `204`（Drone 回退仓库 `.drone.yml`）。

## 4. YAML 扩展响应协议

Drone 请求头通常为：

- `Accept: application/vnd.drone.config.v1+json`

后端响应策略：

- `Accept` 包含 `json`：返回 `application/json`，格式 `{"data":"<yaml>"}`。
- 其他：返回 `text/plain` 原始 YAML。

## 5. GVR 参数约定

### 5.1 repos

- 列表：无必填参数
- 创建：需要 `namespace` + 仓库名（`repo` 查询参数或 body `metadata.name`）
- 更新：需要 `namespace` + path `{name}`
- 删除：需要 `namespace` + path `{name}`

### 5.2 builds

- 列表：需要 `namespace` + `repo`
- 创建：需要 `namespace` + `repo`（body `spec` 透传给 Drone）
- 更新（重启）：需要 `namespace` + `repo` + path `{name}`，`{name}` 为构建号
- 删除（停止）：需要 `namespace` + `repo` + path `{name}`，`{name}` 为构建号

## 6. 常见故障排查

### 6.1 `missing DRONE_YAML_SECRET`

含义：处理 `/drone/yaml` 的进程未配置该变量。  
注意：变量必须配置到“实际运行该 HTTP 服务”的进程（容器或主机进程）。

### 6.2 `invalid drone yaml signature`

优先排查：

1. Drone 与 Kubespark 的 `DRONE_YAML_SECRET` 是否完全一致。
2. 修改变量后是否重启了对应进程。
3. 请求是否经过会改写签名相关头的代理。

### 6.3 `406: Not Acceptable`

通常是扩展响应协商不匹配。当前实现已兼容 Drone vendor accept（`application/vnd.drone.config.v1+json`）。

### 6.4 `invalid character 'k' looking for beginning of value`

含义：Drone 按 JSON 解析响应，但接口返回了纯文本 YAML。  
当前实现已按 `Accept` 自动返回 JSON 包装。

## 7. 调试示例

```bash
# 手工读取 YAML（调试）
curl -i "http://localhost:8080/kapis/v1alpha1/drone/yaml?owner=tanqidi&repo=kubespark"

# 通过 GVR 查询构建
curl -X GET "http://localhost:8080/kapis/v1alpha1/resources/drone/v1/builds?namespace=tanqidi&repo=kubespark" \
  -H "Authorization: Bearer <kubespark-jwt>"
```
