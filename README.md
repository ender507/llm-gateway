# llm-gateway
我的llm实践练习项目。基于 Go + Gin 实现的轻量级 LLM 推理网关，后端对接 Ollama，对外暴露 OpenAI 兼容 API。

## 核心功能

- **OpenAI 兼容 API**：`/v1/chat/completions`（SSE 流式）、`/v1/models`，可直接用 OpenAI SDK 接入
- **多后端管理**：注册多个 Ollama 实例，定时探活，自动摘除故障节点
- **模型反向索引**：健康检查时采集各节点已加载模型，构建 `model → [backends]` 索引，按模型名路由
- **会话亲和**：通过 `X-Session-Id` 请求头，将同一会话固定到同一后端，复用节点 KV Cache
- **最少并发负载均衡**：新会话自动选择当前并发最低的后端
- **流式/非流式分离超时**：流式请求使用长超时，非流式使用短超时

# 目录结构
- main.go: 程序入口，初始化组件、注册服务路由
- utils: 基础能力工具包
  - consts: 基本常量
  - logger: 提供日志实例
- handler: 请求处理
  - chat_completions.go: /v1/chat/completions 处理逻辑
  - list_models.go: /v1/models 聚合模型列表
  - utils.go: handler层通用工具方法
- internal: 
  - llm: 后端llm实例管理
    - backend.go: 后端服务器实例，维护并发计数、状态等
    - backend_manager.go: 后端管理器，探活、索引构建、会话亲和等


# 快速开始

## 前置条件

- Go 1.21+
- Ollama（本地或远程，至少一个实例）

## 启动

```bash
# 1. 启动 Ollama 实例（至少一个）
set OLLAMA_HOST="127.0.0.1:11434" # 设置HOST环境变量
ollama serve

# 2. 保证 Ollama 至少要有一个可用模型
ollama pull qwen:1.5b # pull下载一个模型，如qwen:1.5b

# 3. 启动本服务
go run main.go
```

## API 使用示例
获取模型列表:
```bash
curl http://127.0.0.1:8080/v1/models
```

流式对话（带 session，开启会话亲和）：
```bash
curl --location 'http://localhost:8080/v1/chat/completions' \
--header 'X-Session-Id: 1' \
--header 'Content-Type: application/json' \
--data '{
    "model": "qwen:1.5b",
    "messages": [{"role":"user","content":""}],
    "stream": true
  }'
```

非流式对话（不带 session，自动负载均衡）：
```bash
curl --location 'http://localhost:8080/v1/chat/completions' \
--header 'Content-Type: application/json' \
--data '{
    "model": "qwen:1.5b",
    "messages": [{"role":"user","content":"简单介绍Go语言"}],
    "stream": true
  }'
```