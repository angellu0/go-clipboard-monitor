package internal

import "sync"

type Metrics struct {
	TotalHits   int
	RuleHits    map[string]int
	SessionStart string
	mu          sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		RuleHits: make(map[string]int),
	}
}

func (m *Metrics) Register(rules []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalHits += len(rules)
	for _, r := range rules {
		m.RuleHits[r]++
	}
}

func (m *Metrics) GetTotalHits() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.TotalHits
}

func (m *Metrics) GetRuleHits() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hits := make(map[string]int, len(m.RuleHits))
	for k, v := range m.RuleHits {
		hits[k] = v
	}
	return hits
}
