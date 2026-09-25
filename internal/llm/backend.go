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
	mu                sync.Mutex
}

func (b *Backend) Models() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	b.activeConcurrency--
}

func (b *Backend) Status() Status {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status
}

func (b *Backend) SetStatus(s Status) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.status = s
}
