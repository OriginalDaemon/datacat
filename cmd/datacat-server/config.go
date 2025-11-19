package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Config holds server configuration
type Config struct {
	DataPath             string        `json:"data_path"`
	RetentionDays        int           `json:"retention_days"`
	CleanupIntervalHours int           `json:"cleanup_interval_hours"`
	ServerPort           string        `json:"server_port"`
	CleanupInterval      time.Duration `json:"-"` // Derived field, not serialized
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DataPath:             "./datacat_data",
		RetentionDays:        365,
		CleanupIntervalHours: 24,
		ServerPort:           "8080",
		CleanupInterval:      24 * time.Hour,
	}
}

// LoadConfig loads configuration from file or creates default
func LoadConfig(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Config file not found, using defaults")
		config := DefaultConfig()
		SaveConfig(path, config)
		return config
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Printf("Failed to decode config, using defaults: %v", err)
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
