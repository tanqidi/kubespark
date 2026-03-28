# KubeSpark 项目结构与模块功能说明

## 项目概述

KubeSpark 是一个简化版的 KubeSphere，使用 Go 语言开发，基于 `client-go` 和 `go-restful` 框架，提供 Kubernetes 资源管理的 RESTful API。项目采用清晰的分层架构，将业务逻辑、API 处理、客户端封装等模块分离，便于维护和扩展。

## 项目结构

```
kubespark/
├── cmd/                          # 应用程序入口
│   └── server/
│       └── main.go              # 主程序入口，负责初始化并启动服务
├── pkg/                          # 可复用的公共包
│   ├── apiserver/               # API 服务器核心模块
│   │   ├── server.go           # HTTP 服务器封装
│   │   └── filters.go          # 中间件过滤器（CORS、日志）
│   ├── kapis/                   # KubeSphere API 兼容层
│   │   └── resources/
│   │       └── v1alpha1/       # v1alpha1 版本的资源 API
│   │           ├── handler.go  # API 请求处理器
│   │           └── routes.go   # 路由注册
│   ├── models/                  # 业务模型层
│   │   └── resources/
│   │       ├── interface.go    # 资源操作接口定义
│   │       └── resources.go    # 资源操作实现
│   └── simple/                  # 简单客户端封装
│       └── client/
│           └── k8s/
│               └── client.go   # Kubernetes 客户端封装
├── docs/                         # 项目文档
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖校验和
└── README.md                    # 项目说明文档
```

## 模块详细说明

### 1. cmd/server - 应用程序入口

**路径**: `cmd/server/main.go`

**功能**:
- 应用程序的主入口点
- 负责初始化各个组件并启动 HTTP 服务器
- 实现优雅关闭（Graceful Shutdown）

**主要职责**:
1. **Kubernetes 客户端初始化**: 创建 Kubernetes 客户端，支持集群内和集群外两种模式
2. **资源操作器创建**: 初始化资源操作器，用于执行 Kubernetes 资源操作
3. **API 处理器创建**: 创建 API 处理器，处理 HTTP 请求
4. **服务器创建**: 创建 HTTP 服务器实例
5. **路由注册**: 将 API 路由注册到服务器容器
6. **健康检查端点**: 提供 `/kapis/auth/v1/healthz` 端点，检查 Kubernetes API 连接状态
7. **欢迎页面**: 提供根路径 `/` 的 HTML 欢迎页面，展示所有可用的 API 端点
8. **优雅关闭**: 监听系统信号（SIGINT、SIGTERM），实现优雅关闭

**关键代码流程**:
```go
main() {
    1. 创建 K8s 客户端
    2. 创建资源操作器
    3. 创建 API 处理器
    4. 创建服务器
    5. 注册路由
    6. 注册健康检查和欢迎页面
    7. 启动服务器（支持优雅关闭）
}
```

---

### 2. pkg/apiserver - API 服务器核心模块

#### 2.1 server.go - HTTP 服务器封装

**功能**:
- 封装 HTTP 服务器和 go-restful 容器
- 提供服务器启动、关闭等生命周期管理
- 支持优雅关闭机制

**主要组件**:
- `Server` 结构体: 封装 HTTP 服务器和 restful 容器
- `NewServer()`: 创建新的服务器实例，初始化容器并注册过滤器
- `Container()`: 返回 restful 容器，供外部注册路由
- `Start()`: 启动 HTTP 服务器
- `Shutdown()`: 优雅关闭服务器
- `StartWithGracefulShutdown()`: 启动服务器并处理优雅关闭信号

**特性**:
- 自动注册 CORS 和日志过滤器
- 30 秒超时的优雅关闭
- 支持自定义端口

#### 2.2 filters.go - 中间件过滤器

**功能**:
- 提供 HTTP 请求/响应过滤器
- 实现跨域资源共享（CORS）
- 实现请求日志记录

**过滤器说明**:

1. **CORSFilter**:
   - 允许所有来源的跨域请求（`Access-Control-Allow-Origin: *`）
   - 支持所有 HTTP 方法（GET、POST、PUT、DELETE、PATCH、OPTIONS）
   - 允许常用请求头（Content-Type、Authorization）
   - 处理 OPTIONS 预检请求

2. **LoggingFilter**:
   - 记录所有 HTTP 请求的详细信息
   - 记录请求时间、方法、路径、状态码和响应时间
   - 格式: `[时间] 方法 路径 - 状态码 (响应时间)`

---

### 3. pkg/simple/client/k8s - Kubernetes 客户端封装

**路径**: `pkg/simple/client/k8s/client.go`

**功能**:
- 封装 Kubernetes 客户端创建逻辑
- 支持集群内和集群外两种运行模式
- 提供标准客户端和动态客户端

**主要组件**:

1. **Interface 接口**:
   - `Kubernetes()`: 返回标准 Kubernetes 客户端接口
   - `Dynamic()`: 返回动态客户端接口，用于处理所有类型的资源

2. **Client 结构体**:
   - 封装 `kubernetes.Interface` 和 `dynamic.Interface`
   - 实现 `Interface` 接口

3. **NewClient() 函数**:
   - 自动检测运行环境（集群内/集群外）
   - 优先尝试使用集群内配置（`rest.InClusterConfig()`）
   - 如果失败，回退到 kubeconfig 文件（`~/.kube/config`）
   - 创建标准客户端和动态客户端

**配置优先级**:
1. 集群内配置（ServiceAccount Token）
2. kubeconfig 文件（`~/.kube/config`）

**使用场景**:
- 集群内运行: 使用 Pod 的 ServiceAccount 自动获取认证信息
- 集群外运行: 使用本地 kubeconfig 文件

---

### 4. pkg/models/resources - 业务模型层

#### 4.1 interface.go - 接口定义

**功能**:
- 定义资源操作的接口规范
- 提供清晰的抽象层，便于测试和扩展

**接口方法**:
- `ListResources()`: 按类型列出资源（支持命名空间过滤）
- `GetResource()`: 获取指定资源详情
- `ListResourcesByGVR()`: 通过 GVR（Group Version Resource）列出资源
- `GetClusterInfo()`: 获取集群信息（版本、节点数、命名空间数等）
- `GetPodLogs()`: 获取 Pod 日志

#### 4.2 resources.go - 资源操作实现

**功能**:
- 实现 `Interface` 接口定义的所有方法
- 封装 Kubernetes 资源操作逻辑
- 支持多种资源类型的查询

**ResourcesOperator 结构体**:
- 包含 `k8sClient`，用于执行 Kubernetes API 调用
- 实现所有接口方法

**支持的资源类型**:
- **核心资源**: Pods、Services、ConfigMaps、Secrets、Events、Namespaces、Nodes
- **工作负载**: Deployments、StatefulSets、DaemonSets、ReplicaSets、Jobs、CronJobs
- **存储**: PersistentVolumes、PersistentVolumeClaims、StorageClasses
- **网络**: Ingresses

**主要方法说明**:

1. **ListResources()**:
   - 根据资源类型和命名空间列出资源
   - 支持 `ListOptions`（标签选择器、字段选择器等）
   - 返回 `runtime.Object`，可序列化为 JSON

2. **GetResource()**:
   - 获取指定命名空间中的指定资源
   - 支持 `GetOptions`

3. **ListResourcesByGVR()**:
   - 使用动态客户端查询任意资源类型
   - 支持集群级别和命名空间级别资源
   - 适用于自定义资源（CRD）

4. **GetClusterInfo()**:
   - 获取 Kubernetes 集群版本信息
   - 统计节点数量和命名空间数量
   - 返回服务器时间

5. **GetPodLogs()**:
   - 获取 Pod 的容器日志
   - 支持指定容器名称
   - 支持限制日志行数（tailLines）
   - 包含时间戳

---

### 5. pkg/kapis/resources/v1alpha1 - API 处理层

#### 5.1 handler.go - API 请求处理器

**功能**:
- 处理所有 HTTP API 请求
- 将 HTTP 请求转换为资源操作调用
- 处理错误并返回适当的 HTTP 响应

**Handler 结构体**:
- 包含 `resourcesOperator`，用于执行实际的资源操作

**主要处理方法**:

1. **集群信息**:
   - `GetClusterInfo()`: 返回集群版本、节点数等信息

2. **动态资源 CRUD**:
   - `GetResourceByGVR()`: 通过 GVR 查询任意资源类型
   - `CreateResourceByGVR()`: 通过 GVR 创建资源
   - `UpdateResourceByGVR()`: 通过 GVR 更新资源
   - `DeleteResourceByGVR()`: 通过 GVR 删除资源

**辅助方法**:
- `parseListOptions()`: 解析查询参数为 `ListOptions`
- `handleError()`: 统一错误处理，将 Kubernetes 错误转换为 HTTP 响应

#### 5.2 routes.go - 路由注册

**功能**:
- 定义所有 API 路由
- 注册路由到 go-restful 容器
- 定义路由参数和文档

**路由结构**:
- 基础路径: `/kapis/v1alpha1`
- 所有 API 端点都在此路径下

**路由分类**:

1. **集群信息路由**:
   - `/cluster-info` - 获取集群信息

2. **动态资源路由（统一入口）**:
   - `/resources/{group}/{version}/{resource}` - 通过 GVR 查询资源
   - `/resources/{group}/{version}/{resource}` (POST) - 通过 GVR 创建资源
   - `/resources/{group}/{version}/{resource}/{name}` (PUT) - 通过 GVR 更新资源
   - `/resources/{group}/{version}/{resource}/{name}` (DELETE) - 通过 GVR 删除资源

**单条详情说明**:
- 当前未暴露 `GET /resources/{group}/{version}/{resource}/{name}`。
- 前端查询单条资源建议使用列表接口 + 字段选择器：
  - `/resources/{group}/{version}/{resource}?namespace={ns}&fieldSelector=metadata.name={name}`

**路由特性**:
- 所有路由都支持 JSON 格式的请求和响应
- 支持查询参数（namespace、labelSelector、fieldSelector）
- 包含路由文档说明（`.Doc()`）
- 定义路径参数和查询参数的数据类型

---

## 技术栈

### 核心依赖

1. **go-restful/v3**: RESTful Web 服务框架
   - 提供路由、过滤器、参数绑定等功能
   - 支持 OpenAPI 文档生成

2. **k8s.io/client-go**: Kubernetes 官方 Go 客户端
   - 提供标准客户端（kubernetes.Interface）
   - 提供动态客户端（dynamic.Interface）
   - 支持集群内和集群外配置

3. **k8s.io/api**: Kubernetes API 类型定义
   - 包含所有 Kubernetes 资源类型

4. **k8s.io/apimachinery**: Kubernetes 核心工具库
   - 提供通用类型（runtime.Object、metav1.ListOptions 等）
   - 提供错误处理（errors.StatusError）

### Go 版本要求

- Go 1.21 或更高版本

---

## 模块依赖关系

```
cmd/server/main.go
    ├── pkg/apiserver (服务器封装)
    ├── pkg/kapis/resources/v1alpha1 (API 处理器)
    ├── pkg/models/resources (业务逻辑)
    └── pkg/simple/client/k8s (K8s 客户端)

pkg/kapis/resources/v1alpha1
    └── pkg/models/resources (依赖业务模型层)

pkg/models/resources
    └── pkg/simple/client/k8s (依赖 K8s 客户端)

pkg/apiserver
    └── go-restful/v3 (依赖外部库)
```

**依赖层次**:
1. **最底层**: `pkg/simple/client/k8s` - 客户端封装，不依赖其他业务模块
2. **业务层**: `pkg/models/resources` - 业务逻辑，依赖客户端
3. **API 层**: `pkg/kapis/resources/v1alpha1` - API 处理，依赖业务层
4. **服务器层**: `pkg/apiserver` - 服务器封装，独立模块
5. **入口层**: `cmd/server/main.go` - 组装所有模块

---

## 数据流

### 请求处理流程

```
HTTP 请求
    ↓
go-restful Container
    ↓
CORSFilter (添加 CORS 头)
    ↓
LoggingFilter (记录请求日志)
    ↓
Routes (路由匹配)
    ↓
Handler (API 处理器)
    ↓
ResourcesOperator (业务逻辑)
    ↓
K8s Client (Kubernetes API 调用)
    ↓
Kubernetes API Server
    ↓
响应返回（反向流程）
```

### 示例：获取 Pod 列表（统一 GVR）

1. 客户端发送: `GET /kapis/v1alpha1/resources/core/v1/pods?namespace=default`
2. CORSFilter: 添加 CORS 响应头
3. LoggingFilter: 记录请求开始时间
4. Routes: 匹配到 `GET /resources/{group}/{version}/{resource}`
5. Handler.GetResourceByGVR(): 构造 `GroupVersionResource`
6. ResourcesOperator.ListResourcesByGVR(): 调用动态客户端
7. K8s Client: 执行 `dynamicClient.Resource(gvr).Namespace("default").List()`
8. Kubernetes API: 返回 Pod 列表
9. 响应序列化: 将结果序列化为 JSON
10. LoggingFilter: 记录响应时间和状态码
11. 返回响应: 发送 JSON 响应给客户端

---

## 扩展指南

### 添加新的资源类型

1. **在 `pkg/models/resources/resources.go` 中**:
   - 在 `ListResources()` 的 switch 语句中添加新的 case
   - 在 `GetResource()` 的 switch 语句中添加新的 case（如果需要）

2. **优先使用统一 GVR 路由**:
   - 大多数内置资源和 CRD 都可直接通过 `/resources/{group}/{version}/{resource}` 访问
   - 通常无需新增固定快捷路由

### 添加新的 API 端点

1. 在 `handler.go` 中实现处理方法
2. 在 `routes.go` 中注册路由
3. 如果需要新的业务逻辑，在 `resources.go` 中添加

### 添加新的中间件

1. 在 `pkg/apiserver/filters.go` 中实现新的过滤器函数
2. 在 `pkg/apiserver/server.go` 的 `NewServer()` 中注册过滤器

---

## 总结

KubeSpark 项目采用清晰的分层架构，各模块职责明确：

- **cmd/server**: 应用程序入口，负责组件组装和启动
- **pkg/apiserver**: 服务器核心，提供 HTTP 服务和中间件
- **pkg/simple/client/k8s**: 客户端封装，处理 Kubernetes 连接
- **pkg/models/resources**: 业务逻辑层，封装资源操作
- **pkg/kapis/resources/v1alpha1**: API 处理层，处理 HTTP 请求

这种架构设计使得项目易于维护、测试和扩展，同时保持了代码的清晰性和可读性。
