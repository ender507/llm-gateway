# llm-gateway
我的llm实践练习项目。以ollama为模型框架入口，实现llm模型网关。

# 项目特点与实现功能
- 接口遵循 OpenAI API 规范（参考: [官网](https://platform.openai.com/docs/api-reference) / [中文翻译](https://openaidoc.kaimingwan.com/)）
- 对后端服务器探活，自动摘除异常后端，恢复正常后端
- 通过sessionID控制会话亲和，相同会话优先调度到同一后端服务器

# 目录结构
- main.go: 请求入口，注册服务路由
- utils: 基础能力工具包
  - consts: 基本常量
  - logger: 提供日志实例
- handler: 请求处理
- internal: 
  - llm: 后端llm实例管理
