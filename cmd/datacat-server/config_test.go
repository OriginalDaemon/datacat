package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.DataPath != "./datacat_data" {
		t.Errorf("Expected DataPath ./datacat_data, got %s", config.DataPath)
	}
	if config.RetentionDays != 365 {
		t.Errorf("Expected RetentionDays 365, got %d", config.RetentionDays)
	}
	if config.CleanupIntervalHours != 24 {
		t.Errorf("Expected CleanupIntervalHours 24, got %d", config.CleanupIntervalHours)
	}
	if config.ServerPort != "9090" {
		t.Errorf("Expected ServerPort 9090, got %s", config.ServerPort)
	}
	if config.CleanupInterval != 24*time.Hour {
		t.Errorf("Expected CleanupInterval 24h, got %v", config.CleanupInterval)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"

	// Test loading non-existent config (should create default)
	config := LoadConfig(configPath)
	if config == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if config.DataPath != "./datacat_data" {
		t.Error("Should return default config when file doesn't exist")
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should have been created")
	}
}

func TestLoadConfigExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"

	// Create a custom config file
	customConfig := &Config{
		DataPath:             "/custom/path",
		RetentionDays:        30,
		CleanupIntervalHours: 12,
		ServerPort:           "9090",
	}

	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	json.NewEncoder(file).Encode(customConfig)
	file.Close()

	// Load config
	loaded := LoadConfig(configPath)

	if loaded.DataPath != "/custom/path" {
		t.Errorf("Expected DataPath /custom/path, got %s", loaded.DataPath)
	}
	if loaded.RetentionDays != 30 {
		t.Errorf("Expected RetentionDays 30, got %d", loaded.RetentionDays)
	}
	if loaded.CleanupIntervalHours != 12 {
		t.Errorf("Expected CleanupIntervalHours 12, got %d", loaded.CleanupIntervalHours)
	}
	if loaded.ServerPort != "9090" {
		t.Errorf("Expected ServerPort 9090, got %s", loaded.ServerPort)
	}
	if loaded.CleanupInterval != 12*time.Hour {
		t.Errorf("Expected CleanupInterval 12h, got %v", loaded.CleanupInterval)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"

	config := &Config{
		DataPath:             "/test/path",
		RetentionDays:        100,
		CleanupIntervalHours: 6,
		ServerPort:           "7070",
		CleanupInterval:      6 * time.Hour,
	}

	SaveConfig(configPath, config)

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

	if loaded.DataPath != "/test/path" {
		t.Errorf("Expected DataPath /test/path, got %s", loaded.DataPath)
	}
	if loaded.RetentionDays != 100 {
		t.Errorf("Expected RetentionDays 100, got %d", loaded.RetentionDays)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/invalid.json"

	// Create invalid JSON file
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	file.WriteString("{invalid json")
	file.Close()

	// Load config - should return default on error
	loaded := LoadConfig(configPath)

	if loaded.DataPath != "./datacat_data" {
		t.Error("Should return default config when JSON is invalid")
	}
}

func TestSaveConfigInvalidPath(t *testing.T) {
	config := DefaultConfig()

	// Try to save to invalid path
	err := SaveConfig("/invalid/path/that/does/not/exist/config.json", config)
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}
