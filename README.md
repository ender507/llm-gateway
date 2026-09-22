# llm-gateway
我的llm实践练习项目。以ollama为模型框架入口，实现llm模型网关。

# 项目特点
- 接口遵循 OpenAI API 规范（参考: [官网](https://platform.openai.com/docs/api-reference) / [中文翻译](https://openaidoc.kaimingwan.com/)）

# 目录结构
- main.go: 请求入口，注册服务路由
- utils: 基础能力工具包
  - consts: 基本常量
  - logger: 提供日志实例
- handler: 请求处理
- gateway: 实现路由网关的核心逻辑   
