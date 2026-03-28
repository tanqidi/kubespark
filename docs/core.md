# Kubernetes 中的 group 概念说明（含 core 的特殊性）

## 1. group 是什么？

在 Kubernetes 中，每一种资源都通过 **GVR** 唯一定位：

- **G**roup（API Group）
- **V**ersion（API Version）
- **R**esource（Resource）

也就是说，一个 Kubernetes 资源在 API 层面的唯一身份是：

Group + Version + Resource

在 Go 代码中通常表现为：

schema.GroupVersionResource

---

## 2. 为什么需要 group？

Kubernetes 早期只有一套 API：

/api/v1

当时只有 Pod、Service、Node 等基础资源，根本不存在 group 的概念。

随着 Kubernetes 的发展，资源数量快速增长（Deployment、Ingress、HPA、CRD 等），为了避免 API 冲突并实现更清晰的职责划分，引入了 **API Group**：

/apis/{group}/{version}

每个 group 负责一类资源。

---

## 3. core（Core API）的特殊性

Pod、Service、ConfigMap、Secret 等资源属于 **Core API（也叫 Legacy API）**。

### 核心特性：

- Core API **没有 group**
- 在 Kubernetes 内部代码中，Core API 的 group 使用 **空字符串 ""** 表示

示例：

- Pod
  - group: ""
  - version: v1
  - resource: pods

对应的原生 API 路径：

/api/v1/namespaces/default/pods

注意：这里没有 group

---

## 4. 非 core 资源的示例

Deployment 属于 apps group：

- group: apps
- version: v1
- resource: deployments

对应的 API 路径：

/apis/apps/v1/namespaces/default/deployments

---

## 5. 常见资源与 group 对照表

| 资源 | group | version |
|----|----|----|
| Pod | ""（core） | v1 |
| Service | ""（core） | v1 |
| ConfigMap | ""（core） | v1 |
| Node | ""（core） | v1 |
| Deployment | apps | v1 |
| StatefulSet | apps | v1 |
| DaemonSet | apps | v1 |
| Job | batch | v1 |
| CronJob | batch | v1 |
| Ingress | networking.k8s.io | v1 |
| HPA | autoscaling | v2 |
| CRD | apiextensions.k8s.io | v1 |

---

## 6. 为什么有些系统里会看到 group = "core"？

像 KubeSphere、kubespark 等平台，为了统一前端和 REST API 的结构，**人为引入了 "core" 这个字符串**，用于表示 Core API。

例如：

/kapis/v1alpha1/resources/core/v1/pods

但在真正调用 Kubernetes client-go / dynamic client 时，必须做一次转换：

if group == "core" {
    group = ""
}

这是因为 Kubernetes **只认空字符串的 core group**。

---

## 7. GVR 在代码中的真实含义

在 client-go 中：

schema.GroupVersionResource{
    Group:    group,
    Version: version,
    Resource: resource,
}

表示：

“我要操作哪一个 API group 下、哪个版本中的哪一类资源”

当操作 Pod 时，最终必须是：

Group: ""

否则请求会失败。

---

## 8. 总结

- group 是 Kubernetes API 的逻辑分组
- Core API 没有 group，用空字符串表示
- "core" 只是上层系统的抽象概念，不是 Kubernetes 的真实 group
- 所有通用 Kubernetes 资源操作代码，都必须特殊处理 core → ""

这一点是编写通用 Kubernetes CRUD / 平台型系统时的必备知识。
