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
