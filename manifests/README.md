# Kubespark 部署配置

## 配置说明

### ServiceAccount 配置位置

**ServiceAccount 是在 Deployment 的 `spec.template.spec.serviceAccountName` 字段中配置的。**

具体位置：
- 文件：`manifests/deployment.yaml`
- 字段：`spec.template.spec.serviceAccountName: kubespark`

## 部署步骤

### 1. 创建 ServiceAccount 和 RBAC

```bash
kubectl apply -f manifests/serviceaccount.yaml
kubectl apply -f manifests/clusterrole.yaml
kubectl apply -f manifests/clusterrolebinding.yaml
```

### 2. 部署应用

```bash
# 先构建镜像（根据实际情况修改）
docker build -t kubespark:latest .

# 部署
kubectl apply -f manifests/deployment.yaml
kubectl apply -f manifests/service.yaml
```

### 3. 验证

```bash
# 检查 Pod 是否运行
kubectl get pods -l app=kubespark

# 检查 ServiceAccount token 是否挂载
kubectl exec -it <pod-name> -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# 应该能看到：
# - token
# - ca.crt
# - namespace
```

## 关键配置说明

### ServiceAccount
- 名称：`kubespark`
- 命名空间：`default`（可根据需要修改）

### ClusterRole
- 权限：超级管理员（所有资源的所有操作）
- 名称：`kubespark-admin`

### Deployment
- **`serviceAccountName: kubespark`**：指定使用哪个 ServiceAccount
- **`automountServiceAccountToken: true`**：确保自动挂载 token（默认就是 true）

## 工作原理

1. Pod 启动时，Kubernetes 会自动将 ServiceAccount 的 token 挂载到：
   `/var/run/secrets/kubernetes.io/serviceaccount/`

2. `kubespark/pkg/simple/client/k8s/client.go` 中的 `NewClient()` 函数会：
   - 首先调用 `rest.InClusterConfig()` 读取挂载的 token
   - 如果成功，就使用集群内配置（不需要 kubeconfig）
   - 如果失败，才回退到读取 `~/.kube/config`

3. 因此，只要 Pod 中配置了 `serviceAccountName`，就会自动使用集群内配置。

## 修改命名空间

如果要在其他命名空间部署，需要修改以下文件中的 `namespace`：

1. `serviceaccount.yaml` - metadata.namespace
2. `clusterrolebinding.yaml` - subjects[0].namespace
3. `deployment.yaml` - metadata.namespace
4. `service.yaml` - metadata.namespace

