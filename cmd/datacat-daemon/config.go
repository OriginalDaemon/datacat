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
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DaemonPort:              "8079",
		ServerURL:               "http://localhost:8080",
		BatchIntervalSeconds:    5,
		MaxBatchSize:            100,
		HeartbeatTimeoutSeconds: 60,
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
