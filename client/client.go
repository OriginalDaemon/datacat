// Package client provides a Go client for the datacat REST API
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// DaemonManager manages the local datacat daemon subprocess
type DaemonManager struct {
	daemonPort   string
	serverURL    string
	daemonBinary string
	process      *exec.Cmd
	started      bool
}

// NewDaemonManager creates a new daemon manager
func NewDaemonManager(daemonPort, serverURL, daemonBinary string) *DaemonManager {
	if daemonBinary == "" {
		daemonBinary = findDaemonBinary()
	}
	return &DaemonManager{
		daemonPort:   daemonPort,
		serverURL:    serverURL,
		daemonBinary: daemonBinary,
	}
}

// findDaemonBinary finds the daemon binary in common locations
func findDaemonBinary() string {
	// Determine binary name based on platform
	binaryName := "datacat-daemon"
	if runtime.GOOS == "windows" {
		binaryName = "datacat-daemon.exe"
	}
	
	// Check common locations
	possiblePaths := []string{
		binaryName,                                       // In PATH
		"./" + binaryName,                                // Current directory
		"./cmd/datacat-daemon/" + binaryName,             // Development
		"./bin/" + binaryName,                            // Built binaries
	}

	for _, path := range possiblePaths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return binaryName // Default and let it fail if not found
}

// Start starts the daemon subprocess
func (dm *DaemonManager) Start() error {
	if dm.started && dm.process != nil && dm.process.Process != nil {
		return nil // Already running
	}

	// Create config for daemon
	config := map[string]interface{}{
		"daemon_port":               dm.daemonPort,
		"server_url":                dm.serverURL,
		"batch_interval_seconds":    5,
		"max_batch_size":            100,
		"heartbeat_timeout_seconds": 60,
	}

	// Write config to temporary file
	configPath := "daemon_config.json"
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Start daemon process
	dm.process = exec.Command(dm.daemonBinary)
	if err := dm.process.Start(); err != nil {
		return fmt.Errorf("failed to start daemon binary '%s': %w", dm.daemonBinary, err)
	}

	dm.started = true

	// Wait a bit for daemon to start
	time.Sleep(1 * time.Second)

	return nil
}

// Stop stops the daemon subprocess
func (dm *DaemonManager) Stop() error {
	if dm.process != nil && dm.process.Process != nil {
		if err := dm.process.Process.Kill(); err != nil {
			return err
		}
		_ = dm.process.Wait() // Wait for process to exit, error is expected after Kill
	}
	dm.started = false
	return nil
}

// IsRunning checks if daemon is running
func (dm *DaemonManager) IsRunning() bool {
	return dm.started && dm.process != nil && dm.process.Process != nil
}

// Client is a datacat API client
type Client struct {
	BaseURL       string
	HTTPClient    *http.Client
	UseDaemon     bool
	DaemonManager *DaemonManager
}

// Session represents a datacat session
type Session struct {
	ID           string                 `json:"id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	Active       bool                   `json:"active"`
	State        map[string]interface{} `json:"state"`
	StateHistory []StateSnapshot        `json:"state_history"`
	Events       []Event                `json:"events"`
	Metrics      []Metric               `json:"metrics"`
}

// StateSnapshot represents the state at a specific point in time
type StateSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	State     map[string]interface{} `json:"state"`
}

// Event represents an event in a session
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data"`
}

// Metric represents a metric in a session
type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Tags      []string  `json:"tags,omitempty"`
}

// NewClient creates a new datacat client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UseDaemon: false,
	}
}

// NewClientWithDaemon creates a new datacat client that uses a local daemon
func NewClientWithDaemon(serverURL, daemonPort string) (*Client, error) {
	dm := NewDaemonManager(daemonPort, serverURL, "")
	if err := dm.Start(); err != nil {
		return nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	return &Client{
		BaseURL: fmt.Sprintf("http://localhost:%s", daemonPort),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UseDaemon:     true,
		DaemonManager: dm,
	}, nil
}

// Close closes the client and stops the daemon if running
func (c *Client) Close() error {
	if c.DaemonManager != nil {
		return c.DaemonManager.Stop()
	}
	return nil
}

// CreateSession creates a new session
func (c *Client) CreateSession(product, version string) (string, error) {
	if product == "" || version == "" {
		return "", fmt.Errorf("product and version are required to create a session")
	}

	var url string
	var reqData []byte
	var err error

	if c.UseDaemon {
		url = c.BaseURL + "/register"
		// Send parent PID so daemon can monitor for crashes
		data := map[string]interface{}{
			"parent_pid": os.Getpid(),
			"product":    product,
			"version":    version,
		}
		reqData, err = json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}
	} else {
		url = c.BaseURL + "/api/sessions"
		data := map[string]interface{}{
			"product": product,
			"version": version,
		}
		reqData, err = json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result["session_id"], nil
}

// GetSession retrieves a session by ID
func (c *Client) GetSession(sessionID string) (*Session, error) {
	var url string
	
	if c.UseDaemon {
		url = c.BaseURL + "/session?session_id=" + sessionID
	} else {
		url = c.BaseURL + "/api/sessions/" + sessionID
	}
	
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get session failed with status %d: %s", resp.StatusCode, string(body))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// UpdateState updates session state
func (c *Client) UpdateState(sessionID string, state map[string]interface{}) error {
	var url string
	var data interface{}

	if c.UseDaemon {
		url = c.BaseURL + "/state"
		data = map[string]interface{}{
			"session_id": sessionID,
			"state":      state,
		}
	} else {
		url = c.BaseURL + "/api/sessions/" + sessionID + "/state"
		data = state
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update state failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LogEvent logs an event
func (c *Client) LogEvent(sessionID, name string, data map[string]interface{}) error {
	var url string
	var requestData interface{}

	if c.UseDaemon {
		url = c.BaseURL + "/event"
		requestData = map[string]interface{}{
			"session_id": sessionID,
			"name":       name,
			"data":       data,
		}
	} else {
		url = c.BaseURL + "/api/sessions/" + sessionID + "/events"
		requestData = map[string]interface{}{
			"name": name,
			"data": data,
		}
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("log event failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LogMetric logs a metric
func (c *Client) LogMetric(sessionID, name string, value float64, tags []string) error {
	var url string
	var metricData interface{}

	if c.UseDaemon {
		url = c.BaseURL + "/metric"
		metricData = map[string]interface{}{
			"session_id": sessionID,
			"name":       name,
			"value":      value,
			"tags":       tags,
		}
	} else {
		url = c.BaseURL + "/api/sessions/" + sessionID + "/metrics"
		metricData = map[string]interface{}{
			"name":  name,
			"value": value,
			"tags":  tags,
		}
	}

	jsonData, err := json.Marshal(metricData)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to log metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("log metric failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EndSession ends a session
func (c *Client) EndSession(sessionID string) error {
	var url string
	var reqData []byte
	var err error

	if c.UseDaemon {
		url = c.BaseURL + "/end"
		data := map[string]interface{}{
			"session_id": sessionID,
		}
		reqData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	} else {
		url = c.BaseURL + "/api/sessions/" + sessionID + "/end"
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("end session failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Heartbeat sends a heartbeat to the daemon (only works when using daemon)
func (c *Client) Heartbeat(sessionID string) error {
	if !c.UseDaemon {
		return nil // Heartbeat only relevant with daemon
	}

	url := c.BaseURL + "/heartbeat"
	data := map[string]interface{}{
		"session_id": sessionID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAllSessions retrieves all sessions
func (c *Client) GetAllSessions() ([]*Session, error) {
	var url string
	
	if c.UseDaemon {
		url = c.BaseURL + "/sessions"
	} else {
		url = c.BaseURL + "/api/data/sessions"
	}
	
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get sessions failed with status %d: %s", resp.StatusCode, string(body))
	}

	var sessions []*Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}
