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

	MaxRetries     = 3                      // 请求ollama最多重试次数
	InitialBackoff = 200 * time.Millisecond // 初始退避
	MaxBackoff     = 2 * time.Second        // 退避上限

	CircuitBreakerFailureThreshold  = 0.3              // 熔断阈值，失败请求占比超过该值则触发熔断
	CircuitBreakerWindowDuration    = 10 * time.Second // 检查熔断的周期
	CircuitBreakerCooldownDuration  = 5 * time.Second  // 熔断后恢复的最短等待时长
	CircuitBreakerHalfOpenMaxReq    = 3                // 半熔断的最多请求数量
	CircuitBreakerMinRequestsToOpen = 5                // 判断熔断的最少请求数量，防止新周期少量失败导致熔断
)
