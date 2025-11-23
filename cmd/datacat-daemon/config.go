package main

import (
	"encoding/json"
	"log"
	"os"
)

// Config holds daemon configuration
type Config struct {
	DaemonPort              string `json:"daemon_port"`
	ServerURL               string `json:"server_url"`
	BatchIntervalSeconds    int    `json:"batch_interval_seconds"`
	MaxBatchSize            int    `json:"max_batch_size"`
	HeartbeatTimeoutSeconds int    `json:"heartbeat_timeout_seconds"`
	APIKey                  string `json:"api_key,omitempty"`           // Optional API key for authentication
	EnableCompression       bool   `json:"enable_compression"`          // Enable gzip compression
	TLSVerify               bool   `json:"tls_verify"`                  // Verify TLS certificates (default true)
	TLSInsecureSkipVerify   bool   `json:"tls_insecure_skip_verify"`   // Skip TLS verification (for self-signed certs)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DaemonPort:              "8079",
		ServerURL:               "http://localhost:9090",
		BatchIntervalSeconds:    5,
		MaxBatchSize:            100,
		HeartbeatTimeoutSeconds: 60,
		APIKey:                  "",    // No API key by default
		EnableCompression:       true,  // Enable compression by default
		TLSVerify:               true,  // Verify TLS certificates by default
		TLSInsecureSkipVerify:   false, // Don't skip verification by default
	}
}

// LoadConfig loads configuration from file or creates default
func LoadConfig(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Config file not found, using defaults")
		config := DefaultConfig()
		_ = SaveConfig(path, config) // Ignore error, will use defaults if save fails
		return config
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Printf("Failed to decode config, using defaults: %v", err)
		return DefaultConfig()
	}

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
