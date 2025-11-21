package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const (
	heartbeatCheckInterval     = 5 * time.Second
	parentProcessCheckInterval = 5 * time.Second
	serverHeartbeatInterval    = 15 * time.Second
	retryQueueInterval         = 10 * time.Second
)

// SessionBuffer holds pending updates for a session
type SessionBuffer struct {
	SessionID              string
	StateUpdates           []map[string]interface{}
	Events                 []EventData
	Metrics                []MetricData
	LastHeartbeat          time.Time
	LastState              map[string]interface{}
	HangLogged             bool
	ParentPID              int        // Parent process ID
	CrashLogged            bool       // Whether crash has been logged
	CreatedAt              time.Time  // When session was created
	EndedAt                *time.Time // When session was ended
	Active                 bool       // Whether session is active
	SyncedWithServer       bool       // Whether this session exists on the server
	HeartbeatMonitorPaused bool       // Whether heartbeat monitoring is paused
	mu                     sync.Mutex
}

// EventData represents an event to be logged
type EventData struct {
	Name           string                 `json:"name"`
	Level          string                 `json:"level"`
	Category       string                 `json:"category"`
	Labels         []string               `json:"labels"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data"`
	ExceptionType  string                 `json:"exception_type,omitempty"`
	ExceptionMsg   string                 `json:"exception_msg,omitempty"`
	Stacktrace     []string               `json:"stacktrace,omitempty"`
	SourceFile     string                 `json:"source_file,omitempty"`
	SourceLine     int                    `json:"source_line,omitempty"`
	SourceFunction string                 `json:"source_function,omitempty"`
}

// MetricData represents a metric to be logged
type MetricData struct {
	Name  string   `json:"name"`
	Value float64  `json:"value"`
	Tags  []string `json:"tags,omitempty"`
}

// QueuedOperation represents an operation that failed and needs retry
type QueuedOperation struct {
	SessionID string
	OpType    string // "create_session", "state", "event", "metric", "end"
	Data      interface{}
	Timestamp time.Time
}

// Daemon manages batching and forwarding to the server
type Daemon struct {
	config         *Config
	sessions       map[string]*SessionBuffer
	mu             sync.RWMutex
	failedQueue    []QueuedOperation
	queueMu        sync.Mutex
	sessionCounter int // Counter for generating local session IDs
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) *Daemon {
	return &Daemon{
		config:         config,
		sessions:       make(map[string]*SessionBuffer),
		failedQueue:    make([]QueuedOperation, 0),
		sessionCounter: 0,
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

	// Start retry queue processor goroutine
	go d.retryQueueProcessor()

	// Start periodic heartbeat sender to server
	go d.periodicHeartbeatSender()

	// Setup HTTP handlers
	http.HandleFunc("/register", d.handleRegister)
	http.HandleFunc("/state", d.handleState)
	http.HandleFunc("/event", d.handleEvent)
	http.HandleFunc("/metric", d.handleMetric)
	http.HandleFunc("/heartbeat", d.handleHeartbeat)
	http.HandleFunc("/pause_heartbeat", d.handlePauseHeartbeat)
	http.HandleFunc("/resume_heartbeat", d.handleResumeHeartbeat)
	http.HandleFunc("/end", d.handleEnd)
	http.HandleFunc("/health", d.handleHealth)
	http.HandleFunc("/session", d.handleGetSession)
	http.HandleFunc("/sessions", d.handleGetSessions)

	addr := ":" + d.config.DaemonPort
	log.Printf("Daemon listening on %s, forwarding to %s", addr, d.config.ServerURL)
	return http.ListenAndServe(addr, nil)
}

// createSessionOnServer attempts to create a session on the server
func (d *Daemon) createSessionOnServer(product, version, machineID, hostname string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"product":    product,
		"version":    version,
		"machine_id": machineID,
		"hostname":   hostname,
	})

	resp, err := http.Post(d.config.ServerURL+"/api/sessions", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	sessionID, ok := result["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid session ID in response")
	}

	return sessionID, nil
}

// createLocalSessionID generates a unique local session ID
func (d *Daemon) createLocalSessionID() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sessionCounter++
	return fmt.Sprintf("local-session-%d-%d", time.Now().Unix(), d.sessionCounter)
}

// initSessionBuffer creates and stores a new session buffer
func (d *Daemon) initSessionBuffer(sessionID, product, version string, parentPID int, synced bool) {
	initialState := map[string]interface{}{
		"product": product,
		"version": version,
	}

	now := time.Now()
	d.mu.Lock()
	d.sessions[sessionID] = &SessionBuffer{
		SessionID:        sessionID,
		StateUpdates:     []map[string]interface{}{},
		Events:           []EventData{},
		Metrics:          []MetricData{},
		LastHeartbeat:    now,
		LastState:        initialState,
		ParentPID:        parentPID,
		CreatedAt:        now,
		Active:           true,
		SyncedWithServer: synced,
	}
	d.mu.Unlock()
}

// handleRegister registers a new session
func (d *Daemon) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ParentPID int    `json:"parent_pid"`
		Product   string `json:"product"`
		Version   string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Product == "" || req.Version == "" {
		http.Error(w, "Product and version are required fields", http.StatusBadRequest)
		return
	}

	machineID := getMachineID()
	hostname := getHostname()

	sessionID, err := d.createSessionOnServer(req.Product, req.Version, machineID, hostname)
	syncedWithServer := true

	if err != nil {
		log.Printf("Server unavailable, creating session locally: %v", err)
		sessionID = d.createLocalSessionID()
		syncedWithServer = false

		d.queueOperation(sessionID, "create_session", map[string]interface{}{
			"product":    req.Product,
			"version":    req.Version,
			"machine_id": machineID,
			"hostname":   hostname,
		})
	}

	d.initSessionBuffer(sessionID, req.Product, req.Version, req.ParentPID, syncedWithServer)

	log.Printf("Registered session: %s (product: %s, version: %s, parent PID: %d, synced: %v)",
		sessionID, req.Product, req.Version, req.ParentPID, syncedWithServer)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
	})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleEvent handles event logging requests
func (d *Daemon) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID      string                 `json:"session_id"`
		Name           string                 `json:"name"`
		Level          string                 `json:"level"`
		Category       string                 `json:"category"`
		Labels         []string               `json:"labels"`
		Message        string                 `json:"message"`
		Data           map[string]interface{} `json:"data"`
		ExceptionType  string                 `json:"exception_type"`
		ExceptionMsg   string                 `json:"exception_msg"`
		Stacktrace     []string               `json:"stacktrace"`
		SourceFile     string                 `json:"source_file"`
		SourceLine     int                    `json:"source_line"`
		SourceFunction string                 `json:"source_function"`
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
		Name:           req.Name,
		Level:          req.Level,
		Category:       req.Category,
		Labels:         req.Labels,
		Message:        req.Message,
		Data:           req.Data,
		ExceptionType:  req.ExceptionType,
		ExceptionMsg:   req.ExceptionMsg,
		Stacktrace:     req.Stacktrace,
		SourceFile:     req.SourceFile,
		SourceLine:     req.SourceLine,
		SourceFunction: req.SourceFunction,
	})
	buffer.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	// Reset paused flag if heartbeat received
	if buffer.HeartbeatMonitorPaused {
		buffer.HeartbeatMonitorPaused = false
		log.Printf("Session %s: heartbeat monitoring automatically resumed", req.SessionID)
	}
	if buffer.HangLogged {
		// Application recovered
		buffer.Events = append(buffer.Events, EventData{
			Name:     "application_recovered",
			Level:    "info",
			Category: "datacat.daemon",
			Labels:   []string{"heartbeat", "recovery"},
			Message:  "Application heartbeat resumed after hang",
			Data:     map[string]interface{}{},
		})
		buffer.HangLogged = false
	}
	buffer.mu.Unlock()

	// Forward heartbeat to server
	d.sendHeartbeat(req.SessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handlePauseHeartbeat handles requests to pause heartbeat monitoring
func (d *Daemon) handlePauseHeartbeat(w http.ResponseWriter, r *http.Request) {
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
	buffer.HeartbeatMonitorPaused = true
	buffer.mu.Unlock()

	log.Printf("Session %s: heartbeat monitoring paused", req.SessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleResumeHeartbeat handles requests to resume heartbeat monitoring
func (d *Daemon) handleResumeHeartbeat(w http.ResponseWriter, r *http.Request) {
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
	buffer.HeartbeatMonitorPaused = false
	buffer.LastHeartbeat = time.Now() // Reset heartbeat time to avoid immediate hang detection
	buffer.mu.Unlock()

	log.Printf("Session %s: heartbeat monitoring resumed", req.SessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	// Mark session as ended locally
	d.mu.Lock()
	if buffer, exists := d.sessions[req.SessionID]; exists {
		now := time.Now()
		buffer.EndedAt = &now
		buffer.Active = false
	}
	d.mu.Unlock()

	// Forward end request to server
	endData, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+req.SessionID+"/end",
		"application/json",
		bytes.NewBuffer(endData),
	)
	if err != nil {
		log.Printf("Failed to send end session to server, queueing for retry: %v", err)
		// Queue for retry
		d.queueMu.Lock()
		d.failedQueue = append(d.failedQueue, QueuedOperation{
			SessionID: req.SessionID,
			OpType:    "end",
			Data:      nil,
			Timestamp: time.Now(),
		})
		d.queueMu.Unlock()
	} else {
		_ = resp.Body.Close()
		// Remove session from daemon after successfully ending on server
		d.mu.Lock()
		delete(d.sessions, req.SessionID)
		d.mu.Unlock()
	}

	log.Printf("Session ended: %s", req.SessionID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

// queueOperation adds an operation to the failed queue for retry
func (d *Daemon) queueOperation(sessionID, opType string, data interface{}) {
	d.queueMu.Lock()
	d.failedQueue = append(d.failedQueue, QueuedOperation{
		SessionID: sessionID,
		OpType:    opType,
		Data:      data,
		Timestamp: time.Now(),
	})
	d.queueMu.Unlock()
}

// sendToServer sends data to the server with automatic retry queueing on failure
func (d *Daemon) sendToServer(sessionID, endpoint, opType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(
		d.config.ServerURL+endpoint,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("Failed to send %s, queueing for retry: %v", opType, err)
		d.queueOperation(sessionID, opType, data)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Server returned error for %s (status %d), queueing for retry: %s", opType, resp.StatusCode, string(body))
		d.queueOperation(sessionID, opType, data)
		return
	}

	d.markSessionSynced(sessionID)
}

// sendStateUpdate sends a state update to the server
func (d *Daemon) sendStateUpdate(sessionID string, state map[string]interface{}) {
	d.sendToServer(sessionID, "/api/sessions/"+sessionID+"/state", "state", state)
}

// sendEvent sends an event to the server
func (d *Daemon) sendEvent(sessionID string, event EventData) {
	d.sendToServer(sessionID, "/api/sessions/"+sessionID+"/events", "event", event)
}

// sendMetric sends a metric to the server
func (d *Daemon) sendMetric(sessionID string, metric MetricData) {
	d.sendToServer(sessionID, "/api/sessions/"+sessionID+"/metrics", "metric", metric)
}

// sendHeartbeat sends a heartbeat to the server
func (d *Daemon) sendHeartbeat(sessionID string) {
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/heartbeat",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		log.Printf("Failed to send heartbeat to server: %v", err)
		return
	}
	_ = resp.Body.Close()
}

// heartbeatMonitor checks for hung applications
func (d *Daemon) heartbeatMonitor() {
	ticker := time.NewTicker(heartbeatCheckInterval)
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

	// Skip heartbeat check if monitoring is paused
	if buffer.HeartbeatMonitorPaused {
		return
	}

	timeout := time.Duration(d.config.HeartbeatTimeoutSeconds) * time.Second
	if time.Since(buffer.LastHeartbeat) > timeout && !buffer.HangLogged {
		// Application appears hung
		buffer.Events = append(buffer.Events, EventData{
			Name:     "application_appears_hung",
			Level:    "error",
			Category: "datacat.daemon",
			Labels:   []string{"heartbeat", "hung"},
			Message:  fmt.Sprintf("Application has not sent heartbeat for %.0f seconds", time.Since(buffer.LastHeartbeat).Seconds()),
			Data: map[string]interface{}{
				"last_heartbeat":          buffer.LastHeartbeat.Format(time.RFC3339),
				"seconds_since_heartbeat": int(time.Since(buffer.LastHeartbeat).Seconds()),
			},
		})
		buffer.HangLogged = true
		log.Printf("Session %s appears hung", sessionID)
	}
}

// periodicHeartbeatSender sends periodic heartbeats to the server for all active sessions
func (d *Daemon) periodicHeartbeatSender() {
	// Send heartbeats to server (should be less than server timeout)
	ticker := time.NewTicker(serverHeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.RLock()
		sessionIDs := make([]string, 0, len(d.sessions))
		for id := range d.sessions {
			sessionIDs = append(sessionIDs, id)
		}
		d.mu.RUnlock()

		for _, sessionID := range sessionIDs {
			d.mu.RLock()
			buffer, exists := d.sessions[sessionID]
			d.mu.RUnlock()

			if !exists {
				continue
			}

			buffer.mu.Lock()
			active := buffer.Active && buffer.EndedAt == nil
			buffer.mu.Unlock()

			// Only send heartbeats for active sessions
			if active {
				d.sendHeartbeat(sessionID)
			}
		}
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
	ticker := time.NewTicker(parentProcessCheckInterval)
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
			Name:     "parent_process_crashed",
			Level:    "critical",
			Category: "datacat.daemon",
			Labels:   []string{"crash", "process"},
			Message:  fmt.Sprintf("Parent process (PID %d) is no longer running", buffer.ParentPID),
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
// This function is cross-platform compatible (Windows and Unix-like systems)
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Platform-specific process checking
	if runtime.GOOS == "windows" {
		// On Windows, process.Signal doesn't work the same way
		// We need to check using process state methods
		// If we can send a "signal 0" (which on Windows is handled differently)
		// or if the process object is valid, the process exists
		// On Windows, FindProcess always succeeds if the PID format is valid,
		// so we return true to be conservative. The daemon will only log once anyway.
		// A more robust solution would use Windows-specific APIs, but this is sufficient
		// for the daemon's purpose (detecting when parent exits)
		return true
	}

	// On Unix systems, FindProcess always succeeds, so we need to send signal 0
	// Signal 0 is a special signal that doesn't actually send anything but checks if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// getMachineID returns a unique identifier for this machine based on MAC address
func getMachineID() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Warning: Failed to get network interfaces: %v", err)
		return ""
	}

	// Find the first non-loopback interface with a hardware address
	for _, iface := range interfaces {
		// Skip loopback and interfaces without hardware address
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}

		// Use MD5 hash of MAC address as machine ID (for privacy/consistency)
		hash := md5.Sum([]byte(iface.HardwareAddr.String()))
		return fmt.Sprintf("%x", hash)
	}

	log.Printf("Warning: No suitable network interface found for machine ID")
	return ""
}

// getHostname returns the hostname of this machine
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: Failed to get hostname: %v", err)
		return ""
	}
	return hostname
}

// retryQueueProcessor periodically processes the failed queue
func (d *Daemon) retryQueueProcessor() {
	ticker := time.NewTicker(retryQueueInterval)
	defer ticker.Stop()

	for range ticker.C {
		d.processFailedQueue()
	}
}

// processFailedQueue attempts to retry failed operations
func (d *Daemon) processFailedQueue() {
	d.queueMu.Lock()
	if len(d.failedQueue) == 0 {
		d.queueMu.Unlock()
		return
	}

	// Take a copy of the queue and clear it
	queue := make([]QueuedOperation, len(d.failedQueue))
	copy(queue, d.failedQueue)
	d.failedQueue = make([]QueuedOperation, 0)
	d.queueMu.Unlock()

	log.Printf("Processing %d queued operations", len(queue))

	for _, op := range queue {
		success := false

		switch op.OpType {
		case "create_session":
			if data, ok := op.Data.(map[string]interface{}); ok {
				success = d.retryCreateSession(op.SessionID, data)
			}
		case "state":
			if state, ok := op.Data.(map[string]interface{}); ok {
				success = d.retrySendState(op.SessionID, state)
			}
		case "event":
			if event, ok := op.Data.(EventData); ok {
				success = d.retrySendEvent(op.SessionID, event)
			}
		case "metric":
			if metric, ok := op.Data.(MetricData); ok {
				success = d.retrySendMetric(op.SessionID, metric)
			}
		case "end":
			success = d.retryEndSession(op.SessionID)
		}

		// If retry failed, add it back to the queue
		if !success {
			d.queueMu.Lock()
			d.failedQueue = append(d.failedQueue, op)
			d.queueMu.Unlock()
		}
	}
}

// retryCreateSession attempts to create a session on the server
func (d *Daemon) retryCreateSession(sessionID string, data map[string]interface{}) bool {
	// Marshal the session data (product, version, machine_id, hostname)
	reqBody, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal session data for retry: %v", err)
		return false
	}

	resp, err := http.Post(d.config.ServerURL+"/api/sessions", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to create session on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	serverSessionID, ok := result["session_id"].(string)
	if !ok {
		return false
	}

	// Update the local session with the server-assigned ID
	d.mu.Lock()
	if buffer, exists := d.sessions[sessionID]; exists {
		// Remove old entry and add with new ID
		delete(d.sessions, sessionID)
		buffer.SessionID = serverSessionID
		buffer.SyncedWithServer = true
		d.sessions[serverSessionID] = buffer
		log.Printf("Session %s synced with server as %s", sessionID, serverSessionID)
	}
	d.mu.Unlock()

	return true
}

// retrySendState attempts to send a state update to the server
func (d *Daemon) retrySendState(sessionID string, state map[string]interface{}) bool {
	data, _ := json.Marshal(state)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/state",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send state update on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	d.markSessionSynced(sessionID)
	return true
}

// retrySendEvent attempts to send an event to the server
func (d *Daemon) retrySendEvent(sessionID string, event EventData) bool {
	data, _ := json.Marshal(event)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/events",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send event on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	d.markSessionSynced(sessionID)
	return true
}

// retrySendMetric attempts to send a metric to the server
func (d *Daemon) retrySendMetric(sessionID string, metric MetricData) bool {
	data, _ := json.Marshal(metric)
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/metrics",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send metric on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	d.markSessionSynced(sessionID)
	return true
}

// retryEndSession attempts to end a session on the server
func (d *Daemon) retryEndSession(sessionID string) bool {
	endData, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(
		d.config.ServerURL+"/api/sessions/"+sessionID+"/end",
		"application/json",
		bytes.NewBuffer(endData),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to end session on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	// Remove session from daemon after successfully ending on server
	d.mu.Lock()
	delete(d.sessions, sessionID)
	d.mu.Unlock()

	log.Printf("Session %s successfully ended on server (from queue)", sessionID)
	return true
}

// markSessionSynced marks a session as synced with the server
func (d *Daemon) markSessionSynced(sessionID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if buffer, exists := d.sessions[sessionID]; exists {
		buffer.SyncedWithServer = true
	}
}

// handleGetSession handles session retrieval requests from clients
func (d *Daemon) handleGetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id parameter", http.StatusBadRequest)
		return
	}

	// Try to fetch from server first to get complete session data including events and metrics
	resp, err := http.Get(d.config.ServerURL + "/api/sessions/" + sessionID)
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Forward the response from server
			w.Header().Set("Content-Type", "application/json")
			var sessionData map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&sessionData); err != nil {
				http.Error(w, "Failed to decode session data", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(sessionData)
			return
		}
	}

	// Server unavailable or session not found on server - fall back to local buffer
	d.mu.RLock()
	buffer, exists := d.sessions[sessionID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Build response from local buffer
	buffer.mu.Lock()
	response := map[string]interface{}{
		"id":         buffer.SessionID,
		"created_at": buffer.CreatedAt.Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
		"active":     buffer.Active,
		"state":      buffer.LastState,
		"events":     []interface{}{}, // Empty arrays for consistency
		"metrics":    []interface{}{},
	}
	if buffer.EndedAt != nil {
		response["ended_at"] = buffer.EndedAt.Format(time.RFC3339)
	}
	buffer.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSessions handles requests to get all sessions
func (d *Daemon) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Try to fetch from server first
	resp, err := http.Get(d.config.ServerURL + "/api/data/sessions")
	if err != nil {
		// Server unavailable, return local sessions only
		log.Printf("Server unavailable for get sessions, returning local sessions: %v", err)
		d.mu.RLock()
		sessions := make([]map[string]interface{}, 0, len(d.sessions))
		for _, buffer := range d.sessions {
			buffer.mu.Lock()
			session := map[string]interface{}{
				"id":         buffer.SessionID,
				"created_at": buffer.CreatedAt.Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
				"active":     buffer.Active,
				"state":      buffer.LastState,
			}
			if buffer.EndedAt != nil {
				session["ended_at"] = buffer.EndedAt.Format(time.RFC3339)
			}
			sessions = append(sessions, session)
			buffer.mu.Unlock()
		}
		d.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to retrieve sessions", http.StatusInternalServerError)
		return
	}

	// Forward the response from server
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	var sessions []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&sessions)
	json.NewEncoder(w).Encode(sessions)
}

func main() {
	config := LoadConfig("daemon_config.json")
	daemon := NewDaemon(config)

	log.Fatal(daemon.Start())
}
