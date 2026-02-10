package llm

import (
	"sync"
	"time"
)

type Metrics struct {
	mu              sync.RWMutex
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedReqs      int64
	TotalTokens     int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	LastRequestTime time.Time
	ErrorCounts     map[string]int64
	ProviderMetrics map[string]*ProviderMetrics
}

type ProviderMetrics struct {
	TotalRequests  int64
	SuccessfulReqs int64
	FailedReqs     int64
	TotalTokens    int64
	TotalLatency   time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
}

type Monitor struct {
	metrics *Metrics
}

func NewMonitor() *Monitor {
	return &Monitor{
		metrics: &Metrics{
			MinLatency:      time.Hour,
			MaxLatency:      0,
			ErrorCounts:     make(map[string]int64),
			ProviderMetrics: make(map[string]*ProviderMetrics),
		},
	}
}

func (m *Monitor) RecordRequest(provider string, latency time.Duration, tokens int, err error) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalRequests++
	m.metrics.LastRequestTime = time.Now()
	m.metrics.TotalTokens += int64(tokens)
	m.metrics.TotalLatency += latency

	if latency < m.metrics.MinLatency {
		m.metrics.MinLatency = latency
	}
	if latency > m.metrics.MaxLatency {
		m.metrics.MaxLatency = latency
	}

	if err == nil {
		m.metrics.SuccessfulReqs++
	} else {
		m.metrics.FailedReqs++
		errorType := "unknown"
		if llmErr, ok := err.(*LLMError); ok {
			errorType = llmErr.Code
		}
		m.metrics.ErrorCounts[errorType]++
	}

	if _, exists := m.metrics.ProviderMetrics[provider]; !exists {
		m.metrics.ProviderMetrics[provider] = &ProviderMetrics{
			MinLatency: time.Hour,
			MaxLatency: 0,
		}
	}

	pm := m.metrics.ProviderMetrics[provider]
	pm.TotalRequests++
	pm.TotalTokens += int64(tokens)
	pm.TotalLatency += latency

	if latency < pm.MinLatency {
		pm.MinLatency = latency
	}
	if latency > pm.MaxLatency {
		pm.MaxLatency = latency
	}

	if err == nil {
		pm.SuccessfulReqs++
	} else {
		pm.FailedReqs++
	}
}

func (m *Monitor) GetMetrics() *Metrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	copy := &Metrics{
		TotalRequests:   m.metrics.TotalRequests,
		SuccessfulReqs:  m.metrics.SuccessfulReqs,
		FailedReqs:      m.metrics.FailedReqs,
		TotalTokens:     m.metrics.TotalTokens,
		TotalLatency:    m.metrics.TotalLatency,
		MinLatency:      m.metrics.MinLatency,
		MaxLatency:      m.metrics.MaxLatency,
		LastRequestTime: m.metrics.LastRequestTime,
		ErrorCounts:     make(map[string]int64),
		ProviderMetrics: make(map[string]*ProviderMetrics),
	}

	for k, v := range m.metrics.ErrorCounts {
		copy.ErrorCounts[k] = v
	}

	for k, v := range m.metrics.ProviderMetrics {
		copy.ProviderMetrics[k] = &ProviderMetrics{
			TotalRequests:  v.TotalRequests,
			SuccessfulReqs: v.SuccessfulReqs,
			FailedReqs:     v.FailedReqs,
			TotalTokens:    v.TotalTokens,
			TotalLatency:   v.TotalLatency,
			MinLatency:     v.MinLatency,
			MaxLatency:     v.MaxLatency,
		}
	}

	return copy
}

func (m *Monitor) GetAverageLatency() time.Duration {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	if m.metrics.TotalRequests == 0 {
		return 0
	}
	return m.metrics.TotalLatency / time.Duration(m.metrics.TotalRequests)
}

func (m *Monitor) GetSuccessRate() float64 {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	if m.metrics.TotalRequests == 0 {
		return 0
	}
	return float64(m.metrics.SuccessfulReqs) / float64(m.metrics.TotalRequests) * 100
}

func (m *Monitor) Reset() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalRequests = 0
	m.metrics.SuccessfulReqs = 0
	m.metrics.FailedReqs = 0
	m.metrics.TotalTokens = 0
	m.metrics.TotalLatency = 0
	m.metrics.MinLatency = time.Hour
	m.metrics.MaxLatency = 0
	m.metrics.LastRequestTime = time.Time{}
	m.metrics.ErrorCounts = make(map[string]int64)
	m.metrics.ProviderMetrics = make(map[string]*ProviderMetrics)
}
