package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 请求总计数，label: model,status（200/503/500...）
	reqTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_requests_total",
			Help: "Total requests received by llm gateway",
		},
		[]string{"model", "status"},
	)

	// TTFT 直方图：首token耗时（秒）
	ttftHist = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_gateway_ttft_seconds",
			Help:    "Time to first token in seconds",
			Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 3, 5, 8, 15},
		},
		[]string{"model"},
	)

	// 每个后端活跃并发Gauge
	backendActiveConcurrency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llm_gateway_active_concurrency",
			Help: "Active concurrency per ollama backend",
		},
		[]string{"endpoint"},
	)

	// 排队超时计数
	queueTimeoutTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_queue_timeout_total",
			Help: "Count of requests failed due to queue timeout",
		},
		[]string{"endpoint"},
	)

	// 请求 ollama 的重试次数
	gatewayUpstreamRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_upstream_retry_total",
			Help: "Total count of upstream retry events, labeled by model, backend and outcome",
		},
		[]string{"model", "backend", "detail"},
	)

	// 熔断器状态 Gauge：0=closed 1=open 2=half_open
	circuitStateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llm_gateway_circuit_state",
			Help: "Circuit breaker state per backend: 0=closed 1=open 2=half_open",
		},
		[]string{"endpoint"},
	)

	// 熔断触发计数
	circuitOpenTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_circuit_open_total",
			Help: "Total count of circuit breaker opening events per backend",
		},
		[]string{"endpoint"},
	)
)

// RecordRequest 记录请求
func RecordRequest(model, status string) {
	reqTotal.WithLabelValues(model, status).Inc()
}

// RecordTTFT 记录首token耗时
func RecordTTFT(model string, sec float64) {
	ttftHist.WithLabelValues(model).Observe(sec)
}

// SetBackendConcurrency 更新后端并发Gauge
func SetBackendConcurrency(endpoint string, val int) {
	backendActiveConcurrency.WithLabelValues(endpoint).Set(float64(val))
}

// IncQueueTimeout 排队超时+1
func IncQueueTimeout(endpoint string) {
	queueTimeoutTotal.WithLabelValues(endpoint).Inc()
}

// IncGatewayUpstreamRetryTotal 请求重试次数统计
func IncGatewayUpstreamRetryTotal(model, backend, detail string) {
	gatewayUpstreamRetryTotal.WithLabelValues(model, backend, detail).Inc()
}

// SetCircuitState 熔断状态变化
func SetCircuitState(endpoint string, state float64) {
	circuitStateGauge.WithLabelValues(endpoint).Set(state)
}

// IncCircuitOpen 后端服务器触发熔断
func IncCircuitOpen(endpoint string) {
	circuitOpenTotal.WithLabelValues(endpoint).Inc()
}
