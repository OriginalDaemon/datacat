package main

import (
	"encoding/json"
	"os"
)

// Config holds web UI configuration
type Config struct {
	ServerURL string `json:"server_url"`         // URL of the datacat server
	Port      string `json:"port"`               // Port for the web UI
	LogFile   string `json:"log_file,omitempty"` // Path to log file (empty = use default tmp file)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ServerURL: "http://localhost:9090",
		Port:      "8080",
		LogFile:   "", // Empty = use default tmp file
	}
}

// LoadConfig loads configuration from file or creates default
// Note: This function does not log anything to avoid issues if called before logging is initialized
func LoadConfig(path string) *Config {
	defaults := DefaultConfig()

	file, err := os.Open(path)
	if err != nil {
		// Config file not found, use defaults
		return defaults
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		// Failed to decode, use defaults
		return defaults
	}

	// Merge with defaults - if a field is empty, use the default value
	if config.ServerURL == "" {
		config.ServerURL = defaults.ServerURL
	}
	if config.Port == "" {
		config.Port = defaults.Port
	}
	// LogFile can be empty, that's fine

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
