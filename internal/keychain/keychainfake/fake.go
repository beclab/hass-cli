// Package keychainfake provides an in-memory keychain.KeychainAccess for tests,
// so token-store logic can be exercised without touching the real OS keychain.
package keychainfake

import (
	"sync"

	"github.com/beclab/hass-cli/internal/keychain"
)

// Fake is an in-memory KeychainAccess keyed by "service\x00account".
type Fake struct {
	mu   sync.Mutex
	data map[string]string
}

// New returns an empty in-memory keychain.
func New() *Fake {
	return &Fake{data: map[string]string{}}
}

func key(service, account string) string { return service + "\x00" + account }

// Get returns ("", nil) when the entry is absent, matching keychain.Get.
func (f *Fake) Get(service, account string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data[key(service, account)], nil
}

func (f *Fake) Set(service, account, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key(service, account)] = value
	return nil
}

func (f *Fake) Remove(service, account string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key(service, account))
	return nil
}

var _ keychain.KeychainAccess = (*Fake)(nil)
