package llm

import (
	"sync"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

type Backend struct {
	Endpoint          string
	status            Status
	models            []string
	activeConcurrency int
	mu                sync.RWMutex
}

func (b *Backend) Models() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make([]string, len(b.models))
	copy(cp, b.models)
	return cp
}

func (b *Backend) SetModels(modelNames []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.models = modelNames
}

func (b *Backend) IncrConcurrency() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeConcurrency++
}

func (b *Backend) DecrConcurrency() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.activeConcurrency > 0 {
		b.activeConcurrency--
	}
}

func (b *Backend) ActiveConcurrency() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeConcurrency
}

func (b *Backend) Status() Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

func (b *Backend) SetStatus(s Status) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.status = s
}

func PickLeastConcurrent(candidates []*Backend) *Backend {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	minCnt := best.ActiveConcurrency()

	for _, b := range candidates[1:] {
		cnt := b.ActiveConcurrency()
		if cnt < minCnt {
			minCnt = cnt
			best = b
		}
	}
	return best
}
