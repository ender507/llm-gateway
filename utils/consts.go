package utils

import "time"

const (
	OllamaDomain1 = "http://127.0.0.1:11434"
	OllamaDomain2 = "http://127.0.0.1:11435"

	HandleRequestTimeout       = 20 * time.Second // 单次非流式请求响应超时
	HandleRequestStreamTimeout = 3 * time.Minute  // 单次流式请求响应超时
	BackendHealthCheckDuration = 3 * time.Second  // 后端服务器健康检查周期
	SessionCleanDuration       = 5 * time.Second  // 清理会话亲和任务的执行周期
	SessionAffinityTTL         = 20 * time.Second // 会话亲和过期时间

	TraceID = "trace_id"

	MaxBackendConcurrency = 2               // 每个后端最大并发数
	QueueTimeout          = 5 * time.Second // 并发数达到最大值时，新请求排队最长等待时间
)
