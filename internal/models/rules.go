package models

import (
	"time"
)

type RuleConfig struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	IP          string        `json:"ip"`
	IsMaster    bool          `json:"is_master"`
	DailyLimit  string        `json:"daily_limit"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	LastAccess  time.Time     `json:"last_access,omitempty"`
	UsedTime    time.Duration `json:"used_time,omitempty"`
	IsBlocked   bool          `json:"is_blocked,omitempty"`
}
