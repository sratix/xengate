// // time_manager.go
package proxy

// import (
// 	"sort"
// 	"sync"
// 	"time"
// )

// type AccessControl struct {
// 	mu            sync.RWMutex
// 	accessRecords map[string]*IPAccess
// 	masterIPs     map[string]bool
// 	dailyLimit    time.Duration // محدودیت زمانی روزانه
// }

// type IPAccess struct {
// 	totalTime     time.Duration // مجموع زمان استفاده شده
// 	lastAccess    time.Time     // آخرین دسترسی
// 	activeSession *time.Time    // زمان شروع سشن فعلی
// 	isBlocked     bool          // وضعیت مسدود بودن
// 	lastReset     time.Time     // آخرین ریست شمارنده روزانه
// }

// type IPStats struct {
// 	IP        string
// 	UsedTime  time.Duration
// 	IsBlocked bool
// 	IsMaster  bool
// }

// func (ac *AccessControl) GetAllStats() []IPStats {
// 	ac.mu.RLock()
// 	defer ac.mu.RUnlock()

// 	stats := make([]IPStats, 0, len(ac.accessRecords))

// 	// Add master IPs
// 	for ip := range ac.masterIPs {
// 		stats = append(stats, IPStats{
// 			IP:       ip,
// 			IsMaster: true,
// 		})
// 	}

// 	// Add regular IPs
// 	for ip, access := range ac.accessRecords {
// 		if !ac.masterIPs[ip] {
// 			stats = append(stats, IPStats{
// 				IP:        ip,
// 				UsedTime:  access.totalTime,
// 				IsBlocked: access.isBlocked,
// 				IsMaster:  false,
// 			})
// 		}
// 	}

// 	sort.Slice(stats, func(i, j int) bool {
// 		return stats[i].IP < stats[j].IP
// 	})

// 	return stats
// }

// func (ac *AccessControl) GetActiveIPCount() int {
// 	ac.mu.RLock()
// 	defer ac.mu.RUnlock()
// 	return len(ac.accessRecords) + len(ac.masterIPs)
// }

// func (ac *AccessControl) SetIPLimit(ip string, limit time.Duration) {
// 	ac.mu.Lock()
// 	defer ac.mu.Unlock()

// 	// Remove from master IPs if exists
// 	delete(ac.masterIPs, ip)

// 	// Create or update access record
// 	if _, exists := ac.accessRecords[ip]; !exists {
// 		ac.accessRecords[ip] = &IPAccess{
// 			lastReset: time.Now(),
// 		}
// 	}
// }

// func (ac *AccessControl) GetDefaultLimit() time.Duration {
// 	return ac.dailyLimit
// }

// func NewAccessControl(dailyLimit time.Duration) *AccessControl {
// 	ac := &AccessControl{
// 		accessRecords: make(map[string]*IPAccess),
// 		masterIPs:     make(map[string]bool),
// 		dailyLimit:    dailyLimit,
// 	}

// 	// گوش دادن به تغییر روز برای ریست کردن شمارنده‌ها
// 	go ac.dailyResetWorker()

// 	return ac
// }

// func (ac *AccessControl) dailyResetWorker() {
// 	for {
// 		now := time.Now()
// 		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
// 		time.Sleep(time.Until(nextDay))

// 		ac.mu.Lock()
// 		for ip, access := range ac.accessRecords {
// 			// اگر مستر نیست، ریست کن
// 			if !ac.masterIPs[ip] {
// 				access.totalTime = 0
// 				access.lastReset = nextDay
// 				access.isBlocked = false
// 			}
// 		}
// 		ac.mu.Unlock()
// 	}
// }

// func (ac *AccessControl) AddMasterIP(ip string) {
// 	ac.mu.Lock()
// 	defer ac.mu.Unlock()
// 	ac.masterIPs[ip] = true

// 	// اگر رکورد قبلی داره، پاکش کن
// 	delete(ac.accessRecords, ip)
// }

// func (ac *AccessControl) RemoveMasterIP(ip string) {
// 	ac.mu.Lock()
// 	defer ac.mu.Unlock()
// 	delete(ac.masterIPs, ip)
// }

// func (ac *AccessControl) IsMasterIP(ip string) bool {
// 	ac.mu.RLock()
// 	defer ac.mu.RUnlock()
// 	return ac.masterIPs[ip]
// }

// func (ac *AccessControl) StartSession(ip string) bool {
// 	// اگر IP مستر است، همیشه اجازه دسترسی دارد
// 	if ac.IsMasterIP(ip) {
// 		return true
// 	}

// 	ac.mu.Lock()
// 	defer ac.mu.Unlock()

// 	access, exists := ac.accessRecords[ip]
// 	now := time.Now()

// 	if !exists {
// 		access = &IPAccess{
// 			lastReset: now,
// 		}
// 		ac.accessRecords[ip] = access
// 	}

// 	// چک کردن ریست روزانه
// 	if now.Day() != access.lastReset.Day() {
// 		access.totalTime = 0
// 		access.isBlocked = false
// 		access.lastReset = now
// 	}

// 	// اگر مسدود است
// 	if access.isBlocked {
// 		return false
// 	}

// 	// اگر از محدودیت روزانه عبور کرده
// 	if access.totalTime >= ac.dailyLimit {
// 		access.isBlocked = true
// 		return false
// 	}

// 	// شروع سشن جدید
// 	access.activeSession = &now
// 	access.lastAccess = now
// 	return true
// }

// func (ac *AccessControl) EndSession(ip string) {
// 	// برای IP های مستر نیازی به ثبت زمان نیست
// 	if ac.IsMasterIP(ip) {
// 		return
// 	}

// 	ac.mu.Lock()
// 	defer ac.mu.Unlock()

// 	if access, exists := ac.accessRecords[ip]; exists && access.activeSession != nil {
// 		sessionDuration := time.Since(*access.activeSession)
// 		access.totalTime += sessionDuration
// 		access.activeSession = nil
// 		access.lastAccess = time.Now()

// 		// اگر از محدودیت عبور کرده، مسدود شود
// 		if access.totalTime >= ac.dailyLimit {
// 			access.isBlocked = true
// 		}
// 	}
// }

// func (ac *AccessControl) GetIPStatus(ip string) (time.Duration, bool) {
// 	ac.mu.RLock()
// 	defer ac.mu.RUnlock()

// 	if ac.masterIPs[ip] {
// 		return 0, false // IP های مستر محدودیت ندارند
// 	}

// 	if access, exists := ac.accessRecords[ip]; exists {
// 		return access.totalTime, access.isBlocked
// 	}

// 	return 0, false
// }
