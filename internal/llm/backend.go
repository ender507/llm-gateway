package llm

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ender507/llm-gateway/utils"
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

type BackendManager struct {
	backends        map[string]*Backend
	mu              sync.RWMutex
	httpClient      *http.Client
	modelBackendMap map[string][]*Backend
}

var manager *BackendManager

func InitBackendManager() *BackendManager {
	manager = &BackendManager{
		backends: make(map[string]*Backend),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	return manager
}

func GetBackendManager() *BackendManager {
	return manager
}

func (m *BackendManager) Register(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backends[endpoint] = &Backend{
		Endpoint: endpoint,
		status:   StatusUnhealthy, // 初始标记不健康，等健康检查确认
	}
}

func (m *BackendManager) Unregister(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.backends, endpoint)
}

func (m *BackendManager) GetHealthyBackends() []*Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Backend
	for _, b := range m.backends {
		if b.Status() == StatusHealthy {
			result = append(result, b)
		}
	}
	return result
}

func (m *BackendManager) PickOne() *Backend {
	healthy := m.GetHealthyBackends()
	if len(healthy) == 0 {
		return nil
	}
	return healthy[0]
}

type OllamaTagsResp struct {
	Models []OllamaModelItem `json:"models"`
}

type OllamaModelItem struct {
	Name string `json:"name"`
}

func (m *BackendManager) CheckHealth(b *Backend) {
	var loadedModels []string
	logger := utils.GetLogger()
	resp, err := m.httpClient.Get(b.Endpoint + "/api/tags")
	newStatus := StatusUnhealthy
	if err == nil && resp.StatusCode == http.StatusOK {
		newStatus = StatusHealthy
		tagsResp := OllamaTagsResp{}
		decodeErr := json.NewDecoder(resp.Body).Decode(&tagsResp)
		if decodeErr == nil {
			loadedModels = make([]string, 0, len(tagsResp.Models))
			for _, item := range tagsResp.Models {
				loadedModels = append(loadedModels, item.Name)
			}
		} else {
			logger.Errorw("failed to unmarshal response when cheking health", "endpoint", b.Endpoint)
		}
	}
	if resp != nil {
		resp.Body.Close()
	}
	logger.Infow("backend health update",
		"backend", b.Endpoint,
		"new_status", newStatus,
		"models", loadedModels,
	)
	m.mu.Lock()
	b.SetStatus(newStatus)
	b.SetModels(loadedModels)
	m.mu.Unlock()
}

func (m *BackendManager) StartHealthCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			m.mu.RLock()
			backends := make([]*Backend, 0, len(m.backends))
			for _, b := range m.backends {
				backends = append(backends, b)
			}
			m.mu.RUnlock()

			for _, b := range backends {
				m.CheckHealth(b)
			}
			utils.GetLogger().Infow("health check done",
				"total", len(backends),
				"healthy", len(m.GetHealthyBackends()),
			)
			m.rebuildModelBackendMap()
		}
	}()
}

func (m *BackendManager) rebuildModelBackendMap() {
	modelBackendMap := make(map[string][]*Backend)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.backends {
		if b.Status() != StatusHealthy {
			continue
		}
		models := b.Models()
		for _, modelName := range models {
			modelBackendMap[modelName] = append(modelBackendMap[modelName], b)
		}
	}
	m.modelBackendMap = modelBackendMap
}

func (m *BackendManager) ModelBackendMap() map[string][]*Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	modelBackendMap := make(map[string][]*Backend, len(m.modelBackendMap))
	for modelName, backendList := range m.modelBackendMap {
		modelBackendMap[modelName] = append([]*Backend{}, backendList...)
	}
	return modelBackendMap
}

func (m *BackendManager) GetBackendsByModel(modelName string) []*Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	backendList := make([]*Backend, len(m.modelBackendMap[modelName]))
	copy(backendList, m.modelBackendMap[modelName])
	return backendList
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
