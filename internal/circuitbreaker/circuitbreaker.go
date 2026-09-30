package circuitbreaker

import (
	"sync"
	"time"

	"github.com/ender507/llm-gateway/internal/metrics"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

type Config struct {
	FailureThreshold  float64
	WindowDuration    time.Duration
	CooldownDuration  time.Duration
	HalfOpenMaxReq    int
	MinRequestsToOpen int
}

type CircuitBreaker struct {
	cfg      Config
	endpoint string
	mu       sync.Mutex

	state         State
	successCnt    int
	failureCnt    int
	windowStart   time.Time
	openAt        time.Time
	probeInFlight int // 半熔断状态下正在处理的请求数
	probeSuccess  int // 半熔断状态下已经成功的请求数
	probeFailure  int // 半熔断状态下已经失败的请求数
}

func New(cfg Config, endpoint string) *CircuitBreaker {
	cb := &CircuitBreaker{
		cfg:         cfg,
		endpoint:    endpoint,
		state:       StateClosed,
		windowStart: time.Now(),
	}
	metrics.SetCircuitState(endpoint, 0)
	return cb
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.openAt) > cb.cfg.CooldownDuration {
			// 熔断到期，进入半熔断状态
			cb.state = StateHalfOpen
			cb.probeInFlight, cb.probeSuccess, cb.probeFailure = 0, 0, 0
			cb.updateCircuitState()
		}
		return false
	case StateHalfOpen:
		reqCount := cb.probeInFlight + cb.probeSuccess + cb.probeFailure
		return reqCount < cb.cfg.HalfOpenMaxReq
	}
	return true
}

func (cb *CircuitBreaker) BeginCall() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == StateHalfOpen {
		reqCount := cb.probeInFlight + cb.probeSuccess + cb.probeFailure
		if reqCount >= cb.cfg.HalfOpenMaxReq {
			return false
		}
		cb.probeInFlight++
	}
	return true
}

func (cb *CircuitBreaker) Report(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	switch cb.state {
	case StateClosed:
		if now.Sub(cb.windowStart) > cb.cfg.WindowDuration {
			cb.resetWindow(now)
		}
		if success {
			cb.successCnt++
		} else {
			cb.failureCnt++
		}
		total := cb.successCnt + cb.failureCnt
		if total >= cb.cfg.MinRequestsToOpen && float64(cb.failureCnt)/float64(total) >= cb.cfg.FailureThreshold {
			cb.state = StateOpen
			cb.openAt = now
			metrics.IncCircuitOpen(cb.endpoint)
			cb.updateCircuitState()
		}
	case StateHalfOpen:
		if success {
			cb.probeSuccess++
		} else {
			cb.probeFailure++
		}
		if cb.probeSuccess+cb.probeFailure >= cb.cfg.HalfOpenMaxReq {
			if cb.probeFailure > 0 {
				cb.state = StateOpen
				cb.openAt = now
				metrics.IncCircuitOpen(cb.endpoint)
			} else {
				cb.state = StateClosed
				cb.resetWindow(now)
			}
			cb.updateCircuitState()
		}
	}
}

func (cb *CircuitBreaker) ReleaseProbe() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == StateHalfOpen && cb.probeInFlight > 0 {
		cb.probeInFlight--
	}
}

func (cb *CircuitBreaker) resetWindow(now time.Time) {
	cb.successCnt = 0
	cb.failureCnt = 0
	cb.windowStart = now
}

func (cb *CircuitBreaker) updateCircuitState() {
	switch cb.state {
	case StateClosed:
		metrics.SetCircuitState(cb.endpoint, 0)
	case StateOpen:
		metrics.SetCircuitState(cb.endpoint, 1)
	case StateHalfOpen:
		metrics.SetCircuitState(cb.endpoint, 2)
	}
}
