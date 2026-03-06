# 登录认证流程详解

## 概述

KubeSpark 使用基于 JWT (JSON Web Token) 的认证机制来保护 API 端点。用户需要通过登录接口获取 JWT Token，然后在后续的 API 请求中使用该 Token 进行身份验证。

## 认证配置

### 环境变量配置

系统支持通过环境变量配置登录凭据和 JWT 密钥，如果未设置环境变量，将使用默认值：

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `KUBESPARK_USERNAME` | 登录用户名 | `admin` |
| `KUBESPARK_PASSWORD` | 登录密码 | `123456` |
| `KUBESPARK_JWT_SECRET` | JWT 签名密钥（用于签发和验证 Token） | `kubespark-secret-key-change-in-production` |

**重要提示**：
- `KUBESPARK_JWT_SECRET` 是用于签名和验证 JWT Token 的密钥
- 在生产环境中，**必须**设置一个强随机密钥，不要使用默认值
- 如果修改了 JWT Secret，之前签发的所有 Token 将失效，需要重新登录

### 配置方式

**方式1：使用环境变量（推荐）**

```bash
export KUBESPARK_USERNAME=myuser
export KUBESPARK_PASSWORD=mypassword
export KUBESPARK_JWT_SECRET=your-strong-random-secret-key-here
./server
```

**方式2：使用默认值**

如果不设置环境变量，系统将使用默认值：
- 用户名：`admin`
- 密码：`123456`
- JWT Secret：`kubespark-secret-key-change-in-production`（**生产环境不推荐使用**）

### JWT Secret 配置说明

JWT Secret 是用于签名和验证 JWT Token 的密钥字符串。它配置在以下位置：

**代码位置**：`pkg/kapis/auth/jwt.go`

**配置方式**：
1. **通过环境变量**（推荐）：设置 `KUBESPARK_JWT_SECRET` 环境变量
2. **使用默认值**：如果不设置环境变量，将使用代码中的默认值

**生成强随机密钥的方法**：

```bash
# 使用 openssl 生成随机密钥（推荐）
openssl rand -base64 32

# 或使用其他方法
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

**配置示例**：

```bash
# 生成并设置 JWT Secret
export KUBESPARK_JWT_SECRET=$(openssl rand -base64 32)
echo "JWT Secret: $KUBESPARK_JWT_SECRET"

# 启动服务
./server
```

## 登录端点

### 基本信息

- **URL**: `/kapis/auth.kubespark.io/v1/login`
- **方法**: `POST`
- **Content-Type**: `application/json`
- **认证**: 无需认证（公开端点）

### 请求格式

```json
{
  "username": "admin",
  "password": "123456"
}
```

### 成功响应

**状态码**: `200 OK`

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 错误响应

**无效凭据** (401 Unauthorized)

```json
{
  "code": 401,
  "message": "Invalid username or password"
}
```

**请求格式错误** (400 Bad Request)

```json
{
  "code": 400,
  "message": "Invalid request body"
}
```

**Token 生成失败** (500 Internal Server Error)

```json
{
  "code": 500,
  "message": "Failed to generate token"
}
```

## 登录流程详解

### 流程图

```
┌─────────────┐
│   客户端     │
└──────┬──────┘
       │ 1. POST /login (username, password)
       ▼
┌─────────────────────────────────┐
│   Auth Handler (Login)          │
│   - 读取请求体                   │
│   - 验证请求格式                 │
└──────┬──────────────────────────┘
       │ 2. 获取配置的凭据
       ▼
┌─────────────────────────────────┐
│   getCredentials()              │
│   - 读取 KUBESPARK_USERNAME      │
│   - 读取 KUBESPARK_PASSWORD      │
│   - 使用默认值（如未设置）        │
└──────┬──────────────────────────┘
       │ 3. 验证凭据
       ▼
┌─────────────────────────────────┐
│   凭据验证                       │
│   - 比较用户名                   │
│   - 比较密码                     │
└──────┬──────────────────────────┘
       │ 4. 生成 JWT Token
       ▼
┌─────────────────────────────────┐
│   GenerateToken()               │
│   - 创建 Claims                 │
│   - 设置过期时间（24小时）       │
│   - 签名 Token                  │
└──────┬──────────────────────────┘
       │ 5. 返回 Token
       ▼
┌─────────────┐
│   客户端     │
│   (保存Token)│
└─────────────┘
```

### 详细步骤

#### 步骤 1: 客户端发送登录请求

客户端向登录端点发送 POST 请求，包含用户名和密码：

```bash
curl -X POST http://localhost:8080/kapis/auth.kubespark.io/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "123456"
  }'
```

#### 步骤 2: 服务器解析请求

服务器读取并解析请求体，验证 JSON 格式是否正确：

```go
var loginReq LoginRequest
if err := req.ReadEntity(&loginReq); err != nil {
    // 返回 400 Bad Request
    return
}
```

#### 步骤 3: 获取配置的凭据

服务器调用 `getCredentials()` 函数获取配置的用户名和密码：

```go
func getCredentials() (username, password string) {
    username = os.Getenv("KUBESPARK_USERNAME")
    if username == "" {
        username = "admin"  // 默认值
    }

    password = os.Getenv("KUBESPARK_PASSWORD")
    if password == "" {
        password = "123456"  // 默认值
    }

    return username, password
}
```

#### 步骤 4: 验证凭据

服务器比较客户端提供的凭据与配置的凭据：

```go
expectedUsername, expectedPassword := getCredentials()

if loginReq.Username != expectedUsername || 
   loginReq.Password != expectedPassword {
    // 返回 401 Unauthorized
    return
}
```

#### 步骤 5: 生成 JWT Token

如果凭据验证通过，服务器生成 JWT Token：

```go
token, err := GenerateToken(loginReq.Username, loginReq.Password)
```

**Token 包含的信息**：
- `username`: 用户名
- `password_verifier`: 密码验证器（HMAC-SHA256(secret, password)）
- `exp`: 过期时间（24小时后）
- `iat`: 签发时间
- `nbf`: 生效时间
- `iss`: 签发者（"kubespark"）
- `sub`: 主题（用户名）

**Token 配置**：
- 签名算法: `HS256`
- 有效期: `24 小时`
- 密钥: 通过 `KUBESPARK_JWT_SECRET` 环境变量配置，默认值为 `kubespark-secret-key-change-in-production`（生产环境必须修改）

**安全特性**：
- **密码验证器机制**: Token 中包含密码验证器（HMAC-SHA256），即使攻击者获得了 JWT Secret 和用户名，也无法生成有效 Token，因为需要知道实际密码
- **防止离线暴力破解**: 使用 HMAC 而非普通哈希，攻击者即使看到 Token 中的验证器，也无法离线破解密码，因为需要 JWT Secret 来验证任何密码猜测

#### 步骤 6: 返回 Token

服务器将生成的 Token 返回给客户端：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

## JWT Token 说明

### Token 结构

JWT Token 由三部分组成，用 `.` 分隔：

```
header.payload.signature
```

**Header** (Base64 编码):
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload** (Base64 编码):
```json
{
  "username": "admin",
  "password_verifier": "a1b2c3d4e5f6...",
  "exp": 1234567890,
  "iat": 1234567890,
  "nbf": 1234567890,
  "iss": "kubespark",
  "sub": "admin"
}
```

**password_verifier 说明**：
- 使用 HMAC-SHA256 算法计算：`HMAC-SHA256(JWT_SECRET, password)`
- 以十六进制字符串形式存储在 Token 中
- 用于防止攻击者在获得 JWT Secret 和用户名后生成有效 Token
- 即使攻击者看到验证器，也无法离线破解密码，因为需要 JWT Secret 来验证密码猜测

**Signature**:
```
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret
)
```

### Token 验证

Token 验证采用**三层验证机制**，确保最高级别的安全性：

1. **签名验证**：验证 Token 的 HMAC-SHA256 签名，确保 Token 未被篡改
2. **过期时间验证**：检查 `exp` 声明，确保 Token 未过期
3. **生效时间验证**：检查 `nbf` 声明，确保 Token 已生效
4. **签名方法验证**：确保使用正确的签名算法（HS256）
5. **用户名验证**：验证 Token 中的用户名与配置的用户名匹配
6. **密码验证器验证**：验证 Token 中的密码验证器与配置密码的验证器匹配

**安全优势**：
- **防止 Token 伪造**：即使攻击者获得了 JWT Secret 和用户名，也无法生成有效 Token，因为需要知道实际密码来计算正确的密码验证器
- **防止离线暴力破解**：密码验证器使用 HMAC-SHA256，攻击者即使看到 Token 中的验证器，也无法离线破解密码，因为需要 JWT Secret 来验证任何密码猜测
- **防止 Token 重用**：如果密码被更改，旧的 Token 将立即失效，因为密码验证器不匹配

## 使用 Token 访问 API

### 认证过滤器

所有 API 请求（除登录端点外）都需要通过 `AuthFilter` 进行认证验证。

**免认证的端点**：
- `/kapis/auth.kubespark.io/v1/login` - 登录端点
- `/swagger` - API 文档
- `/` - 根路径
- `/apidocs.json` - API 文档 JSON
- `OPTIONS` 请求 - CORS 预检请求

### 请求头格式

在后续的 API 请求中，需要在 `Authorization` 头中携带 Token：

```
Authorization: Bearer <token>
```

### 认证流程

```
┌─────────────┐
│   客户端     │
└──────┬──────┘
       │ 1. API 请求 + Authorization Header
       ▼
┌─────────────────────────────────┐
│   AuthFilter                    │
│   - 检查是否为免认证端点         │
│   - 提取 Authorization Header   │
└──────┬──────────────────────────┘
       │ 2. 解析 Bearer Token
       ▼
┌─────────────────────────────────┐
│   ValidateToken()               │
│   - 解析 Token                   │
│   - 验证签名（HMAC-SHA256）      │
│   - 检查过期时间                 │
│   - 验证用户名匹配               │
│   - 验证密码验证器匹配           │
└──────┬──────────────────────────┘
       │ 3. 提取用户名
       ▼
┌─────────────────────────────────┐
│   存储到请求上下文               │
│   req.SetAttribute("username")  │
└──────┬──────────────────────────┘
       │ 4. 继续处理请求
       ▼
┌─────────────┐
│   API Handler│
└─────────────┘
```

### 示例：使用 Token 访问 API

```bash
# 1. 登录获取 Token
TOKEN=$(curl -X POST http://localhost:8080/kapis/auth.kubespark.io/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}' \
  | jq -r '.data.token')

# 2. 使用 Token 访问受保护的 API
curl -X GET http://localhost:8080/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces \
  -H "Authorization: Bearer $TOKEN"
```

## 错误处理

### 常见错误场景

#### 1. 缺少 Token

**请求**:
```bash
curl -X GET http://localhost:8080/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces
```

**响应** (401 Unauthorized):
```json
{
  "code": 401,
  "message": "Missing authorization token"
}
```

#### 2. Token 格式错误

**请求**:
```bash
curl -X GET http://localhost:8080/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces \
  -H "Authorization: InvalidFormat token123"
```

**响应** (401 Unauthorized):
```json
{
  "code": 401,
  "message": "Invalid authorization header format"
}
```

**正确的格式**:
```
Authorization: Bearer <token>
```

#### 3. Token 无效或过期

**请求**:
```bash
curl -X GET http://localhost:8080/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces \
  -H "Authorization: Bearer invalid_token"
```

**响应** (401 Unauthorized):
```json
{
  "code": 401,
  "message": "Invalid or expired token"
}
```

**解决方案**: 重新登录获取新的 Token

#### 4. 登录凭据错误

**请求**:
```bash
curl -X POST http://localhost:8080/kapis/auth.kubespark.io/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"wrong","password":"wrong"}'
```

**响应** (401 Unauthorized):
```json
{
  "code": 401,
  "message": "Invalid username or password"
}
```

## 代码示例

### Go 客户端示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    struct {
        Token string `json:"token"`
    } `json:"data"`
}

func login(baseURL, username, password string) (string, error) {
    loginReq := LoginRequest{
        Username: username,
        Password: password,
    }

    jsonData, _ := json.Marshal(loginReq)
    resp, err := http.Post(
        baseURL+"/kapis/auth.kubespark.io/v1/login",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    var loginResp LoginResponse
    json.Unmarshal(body, &loginResp)

    if loginResp.Code != 200 {
        return "", fmt.Errorf("login failed: %s", loginResp.Message)
    }

    return loginResp.Data.Token, nil
}

func makeAuthenticatedRequest(url, token string) (*http.Response, error) {
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    client := &http.Client{}
    return client.Do(req)
}

func main() {
    baseURL := "http://localhost:8080"
    
    // 登录
    token, err := login(baseURL, "admin", "123456")
    if err != nil {
        fmt.Printf("Login error: %v\n", err)
        return
    }
    
    fmt.Printf("Token: %s\n", token)
    
    // 使用 Token 访问 API
    resp, err := makeAuthenticatedRequest(
        baseURL+"/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces",
        token,
    )
    if err != nil {
        fmt.Printf("Request error: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Printf("Status: %s\n", resp.Status)
}
```

### Python 客户端示例

```python
import requests
import json

BASE_URL = "http://localhost:8080"

def login(username, password):
    """登录并获取 Token"""
    url = f"{BASE_URL}/kapis/auth.kubespark.io/v1/login"
    payload = {
        "username": username,
        "password": password
    }
    
    response = requests.post(url, json=payload)
    response.raise_for_status()
    
    data = response.json()
    if data.get("code") != 200:
        raise Exception(f"Login failed: {data.get('message')}")
    
    return data["data"]["token"]

def make_authenticated_request(url, token):
    """使用 Token 发送认证请求"""
    headers = {
        "Authorization": f"Bearer {token}"
    }
    
    response = requests.get(url, headers=headers)
    response.raise_for_status()
    
    return response.json()

# 使用示例
if __name__ == "__main__":
    # 登录
    token = login("admin", "123456")
    print(f"Token: {token}")
    
    # 访问受保护的 API
    namespaces = make_authenticated_request(
        f"{BASE_URL}/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces",
        token
    )
    print(f"Namespaces: {json.dumps(namespaces, indent=2)}")
```

### JavaScript/Node.js 客户端示例

```javascript
const axios = require('axios');

const BASE_URL = 'http://localhost:8080';

async function login(username, password) {
    const response = await axios.post(
        `${BASE_URL}/kapis/auth.kubespark.io/v1/login`,
        { username, password }
    );
    
    if (response.data.code !== 200) {
        throw new Error(`Login failed: ${response.data.message}`);
    }
    
    return response.data.data.token;
}

async function makeAuthenticatedRequest(url, token) {
    const response = await axios.get(url, {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    });
    
    return response.data;
}

// 使用示例
(async () => {
    try {
        // 登录
        const token = await login('admin', '123456');
        console.log('Token:', token);
        
        // 访问受保护的 API
        const namespaces = await makeAuthenticatedRequest(
            `${BASE_URL}/kapis/resources.kubespark.io/v1alpha1/resources/core/v1/namespaces`,
            token
        );
        console.log('Namespaces:', JSON.stringify(namespaces, null, 2));
    } catch (error) {
        console.error('Error:', error.message);
    }
})();
```

## 安全特性

### 已实现的安全机制

KubeSpark 实现了多层安全防护机制，确保系统的安全性：

#### 1. JWT 签名验证
- 使用 HMAC-SHA256 算法对 Token 进行签名
- 确保 Token 的完整性和真实性
- 防止 Token 被篡改或伪造

#### 2. 密码验证器机制
- Token 中包含密码验证器：`HMAC-SHA256(JWT_SECRET, password)`
- **防止 Token 伪造**：即使攻击者获得了 JWT Secret 和用户名，也无法生成有效 Token
- **防止离线暴力破解**：攻击者即使看到 Token 中的验证器，也无法离线破解密码，因为需要 JWT Secret 来验证密码猜测

#### 3. 三层验证机制
Token 验证时执行三层检查：
1. **签名和过期验证**：标准 JWT 验证
2. **用户名验证**：确保 Token 中的用户名与配置匹配
3. **密码验证器验证**：确保 Token 中的密码验证器与配置密码匹配

#### 4. 统一错误信息
- 登录失败时返回统一的错误信息："Invalid username or password"
- 防止通过错误信息泄露用户名是否存在等敏感信息

#### 5. Token 过期机制
- Token 有效期为 24 小时
- 过期后需要重新登录获取新 Token

### 安全建议

#### 生产环境配置

1. **设置 JWT Secret**: 在生产环境中，**必须**通过 `KUBESPARK_JWT_SECRET` 环境变量设置强随机密钥，不要使用默认值
   ```bash
   export KUBESPARK_JWT_SECRET=$(openssl rand -base64 32)
   ```

2. **使用 HTTPS**: 在生产环境中使用 HTTPS 传输，防止 Token 被截获
   - 建议使用反向代理（如 Nginx、Traefik）处理 TLS
   - 或在应用层配置 TLS 证书

3. **设置强密码**: 通过环境变量设置强密码，避免使用默认密码 `123456`
   ```bash
   export KUBESPARK_PASSWORD=$(openssl rand -base64 16)
   ```

4. **Token 过期时间**: 根据安全需求调整 Token 过期时间（当前为 24 小时）
   - 可在代码中修改 `TokenExpiration` 常量

5. **保护 JWT Secret**: JWT Secret 是系统安全的关键，不要将其提交到代码仓库或日志中
   - 使用 Kubernetes Secret 或环境变量管理
   - 定期轮换 JWT Secret（注意：轮换后所有现有 Token 将失效）

6. **网络安全**: 
   - 使用防火墙限制访问
   - 考虑实现 IP 白名单
   - 在生产环境中禁用或限制 Swagger UI 访问

### 环境变量管理

建议使用以下方式管理环境变量：

**Docker Compose**:
```yaml
services:
  kubespark:
    environment:
      - KUBESPARK_USERNAME=admin
      - KUBESPARK_PASSWORD=${KUBESPARK_PASSWORD}
      - KUBESPARK_JWT_SECRET=${KUBESPARK_JWT_SECRET}  # 从环境变量或 secrets 文件读取
```

**Kubernetes Secret**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubespark-auth
type: Opaque
stringData:
  username: admin
  password: <strong-password>
  jwt-secret: <strong-random-jwt-secret>  # 使用强随机密钥
```

然后在 Deployment 中引用：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubespark
spec:
  template:
    spec:
      containers:
      - name: kubespark
        env:
        - name: KUBESPARK_USERNAME
          valueFrom:
            secretKeyRef:
              name: kubespark-auth
              key: username
        - name: KUBESPARK_PASSWORD
          valueFrom:
            secretKeyRef:
              name: kubespark-auth
              key: password
        - name: KUBESPARK_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: kubespark-auth
              key: jwt-secret
```

## 总结

KubeSpark 的认证系统采用 JWT Token 机制，实现了多层安全防护：

### 认证流程

1. **登录**: 客户端使用用户名和密码登录，获取 JWT Token
2. **认证**: 后续请求在 `Authorization` 头中携带 Token
3. **验证**: 服务器执行三层验证（签名、用户名、密码验证器）
4. **授权**: 验证通过后，请求继续处理

### 安全特性总结

✅ **JWT 签名验证** - 使用 HMAC-SHA256 确保 Token 完整性  
✅ **密码验证器机制** - 防止 Token 伪造和离线暴力破解  
✅ **三层验证机制** - 签名、用户名、密码验证器三重检查  
✅ **统一错误信息** - 防止信息泄露  
✅ **Token 过期机制** - 24 小时有效期，自动失效  
✅ **环境变量配置** - 灵活的凭据管理  

### 安全优势

- **防止 Token 伪造**：即使攻击者获得 JWT Secret 和用户名，也无法生成有效 Token
- **防止离线暴力破解**：密码验证器使用 HMAC，需要 JWT Secret 才能验证密码猜测
- **防止 Token 重用**：密码更改后，旧 Token 立即失效
- **防止信息泄露**：统一的错误信息，不泄露用户名是否存在等信息

通过环境变量可以灵活配置登录凭据，同时保持代码的简洁性。在生产环境中，请务必遵循安全建议，确保系统的安全性。

