package llm

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ender507/llm-gateway/utils"
)

type BackendManager struct {
	backends        map[string]*Backend
	mu              sync.RWMutex
	httpClient      *http.Client
	modelBackendMap map[string][]*Backend
	sessionMap      map[string]*sessionBinding
}

type sessionBinding struct {
	backend    *Backend
	lastAccess time.Time
}

var manager *BackendManager

func InitBackendManager() *BackendManager {
	manager = &BackendManager{
		backends: make(map[string]*Backend),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		sessionMap: make(map[string]*sessionBinding),
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
		Endpoint:       endpoint,
		status:         StatusUnhealthy, // 初始标记不健康，等健康检查确认
		maxConcurrency: utils.MaxBackendConcurrency,
		sem:            make(chan struct{}, utils.MaxBackendConcurrency),
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
	b.SetStatus(newStatus)
	b.SetModels(loadedModels)
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

// GetBackendBySession 根据 sessionID 选后端服务器
// 用于实现会话亲和，相同会话更倾向于选择相同后端
func (m *BackendManager) GetBackendBySession(sessionID string, candidates []*Backend) (*Backend, bool) {
	if len(candidates) == 0 {
		return nil, false
	}
	if sessionID == "" {
		return PickLeastConcurrent(candidates), false
	}

	m.mu.RLock()
	binding, exists := m.sessionMap[sessionID]
	m.mu.RUnlock()

	if exists {
		for _, cand := range candidates {
			if cand == binding.backend {
				m.mu.Lock()
				binding.lastAccess = time.Now()
				m.mu.Unlock()
				return binding.backend, true
			}
		}
		// 绑定的后端不在候选里了，删掉旧绑定
		m.mu.Lock()
		delete(m.sessionMap, sessionID)
		m.mu.Unlock()
	}

	// 新会话或会话超时失效
	selected := PickLeastConcurrent(candidates)
	m.mu.Lock()
	m.sessionMap[sessionID] = &sessionBinding{
		backend:    selected,
		lastAccess: time.Now(),
	}
	m.mu.Unlock()
	return selected, false
}

func (m *BackendManager) StartSessionCleaner(interval time.Duration, ttl time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			m.mu.Lock()
			for sid, b := range m.sessionMap {
				if now.Sub(b.lastAccess) > ttl {
					delete(m.sessionMap, sid)
				}
			}
			m.mu.Unlock()
		}
	}()
}
