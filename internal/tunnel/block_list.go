// block_list.go
package tunnel

import (
	"sync"
	"time"
)

type IPBlocklist struct {
	mu      sync.RWMutex
	blocked map[string]time.Time
}

func NewIPBlocklist() *IPBlocklist {
	return &IPBlocklist{
		blocked: make(map[string]time.Time),
	}
}

func (bl *IPBlocklist) Add(ip string) {
	bl.mu.Lock()
	bl.blocked[ip] = time.Now()
	bl.mu.Unlock()
}

func (bl *IPBlocklist) Remove(ip string) {
	bl.mu.Lock()
	delete(bl.blocked, ip)
	bl.mu.Unlock()
}

func (bl *IPBlocklist) IsBlocked(ip string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	_, exists := bl.blocked[ip]
	return exists
}

func (bl *IPBlocklist) GetAll() map[string]time.Time {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	result := make(map[string]time.Time, len(bl.blocked))
	for ip, timestamp := range bl.blocked {
		result[ip] = timestamp
	}
	return result
}
