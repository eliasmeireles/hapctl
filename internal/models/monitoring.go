package models

import "time"

type BindStatus struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type MonitoringReport struct {
	Timestamp time.Time    `json:"timestamp"`
	Binds     []BindStatus `json:"binds"`
}
