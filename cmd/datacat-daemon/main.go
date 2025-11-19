package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

// SessionBuffer holds pending updates for a session
type SessionBuffer struct {
	SessionID     string
	StateUpdates  []map[string]interface{}
	Events        []EventData
	Metrics       []MetricData
	LastHeartbeat time.Time
	LastState     map[string]interface{}
	HangLogged    bool
	ParentPID     int  // Parent process ID
	CrashLogged   bool // Whether crash has been logged
	mu            sync.Mutex
}

// EventData represents an event to be logged
type EventData struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

// MetricData represents a metric to be logged
type MetricData struct {
	Name  string   `json:"name"`
	Value float64  `json:"value"`
	Tags  []string `json:"tags,omitempty"`
}

// Daemon manages batching and forwarding to the server
type Daemon struct {
	config   *Config
	sessions map[string]*SessionBuffer
	mu       sync.RWMutex
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) *Daemon {
	return &Daemon{
		config:   config,
		sessions: make(map[string]*SessionBuffer),
	}
}

// Start starts the daemon HTTP server and background workers
func (d *Daemon) Start() error {
	// Start batch sender goroutine
	go d.batchSender()

	// Start heartbeat monitor goroutine
	go d.heartbeatMonitor()

	// Start parent process monitor goroutine
	go d.parentProcessMonitor()

	// Setup HTTP handlers
	http.HandleFunc("/register", d.handleRegister)
	http.HandleFunc("/state", d.handleState)
	http.HandleFunc("/event", d.handleEvent)
	http.HandleFunc("/metric", d.handleMetric)
	http.HandleFunc("/heartbeat", d.handleHeartbeat)
	http.HandleFunc("/end", d.handleEnd)
	http.HandleFunc("/health", d.handleHealth)

	addr := ":" + d.config.DaemonPort
	log.Printf("Daemon listening on %s, forwarding to %s", addr, d.config.ServerURL)
	return http.ListenAndServe(addr, nil)
}

// handleRegister registers a new session
func (d *Daemon) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body to get parent PID
	var req struct {
		ParentPID int `json:"parent_pid"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Forward to server to create session
	resp, err := http.Post(d.config.ServerURL+"/api/sessions", "application/json", nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode response: %v", err), http.StatusInternalServerError)
		return
	}

	sessionID, ok := result["session_id"].(string)
	if !ok {
		http.Error(w, "Invalid session ID in response", http.StatusInternalServerError)
		return
	}

	// Create buffer for this session
	d.mu.Lock()
	d.sessions[sessionID] = &SessionBuffer{
		SessionID:     sessionID,
		StateUpdates:  make([]map[string]interface{}, 0),
		Events:        make([]EventData, 0),
		Metrics:       make([]MetricData, 0),
		LastHeartbeat: time.Now(),
		LastState:     make(map[string]interface{}),
		HangLogged:    false,
		ParentPID:     req.ParentPID,
		CrashLogged:   false,
	}
	d.mu.Unlock()

	log.Printf("Registered session: %s (parent PID: %d)", sessionID, req.ParentPID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleState handles state update requests
func (d *Daemon) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string                 `json:"session_id"`
		State     map[string]interface{} `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	d.mu.RLock()
	buffer, exists := d.sessions[req.SessionID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	// Check if state actually changed
	if d.hasStateChanged(buffer.LastState, req.State) {
		buffer.StateUpdates = append(buffer.StateUpdates, req.State)
		// Update last known state
		buffer.LastState = d.mergeState(buffer.LastState, req.State)
	}

	w.WriteHeader(http.StatusOK)
}

// handleEvent handles event logging requests
func (d *Daemon) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string                 `json:"session_id"`
		Name      string                 `json:"name"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	d.mu.RLock()
	buffer, exists := d.sessions[req.SessionID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	buffer.mu.Lock()
	buffer.Events = append(buffer.Events, EventData{
		Name: req.Name,
		Data: req.Data,
	})
	buffer.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// handleMetric handles metric logging requests
func (d *Daemon) handleMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string   `json:"session_id"`
		Name      string   `json:"name"`
		Value     float64  `json:"value"`
		Tags      []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	d.mu.RLock()
	buffer, exists := d.sessions[req.SessionID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	buffer.mu.Lock()
	buffer.Metrics = append(buffer.Metrics, MetricData{
		Name:  req.Name,
		Value: req.Value,
		Tags:  req.Tags,
	})
	buffer.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// handleHeartbeat handles heartbeat requests
func (d *Daemon) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	d.mu.RLock()
	buffer, exists := d.sessions[req.SessionID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	buffer.mu.Lock()
	buffer.LastHeartbeat = time.Now()
	if buffer.HangLogged {
		// Application recovered
		buffer.Events = append(buffer.Events, EventData{
			Name: "application_recovered",
			Data: map[string]interface{}{},
		})
		buffer.HangLogged = false
	}
	buffer.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// handleEnd handles session end requests
func (d *Daemon) handleEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Flush any pending data for this session
	d.flushSession(req.SessionID)

	// Forward end request to server
	endData, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+req.SessionID+"/end",
		"application/json",
		bytes.NewBuffer(endData),
	)
	if err == nil {
		_ = resp.Body.Close()
	}

	// Remove session from daemon
	d.mu.Lock()
	delete(d.sessions, req.SessionID)
	d.mu.Unlock()

	log.Printf("Session ended: %s", req.SessionID)
	w.WriteHeader(http.StatusOK)
}

// handleHealth handles health check requests
func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"sessions": len(d.sessions),
	})
}

// batchSender periodically sends batched data to server
func (d *Daemon) batchSender() {
	ticker := time.NewTicker(time.Duration(d.config.BatchIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.RLock()
		sessionIDs := make([]string, 0, len(d.sessions))
		for id := range d.sessions {
			sessionIDs = append(sessionIDs, id)
		}
		d.mu.RUnlock()

		for _, sessionID := range sessionIDs {
			d.flushSession(sessionID)
		}
	}
}

// flushSession sends all pending data for a session to the server
func (d *Daemon) flushSession(sessionID string) {
	d.mu.RLock()
	buffer, exists := d.sessions[sessionID]
	d.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	stateUpdates := buffer.StateUpdates
	events := buffer.Events
	metrics := buffer.Metrics
	buffer.StateUpdates = make([]map[string]interface{}, 0)
	buffer.Events = make([]EventData, 0)
	buffer.Metrics = make([]MetricData, 0)
	buffer.mu.Unlock()

	// Send state updates
	for _, state := range stateUpdates {
		d.sendStateUpdate(sessionID, state)
	}

	// Send events
	for _, event := range events {
		d.sendEvent(sessionID, event)
	}

	// Send metrics
	for _, metric := range metrics {
		d.sendMetric(sessionID, metric)
	}
}

// sendStateUpdate sends a state update to the server
func (d *Daemon) sendStateUpdate(sessionID string, state map[string]interface{}) {
	data, _ := json.Marshal(state)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/state",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Printf("Failed to send state update: %v", err)
		return
	}
	_ = resp.Body.Close()
}

// sendEvent sends an event to the server
func (d *Daemon) sendEvent(sessionID string, event EventData) {
	data, _ := json.Marshal(event)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/events",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	_ = resp.Body.Close()
}

// sendMetric sends a metric to the server
func (d *Daemon) sendMetric(sessionID string, metric MetricData) {
	data, _ := json.Marshal(metric)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/metrics",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Printf("Failed to send metric: %v", err)
		return
	}
	_ = resp.Body.Close()
}

// heartbeatMonitor checks for hung applications
func (d *Daemon) heartbeatMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.RLock()
		sessionIDs := make([]string, 0, len(d.sessions))
		for id := range d.sessions {
			sessionIDs = append(sessionIDs, id)
		}
		d.mu.RUnlock()

		for _, sessionID := range sessionIDs {
			d.checkHeartbeat(sessionID)
		}
	}
}

// checkHeartbeat checks if a session has hung
func (d *Daemon) checkHeartbeat(sessionID string) {
	d.mu.RLock()
	buffer, exists := d.sessions[sessionID]
	d.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	timeout := time.Duration(d.config.HeartbeatTimeoutSeconds) * time.Second
	if time.Since(buffer.LastHeartbeat) > timeout && !buffer.HangLogged {
		// Application appears hung
		buffer.Events = append(buffer.Events, EventData{
			Name: "application_appears_hung",
			Data: map[string]interface{}{
				"last_heartbeat": buffer.LastHeartbeat.Format(time.RFC3339),
			},
		})
		buffer.HangLogged = true
		log.Printf("Session %s appears hung", sessionID)
	}
}

// hasStateChanged checks if state has actually changed
func (d *Daemon) hasStateChanged(old, new map[string]interface{}) bool {
	// Simple comparison - check if any keys are different
	for k, newVal := range new {
		oldVal, exists := old[k]
		if !exists {
			return true
		}
		if !d.deepEqual(oldVal, newVal) {
			return true
		}
	}
	return false
}

// deepEqual checks if two values are equal (simple version)
func (d *Daemon) deepEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// mergeState deep merges new state into old state
func (d *Daemon) mergeState(old, new map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy old state
	for k, v := range old {
		result[k] = v
	}

	// Merge new state
	for k, v := range new {
		if oldVal, exists := result[k]; exists {
			if oldMap, ok := oldVal.(map[string]interface{}); ok {
				if newMap, ok := v.(map[string]interface{}); ok {
					result[k] = d.mergeState(oldMap, newMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

// parentProcessMonitor checks if parent processes are still alive
func (d *Daemon) parentProcessMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.RLock()
		sessionIDs := make([]string, 0, len(d.sessions))
		for id := range d.sessions {
			sessionIDs = append(sessionIDs, id)
		}
		d.mu.RUnlock()

		for _, sessionID := range sessionIDs {
			d.checkParentProcess(sessionID)
		}
	}
}

// checkParentProcess checks if parent process is still alive
func (d *Daemon) checkParentProcess(sessionID string) {
	d.mu.RLock()
	buffer, exists := d.sessions[sessionID]
	d.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	// Skip if no parent PID set or already logged
	if buffer.ParentPID == 0 || buffer.CrashLogged {
		return
	}

	// Check if process is still running
	if !isProcessRunning(buffer.ParentPID) {
		// Parent process has crashed or exited abnormally
		buffer.Events = append(buffer.Events, EventData{
			Name: "parent_process_crashed",
			Data: map[string]interface{}{
				"parent_pid": buffer.ParentPID,
			},
		})
		buffer.CrashLogged = true
		log.Printf("Session %s: parent process %d crashed/exited", sessionID, buffer.ParentPID)

		// Immediately flush this event
		go d.flushSession(sessionID)
	}
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	// This works on Unix-like systems
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func main() {
	config := LoadConfig("daemon_config.json")
	daemon := NewDaemon(config)

	log.Fatal(daemon.Start())
}
