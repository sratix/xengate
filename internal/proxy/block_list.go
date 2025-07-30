package proxy

import "sync"

// در بالای فایل‌ها
type IPBlocklist struct {
	mu      sync.RWMutex
	blocked map[string]bool
}

func NewIPBlocklist() *IPBlocklist {
	return &IPBlocklist{
		blocked: make(map[string]bool),
	}
}

func (bl *IPBlocklist) Add(ip string) {
	bl.mu.Lock()
	bl.blocked[ip] = true
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
	return bl.blocked[ip]
}
