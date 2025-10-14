# Poorman FaaS - 轻量级函数即服务平台

## 项目概述

Poorman FaaS 是一个基于 Kubernetes 的轻量级函数即服务（Function as a Service）平台，专门为 Python 函数提供无服务器计算能力。该项目采用 Go 语言开发，利用 Kubernetes 的原生能力实现函数的自动部署、扩缩容和生命周期管理。

## 核心特性

- 🚀 **快速部署**: 支持 Python 函数的快速部署和运行
- 🔄 **自动扩缩容**: 基于请求量自动扩缩容，支持缩放到零
- 📦 **PEP 723 兼容**: 支持 Python 脚本的依赖管理标准
- 🛠️ **Kubernetes 原生**: 完全基于 Kubernetes 构建，无需额外基础设施
- ⚡ **轻量级**: 最小化资源占用，适合边缘计算场景
- 🔧 **简单易用**: 提供 RESTful API 进行函数管理

## 项目架构

### 核心组件

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Gateway API   │    │   Admin API     │    │   Reaper        │
│   (代理路由)     │    │   (函数管理)     │    │   (资源清理)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Kubernetes     │
                    │   (函数运行时)    │
                    └─────────────────┘
```

### 主要模块

1. **Gateway API** (`/gateway/{serviceName}/*`): 代理用户请求到具体的函数服务
2. **Admin API** (`/admin/python`): 管理函数的部署、更新和删除
3. **Reaper**: 自动清理长时间未使用的函数资源
4. **Helm Chart**: 生成和管理 Kubernetes 资源

## 技术栈

- **后端**: Go 1.24.6
- **Web 框架**: Chi Router
- **容器运行时**: UV (Python 包管理器)
- **编排平台**: Kubernetes
- **依赖管理**: PEP 723 标准
- **构建工具**: Just (任务运行器)

## 快速开始

### 环境要求

- Go 1.24.6+
- Kubernetes 集群 (支持 LoadBalancer)
- Docker/Podman
- Just 任务运行器

### 安装部署

1. **克隆项目**
```bash
git clone <repository-url>
cd poorman-faas
```

2. **构建和部署**
```bash
# 构建 FaaS 网关
just build-faas

# 部署到 Kubernetes
just deploy-faas
```

3. **验证部署**
```bash
# 检查服务状态
kubectl get pods -n faas
kubectl get svc -n faas
```

### 使用示例

#### 1. 创建 Python 函数

创建一个符合 PEP 723 标准的 Python 脚本：

```python
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastapi",
#     "pydantic",
#     "uvicorn",
# ]
# ///

import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class EchoInput(BaseModel):
    message: str

class EchoOutput(BaseModel):
    received_message: str

@app.post("/echo/")
async def echo_message(data: EchoInput):
    return EchoOutput(received_message=data.message)

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

#### 2. 部署函数

```bash
# 获取网关地址
LB_IP=$(kubectl -n faas get svc/faas-gateway -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

# 编码脚本和配置文件
SCRIPT=$(cat your-script.py | base64)
DOTFILE=$(cat your-config.ini | base64)

# 部署函数
curl -X POST "http://${LB_IP}:8080/admin/python" \
  -H "Content-Type: application/json" \
  -d '{
    "script": "'$SCRIPT'",
    "dot_file": "'$DOTFILE'",
    "option": {
      "user": "your-username",
      "replica": 1
    }
  }'
```

#### 3. 调用函数

```bash
# 调用函数
curl -X POST "http://${LB_IP}:8080/gateway/{serviceName}/echo/" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, FaaS!"}'
```

## 项目结构

```
poorman-faas/
├── cmd/                    # 可执行程序
│   ├── faas/              # 主网关服务
│   ├── echo/              # 示例函数
│   └── hello/             # 简单示例
├── pkg/                   # 核心包
│   ├── helm/              # Kubernetes 资源模板
│   ├── proxy/             # 反向代理
│   ├── reaper/            # 资源清理器
│   └── util/              # 工具函数
├── hack/                  # 部署配置
├── test/                  # 测试脚本
└── docs/                  # 文档
```

## 核心概念

### 1. 函数生命周期

1. **部署**: 通过 Admin API 上传 Python 脚本
2. **验证**: 检查 PEP 723 合规性和依赖
3. **创建**: 生成 Kubernetes 资源 (ConfigMap, Deployment, Service)
4. **运行**: 函数在 Pod 中执行
5. **代理**: Gateway 将请求转发到函数
6. **清理**: Reaper 自动清理未使用的函数

### 2. PEP 723 支持

项目支持 Python 的 PEP 723 标准，允许在脚本中直接声明依赖：

```python
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastapi",
#     "pydantic",
#     "uvicorn",
# ]
# ///
```

### 3. 资源配置

每个函数会创建以下 Kubernetes 资源：
- **ConfigMap**: 存储 Python 脚本
- **Deployment**: 管理 Pod 生命周期
- **Service**: 提供网络访问

## 开发指南

### 本地开发

```bash
# 安装依赖
go mod tidy

# 运行代码检查
just lint

# 本地运行
just dev faas
```

### 测试

```bash
# 运行测试脚本
./test/upload-code.sh
./test/health-check.sh
```

### 构建

```bash
# 构建二进制文件
just build faas

# 构建容器镜像
just build-faas
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LOG_LEVEL` | debug | 日志级别 |
| `REAPER_POLL_EVERY` | 10s | 清理器轮询间隔 |
| `REAPER_TIME_TO_LIVE` | 30s | 函数生存时间 |
| `K8S_NAMESPACE` | faas | Kubernetes 命名空间 |
| `PORT` | 8080 | 服务端口 |
| `GATEWAY_PATH_PREFIX` | /gateway | 网关路径前缀 |

### Kubernetes 配置

项目使用以下 Kubernetes 资源：
- **Namespace**: `faas`
- **ServiceAccount**: `faas-gateway-sa`
- **RBAC**: 必要的权限配置

## 故障排除

### 常见问题

1. **LoadBalancer IP 获取失败**
   - 检查 MetalLB 或云提供商 LoadBalancer 配置
   - 使用 `kubectl get svc -n faas` 检查服务状态

2. **函数部署失败**
   - 检查 PEP 723 格式是否正确
   - 验证依赖包是否可用
   - 查看 Pod 日志: `kubectl logs -n faas <pod-name>`

3. **代理请求失败**
   - 检查函数服务是否正常运行
   - 验证网络策略配置
   - 查看网关日志

### 调试命令

```bash
# 查看所有资源
kubectl get all -n faas

# 查看网关日志
kubectl logs -n faas deployment/faas-gateway

# 查看函数 Pod 日志
kubectl logs -n faas -l app=<function-name>

# 检查服务端点
kubectl get endpoints -n faas
```

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。

## 相关链接

- [PEP 723 规范](https://peps.python.org/pep-0723/)
- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Chi Router](https://github.com/go-chi/chi)
- [UV Python 包管理器](https://github.com/astral-sh/uv)


