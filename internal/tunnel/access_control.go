package tunnel

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/internal/storage"

	"fyne.io/fyne/v2"
)

type AccessRule struct {
	ID          string
	Title       string
	IP          string
	IsMaster    bool
	DailyLimit  time.Duration
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AccessStatus struct {
	RuleID      string
	UsedTime    time.Duration
	IsBlocked   bool
	LastAccess  time.Time
	ResetTime   time.Time
	ActiveSince *time.Time
}

type AccessControl struct {
	mu            sync.RWMutex
	rules         map[string]*AccessRule   // key: rule ID
	rulesByIP     map[string]string        // key: IP, value: rule ID
	status        map[string]*AccessStatus // key: rule ID
	defaultLimit  time.Duration
	configManager common.ConfigManager
	storage       *storage.AppStorage
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func NewAccessControl(app fyne.App, defaultLimit time.Duration) *AccessControl {
	ac := &AccessControl{
		rules:        make(map[string]*AccessRule),
		rulesByIP:    make(map[string]string),
		status:       make(map[string]*AccessStatus),
		defaultLimit: defaultLimit,
	}

	ac.storage, _ = storage.NewAppStorage(app)
	ac.configManager = &DefaultConfigManager{
		Storage: ac.storage,
	}

	rules := ac.configManager.LoadConfig().Rules
	for _, rule := range rules {
		duration, err := time.ParseDuration(rule.DailyLimit)
		if err != nil {
			duration = 1 * time.Hour
		}
		ac.rules[rule.ID] = &AccessRule{
			ID:          rule.ID,
			Title:       rule.Title,
			IP:          rule.IP,
			IsMaster:    rule.IsMaster,
			DailyLimit:  duration,
			Description: rule.Description,
			CreatedAt:   rule.CreatedAt,
			UpdatedAt:   rule.UpdatedAt,
		}
		ac.rulesByIP[rule.IP] = rule.ID

		ac.status[rule.ID] = &AccessStatus{
			RuleID:     rule.ID,
			UsedTime:   rule.UsedTime,
			IsBlocked:  rule.IsBlocked,
			LastAccess: rule.LastAccess,
			ResetTime:  time.Now(),
		}
	}

	// Start daily reset worker
	go ac.dailyResetWorker()

	return ac
}

func (ac *AccessControl) dailyResetWorker() {
	for {
		now := time.Now()
		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(nextDay))

		ac.mu.Lock()
		for _, status := range ac.status {
			if rule, exists := ac.rules[status.RuleID]; exists && !rule.IsMaster {
				status.UsedTime = 0
				status.IsBlocked = false
				status.ResetTime = nextDay
			}
		}
		ac.mu.Unlock()
	}
}

func (ac *AccessControl) AddRule(rule *AccessRule) error {
	ac.mu.Lock()

	if rule.ID == "" {
		rule.ID = generateUUID()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	newRule := &AccessRule{
		ID:          rule.ID,
		Title:       rule.Title,
		IP:          rule.IP,
		IsMaster:    rule.IsMaster,
		DailyLimit:  rule.DailyLimit,
		Description: rule.Description,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}

	ac.rules[newRule.ID] = newRule
	ac.rulesByIP[newRule.IP] = newRule.ID
	ac.status[newRule.ID] = &AccessStatus{
		RuleID:    newRule.ID,
		ResetTime: time.Now(),
	}

	ac.mu.Unlock()

	return ac.SaveRules()
}

func (ac *AccessControl) UpdateRule(rule *AccessRule) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if existing, ok := ac.rules[rule.ID]; ok {
		// Remove old IP mapping
		delete(ac.rulesByIP, existing.IP)

		// Update rule
		rule.CreatedAt = existing.CreatedAt
		rule.UpdatedAt = time.Now()
		ac.rules[rule.ID] = rule
		ac.rulesByIP[rule.IP] = rule.ID
	}

	return ac.SaveRules()
}

func (ac *AccessControl) DeleteRule(ruleID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if rule, ok := ac.rules[ruleID]; ok {
		delete(ac.rulesByIP, rule.IP)
		delete(ac.rules, ruleID)
		delete(ac.status, ruleID)
	}

	return ac.SaveRules()
}

func (ac *AccessControl) ResetRule(ruleID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if status, ok := ac.status[ruleID]; ok {
		status.UsedTime = 0
		status.IsBlocked = false
		status.ResetTime = time.Now()
		status.ActiveSince = nil
	}

	return ac.SaveRules()
}

func (ac *AccessControl) GetRule(ruleID string) (*AccessRule, *AccessStatus) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	rule := ac.rules[ruleID]
	status := ac.status[ruleID]
	return rule, status
}

func (ac *AccessControl) GetRuleByIP(ip string) (*AccessRule, *AccessStatus) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ruleID, ok := ac.rulesByIP[ip]; ok {
		return ac.rules[ruleID], ac.status[ruleID]
	}
	return nil, nil
}

func (ac *AccessControl) GetAllRules() []*AccessRule {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	rules := make([]*AccessRule, 0, len(ac.rules))
	for _, rule := range ac.rules {
		rules = append(rules, rule)
	}
	return rules
}

func (ac *AccessControl) StartSession(ip string) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	rule, status := ac.getRuleByIPLocked(ip)
	if rule == nil {
		return true // No rule means no restriction
	}

	if rule.IsMaster {
		return true
	}

	now := time.Now()
	if status.ActiveSince == nil {
		if !status.IsBlocked && status.UsedTime < rule.DailyLimit {
			status.ActiveSince = &now
			status.LastAccess = now
			return true
		}
	}
	return false
}

func (ac *AccessControl) EndSession(ip string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	rule, status := ac.getRuleByIPLocked(ip)
	if rule == nil || rule.IsMaster {
		return
	}

	if status.ActiveSince != nil {
		now := time.Now()
		status.UsedTime += now.Sub(*status.ActiveSince)
		status.LastAccess = now
		status.ActiveSince = nil

		if status.UsedTime >= rule.DailyLimit {
			status.IsBlocked = true
		}

		ac.SaveRules() // Ignore error here
	}
}

func (ac *AccessControl) getRuleByIPLocked(ip string) (*AccessRule, *AccessStatus) {
	if ruleID, ok := ac.rulesByIP[ip]; ok {
		return ac.rules[ruleID], ac.status[ruleID]
	}
	return nil, nil
}

func (ac *AccessControl) SaveRules() error {
	ac.mu.RLock()
	rules := make([]*models.Rule, 0, len(ac.rules))

	for _, rule := range ac.rules {
		status := ac.status[rule.ID]
		rules = append(rules, &models.Rule{
			ID:          rule.ID,
			Title:       rule.Title,
			IP:          rule.IP,
			IsMaster:    rule.IsMaster,
			DailyLimit:  rule.DailyLimit.String(),
			Description: rule.Description,
			CreatedAt:   rule.CreatedAt,
			UpdatedAt:   rule.UpdatedAt,
			LastAccess:  status.LastAccess,
			UsedTime:    status.UsedTime,
			IsBlocked:   status.IsBlocked,
		})
	}
	ac.mu.RUnlock()

	config := &common.Config{
		Rules: rules,
	}

	return ac.configManager.SaveConfig(config)
}
