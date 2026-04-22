# Drone 集成说明（当前实现）

更新时间：2026-04-19

本文描述 `kubespark` 后端目前已上线的 Drone 集成能力，包含：

- Drone 资源 GVR 代理（`/resources/drone/v1/*`）
- Drone YAML Extension（`/kapis/v1alpha1/drone/yaml`）
- PipelineRun 自动触发 Drone Build 与状态回写

## 1. 配置来源（固定 Secret）

后端统一从固定 Secret 读取 Drone 配置：

- namespace: `kubespark`
- name: `kubespark-secret`

必需键：

- `DRONE_SERVER`
- `DRONE_TOKEN`
- `DRONE_YAML_SECRET`

可选键：

- `KUBESPARK_GITHUB_TOKEN`（用于分支列表 GitHub 回退查询）

说明：

- 不读取后端环境变量兜底。
- 缺少 `DRONE_SERVER` / `DRONE_TOKEN` 时，Drone 能力不可用（相关接口返回 `503`）。

## 2. 路由与能力边界

### 2.1 Drone YAML Extension

- `POST /kapis/v1alpha1/drone/yaml`
- `GET /kapis/v1alpha1/drone/yaml`（仅调试）

要点：

- 该接口不走 Kubespark JWT（由 Drone Server 调用）。
- 使用 `DRONE_YAML_SECRET` 做 HTTP Signature 验签。
- 命中返回 YAML；未命中返回 `204`，由 Drone 回退仓库 `.drone.yml`。

### 2.2 Drone GVR 代理

统一入口：

- `/kapis/v1alpha1/resources/drone/v1/{resource}`

当前资源支持：

- `repos`
- `reposync`
- `builds`
- `secrets`
- `branches`

当前支持的方法：

- `GET /resources/drone/v1/repos`
- `POST /resources/drone/v1/repos`
- `POST /resources/drone/v1/reposync`
- `GET /resources/drone/v1/builds`
- `POST /resources/drone/v1/builds`
- `GET /resources/drone/v1/secrets`
- `POST /resources/drone/v1/secrets`
- `DELETE /resources/drone/v1/secrets/{name}`
- `GET /resources/drone/v1/branches`

`branches` 兼容说明：

- 不再调用 Drone `/branches`。
- 直接调用 GitHub API：`GET https://api.github.com/repos/{owner}/{repo}/branches?per_page=100`
- 使用 `kubespark/kubespark-secret` 中 `KUBESPARK_GITHUB_TOKEN`。

当前不支持（会返回 unsupported）：

- `PUT /resources/drone/v1/repos/{name}`
- `DELETE /resources/drone/v1/repos/{name}`
- `PUT /resources/drone/v1/builds/{name}`
- `DELETE /resources/drone/v1/builds/{name}`
- `PUT /resources/drone/v1/secrets/{name}`

## 3. YAML 来源与匹配规则

`.drone.yml` 内容来自 `PipelineRun` 注解，不再从固定 YAML Secret 读取。

读取注解键：

- `tanqidi.com/drone-yaml`

匹配策略：

1. 从 Drone 请求体解析 `owner/repo`。
2. 要求 `build.repo_id > 0`（仅用于有效性约束）。
3. 在 `PipelineRun` 中筛选：
   - 注解 `tanqidi.com/drone-yaml` 非空；
   - 注解 `tanqidi.com/drone` 为空（未绑定 Drone 结果）；
   - `spec.data.namespace/repo`（或兼容键）与请求 `owner/repo` 一致。
4. 取“最新创建”的一条作为命中项返回。

## 4. PipelineRun 自动触发链路

当创建 `PipelineRun` 后：

1. 后端根据 `PipelineRun.spec.data` + `Pipeline.spec.data` 解析 `namespace/repo`。
2. 调用 Drone：
   - 检查仓库列表；
   - 若未激活则激活仓库；
   - 创建 build。
3. 将 Drone build 结果 JSON 写回：
   - `metadata.annotations["tanqidi.com/drone"]`
4. 后台同步任务会定时刷新该注解中的构建状态。

说明：

- 历史参数注入（例如 `kubespark_pipeline_run`、`build.inputs`、`build.action`）已移除，不再作为匹配依据。

## 5. 常见问题

### 5.1 401 Unauthorized（调用 Drone API）

通常为 `DRONE_TOKEN` 无效、过期或权限不足。  
先确认 `kubespark/kubespark-secret` 中 `DRONE_TOKEN` 正确，再重启后端。

### 5.2 500 Bad credentials（激活仓库/触发构建）

Drone 服务端凭据不可用（常见于 provider token 配置错误）。  
需在 Drone Server 侧修复 provider 认证配置，再重试。

### 5.3 `invalid drone yaml signature`

`DRONE_YAML_SECRET` 两端不一致，或修改后未重启对应进程。

