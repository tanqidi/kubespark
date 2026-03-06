# 前端请求说明（统一 GVR 接口）

本文用于前端联调时快速确认请求路径与参数规则，仅做概念和调用约定说明。

## 1. 统一入口

- 基础路径：`/kapis/resources.kubespark.io/v1alpha1`
- 统一资源路径：`/resources/{group}/{version}/{resource}`

说明：
- `group=core` 表示 Kubernetes 核心资源（如 pods、services、namespaces）。
- 非核心资源使用真实 group（如 `apps`、`batch`、`networking.k8s.io`）。

## 2. 常见请求场景

1. 列表查询  
使用：`GET /resources/{group}/{version}/{resource}`  
可附加筛选参数：`namespace`、`labelSelector`、`fieldSelector`。

2. 单条详情查询（当前推荐方式）  
使用列表接口 + 字段过滤：  
`GET /resources/{group}/{version}/{resource}?namespace={ns}&fieldSelector=metadata.name={name}`  

3. 创建资源  
使用：`POST /resources/{group}/{version}/{resource}`  
命名空间资源建议带 `namespace` 参数。

4. 更新资源  
使用：`PUT /resources/{group}/{version}/{resource}/{name}`  
命名空间资源建议带 `namespace` 参数。

5. 删除资源  
使用：`DELETE /resources/{group}/{version}/{resource}/{name}`  
命名空间资源建议带 `namespace` 参数。

## 3. namespace 参数约定

- 命名空间级资源（如 pods、services、deployments）应传 `namespace`。
- 集群级资源（如 namespaces、nodes、storageclasses、persistentvolumes）通常不需要 `namespace`。

## 4. 过滤参数约定

- `labelSelector`：按标签过滤。
- `fieldSelector`：按字段过滤，常用于按名称查询（`metadata.name=xxx`）。

## 5. 前端适配建议

- 资源列表页统一使用“列表接口 + 筛选参数”模式。
- 资源详情页统一使用“列表接口 + `fieldSelector=metadata.name=...`”模式。
- 删除操作使用统一删除路径，避免保留旧快捷接口分支。
- 前端路由层统一维护一份 GVR 映射，减少散落的硬编码路径。

## 6. 认证要求

- 资源接口默认需要 JWT Token。
- 请求头统一携带：`Authorization: Bearer {token}`。
- 认证与登录流程以 `docs/authentication.md` 为准。
