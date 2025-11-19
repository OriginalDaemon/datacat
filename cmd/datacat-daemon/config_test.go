package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.DaemonPort != "8079" {
		t.Errorf("Expected DaemonPort 8079, got %s", config.DaemonPort)
	}
	if config.ServerURL != "http://localhost:8080" {
		t.Errorf("Expected ServerURL http://localhost:8080, got %s", config.ServerURL)
	}
	if config.BatchIntervalSeconds != 5 {
		t.Errorf("Expected BatchIntervalSeconds 5, got %d", config.BatchIntervalSeconds)
	}
	if config.MaxBatchSize != 100 {
		t.Errorf("Expected MaxBatchSize 100, got %d", config.MaxBatchSize)
	}
	if config.HeartbeatTimeoutSeconds != 60 {
		t.Errorf("Expected HeartbeatTimeoutSeconds 60, got %d", config.HeartbeatTimeoutSeconds)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/daemon_config.json"
	
	// Test loading non-existent config (should create default)
	config := LoadConfig(configPath)
	if config == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if config.DaemonPort != "8079" {
		t.Error("Should return default config when file doesn't exist")
	}
	
	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should have been created")
	}
}

func TestLoadConfigExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/daemon_config.json"
	
	// Create a custom config file
	customConfig := &Config{
		DaemonPort:              "9079",
		ServerURL:               "http://example.com:8080",
		BatchIntervalSeconds:    10,
		MaxBatchSize:            200,
		HeartbeatTimeoutSeconds: 120,
	}
	
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	json.NewEncoder(file).Encode(customConfig)
	file.Close()
	
	// Load config
	loaded := LoadConfig(configPath)
	
	if loaded.DaemonPort != "9079" {
		t.Errorf("Expected DaemonPort 9079, got %s", loaded.DaemonPort)
	}
	if loaded.ServerURL != "http://example.com:8080" {
		t.Errorf("Expected ServerURL http://example.com:8080, got %s", loaded.ServerURL)
	}
	if loaded.BatchIntervalSeconds != 10 {
		t.Errorf("Expected BatchIntervalSeconds 10, got %d", loaded.BatchIntervalSeconds)
	}
	if loaded.MaxBatchSize != 200 {
		t.Errorf("Expected MaxBatchSize 200, got %d", loaded.MaxBatchSize)
	}
	if loaded.HeartbeatTimeoutSeconds != 120 {
		t.Errorf("Expected HeartbeatTimeoutSeconds 120, got %d", loaded.HeartbeatTimeoutSeconds)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/daemon_config.json"
	
	config := &Config{
		DaemonPort:              "7079",
		ServerURL:               "http://test.com:8080",
		BatchIntervalSeconds:    3,
		MaxBatchSize:            50,
		HeartbeatTimeoutSeconds: 30,
	}
	
	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}
	
	// Load and verify
	file, err := os.Open(configPath)
	if err != nil {
		t.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()
	
	var loaded Config
	json.NewDecoder(file).Decode(&loaded)
	
	if loaded.DaemonPort != "7079" {
		t.Errorf("Expected DaemonPort 7079, got %s", loaded.DaemonPort)
	}
	if loaded.ServerURL != "http://test.com:8080" {
		t.Errorf("Expected ServerURL http://test.com:8080, got %s", loaded.ServerURL)
	}
}
