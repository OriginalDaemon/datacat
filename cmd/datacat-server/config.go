package main

import (
	"encoding/json"
	"os"
	"time"
)

// Config holds server configuration
type Config struct {
	DataPath                string        `json:"data_path"`
	RetentionDays           int           `json:"retention_days"`
	CleanupIntervalHours    int           `json:"cleanup_interval_hours"`
	ServerPort              string        `json:"server_port"`
	HeartbeatTimeoutSeconds int           `json:"heartbeat_timeout_seconds"`
	APIKey                  string        `json:"api_key,omitempty"`       // Optional API key for authentication
	RequireAPIKey           bool          `json:"require_api_key"`         // Require API key for all requests
	TLSCertFile             string        `json:"tls_cert_file,omitempty"` // Path to TLS certificate
	TLSKeyFile              string        `json:"tls_key_file,omitempty"`  // Path to TLS private key
	LogFile                 string        `json:"log_file,omitempty"`      // Path to log file (empty = use default tmp file)
	CleanupInterval         time.Duration `json:"-"`                       // Derived field, not serialized
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DataPath:                "./datacat_data",
		RetentionDays:           365,
		CleanupIntervalHours:    24,
		ServerPort:              "9090",
		HeartbeatTimeoutSeconds: 60,
		APIKey:                  "",    // No API key by default (open access)
		RequireAPIKey:           false, // Don't require API key by default
		TLSCertFile:             "",    // No TLS by default
		TLSKeyFile:              "",    // No TLS by default
		CleanupInterval:         24 * time.Hour,
	}
}

// LoadConfig loads configuration from file or creates default
// Note: This function does not log anything to avoid issues if called before logging is initialized
func LoadConfig(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		// Config file not found, use defaults and try to save (ignore errors)
		config := DefaultConfig()
		_ = SaveConfig(path, config) // Ignore error, will use defaults if save fails
		return config
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		// Failed to decode, use defaults
		return DefaultConfig()
	}

	// Compute derived fields
	config.CleanupInterval = time.Duration(config.CleanupIntervalHours) * time.Hour

	return &config
}

// SaveConfig saves configuration to file
func SaveConfig(path string, config *Config) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}
