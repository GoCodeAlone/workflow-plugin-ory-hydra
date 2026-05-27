package internal

import (
	"sync"

	hydra "github.com/ory/hydra-client-go/v2"
)

type OryHydraClient struct {
	Admin    *hydra.APIClient
	AdminURL string
}

var (
	clientMu       sync.RWMutex
	clientRegistry = map[string]*OryHydraClient{}
)

func RegisterClient(name string, client *OryHydraClient) {
	clientMu.Lock()
	defer clientMu.Unlock()
	clientRegistry[name] = client
}

func GetClient(name string) (*OryHydraClient, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, ok := clientRegistry[name]
	return client, ok
}

func UnregisterClient(name string) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(clientRegistry, name)
}
