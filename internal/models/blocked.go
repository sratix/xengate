package models

import "time"

type BlockedIPInfo struct {
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}
