package models

import "time"

type SSLConfig struct {
	Config struct {
		Mail            string            `yaml:"mail"`
		Domain          []string          `yaml:"domain"`
		DNSProvider     string            `yaml:"dns-provider,omitempty"`
		DNSCredentials  map[string]string `yaml:"dns-credentials,omitempty"`
		Staging         bool              `yaml:"staging,omitempty"`
		KeySize         int               `yaml:"key-size,omitempty"`
		KeyType         string            `yaml:"key-type,omitempty"`
		ECDSACurve      string            `yaml:"ecdsa-curve,omitempty"`
		PreferredChain  string            `yaml:"preferred-chain,omitempty"`
		MustStaple      bool              `yaml:"must-staple,omitempty"`
		ReuseKey        bool              `yaml:"reuse-key,omitempty"`
		HTTPPort        int               `yaml:"http-port,omitempty"`
		TLSPort         int               `yaml:"tls-port,omitempty"`
	} `yaml:"config"`
}

type SSLManagerConfig struct {
	Enabled        bool              `yaml:"enabled"`
	ConfigPath     string            `yaml:"config-path"`
	CertPath       string            `yaml:"cert-path"`
	RenewalCheck   time.Duration     `yaml:"renewal-check"`
	Email          string            `yaml:"email"`
	DNSProvider    string            `yaml:"dns-provider,omitempty"`
	DNSCredentials map[string]string `yaml:"dns-credentials,omitempty"`
}

type Certificate struct {
	Domain      string
	CertPath    string
	KeyPath     string
	FullChain   string
	PEMPath     string
	ExpiryDate  time.Time
	IsWildcard  bool
	LastChecked time.Time
}

type CertificateStatus struct {
	Domain        string    `json:"domain"`
	ExpiryDate    time.Time `json:"expiry_date"`
	DaysRemaining int       `json:"days_remaining"`
	NeedsRenewal  bool      `json:"needs_renewal"`
	IsValid       bool      `json:"is_valid"`
	Error         string    `json:"error,omitempty"`
}
