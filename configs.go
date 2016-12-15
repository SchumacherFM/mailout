package mailout

import "sync"

// todo

// configEndpoints
type configEndpoints struct {
	sync.RWMutex
	configs map[string]*config
}

func newConfigEndpoints() *configEndpoints {
	return &configEndpoints{
		configs: make(map[string]*config),
	}
}

func (ce *configEndpoints) byEndpoint(url string) *config {
	ce.RLock()
	defer ce.RUnlock()
	return ce.configs[url]
}

func (ce *configEndpoints) addEndpoint(url string, c *config) {
	ce.Lock()
	defer ce.Unlock()
	ce.configs[url] = c
}
