package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

// StateUpdate represents a state update with timestamp
type StateUpdate struct {
	Timestamp time.Time              `json:"timestamp"` // Daemon timestamp when state update was received from client
	State     map[string]interface{} `json:"state"`
}

// CounterKey uniquely identifies a counter by name and tags
type CounterKey struct {
	Name string
	Tags string // JSON-encoded sorted tags for consistent key
}

// HistogramBucket represents a single bucket in a histogram
type HistogramBucket struct {
	UpperBound float64 `json:"le"`    // "less than or equal to"
	Count      int64   `json:"count"` // Number of samples in this bucket
}

// Histogram tracks a distribution of values using buckets
type Histogram struct {
	Buckets []float64          // Bucket upper bounds (sorted)
	Counts  []int64            // Count per bucket
	Sum     float64            // Sum of all observed values
	Count   int64              // Total number of observations
	Tags    []string           // Tags for this histogram
	Unit    string             // Unit of measurement
	Metadata map[string]interface{} // Additional metadata
}

// Default histogram buckets (similar to Prometheus defaults, covering microseconds to seconds)
var DefaultHistogramBuckets = []float64{
	0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0,
	// +Inf is implicit (all values go into last bucket)
}

// SessionBuffer holds pending updates for a session
type SessionBuffer struct {
	SessionID              string
	StateUpdates           []StateUpdate
	Events                 []EventData
	Metrics                []MetricData
	Counters               map[string]float64       // Counter name+tags -> cumulative value
	Histograms             map[string]*Histogram    // Histogram name+tags+buckets -> histogram
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
	Timestamp      time.Time              `json:"timestamp"`       // Daemon timestamp when event was received from client
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
	Timestamp time.Time              `json:"timestamp"`           // Daemon timestamp when metric was received from client
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`                // "gauge", "counter", "histogram", "timer"
	Value     float64                `json:"value"`
	Count     *int                   `json:"count,omitempty"`     // For timers
	Unit      string                 `json:"unit,omitempty"`      // e.g., "seconds", "milliseconds"
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
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
	sessionCounter int           // Counter for generating local session IDs
	shutdownChan   chan struct{} // Channel to signal shutdown
	httpServer     *http.Server  // HTTP server instance for graceful shutdown
	httpClient     *http.Client  // HTTP client for server requests with TLS config
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) *Daemon {
	// Create HTTP client with TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSInsecureSkipVerify,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Daemon{
		config:         config,
		sessions:       make(map[string]*SessionBuffer),
		failedQueue:    make([]QueuedOperation, 0),
		sessionCounter: 0,
		shutdownChan:   make(chan struct{}),
		httpClient:     httpClient,
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

	// Create HTTP server instance for graceful shutdown
	d.httpServer = &http.Server{
		Addr: addr,
	}

	// Start shutdown monitor
	go d.shutdownMonitor()

	return d.httpServer.ListenAndServe()
}

// postToServer sends a POST request to the server with compression and authentication
func (d *Daemon) postToServer(endpoint string, data []byte) (*http.Response, error) {
	var body io.Reader = bytes.NewBuffer(data)

	// Compress if enabled
	if d.config.EnableCompression {
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)
		if _, err := gzipWriter.Write(data); err != nil {
			return nil, fmt.Errorf("failed to compress: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		body = &buf
	}

	req, err := http.NewRequest("POST", d.config.ServerURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add compression header if enabled
	if d.config.EnableCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}

	// Add API key if configured
	if d.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+d.config.APIKey)
	}

	return d.httpClient.Do(req)
}

// createSessionOnServer attempts to create a session on the server
func (d *Daemon) createSessionOnServer(product, version, machineID, hostname string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"product":    product,
		"version":    version,
		"machine_id": machineID,
		"hostname":   hostname,
	})

	resp, err := d.postToServer("/api/sessions", reqBody)
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
		StateUpdates:     []StateUpdate{},
		Events:           []EventData{},
		Metrics:          []MetricData{},
		Counters:         make(map[string]float64),
		Histograms:       make(map[string]*Histogram),
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
		// Timestamp when daemon receives the state update from client
		now := time.Now()
		buffer.StateUpdates = append(buffer.StateUpdates, StateUpdate{
			Timestamp: now,
			State:     req.State,
		})
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

	// Timestamp when daemon receives the event from client (this is effectively the client log time)
	now := time.Now()

	buffer.mu.Lock()
	buffer.Events = append(buffer.Events, EventData{
		Timestamp:      now,
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

// makeCounterKey creates a unique key for a counter based on name and tags
func makeCounterKey(name string, tags []string) string {
	if len(tags) == 0 {
		return name
	}
	// Sort tags for consistent key
	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)
	tagsJSON, _ := json.Marshal(sortedTags)
	return name + ":" + string(tagsJSON)
}

// makeHistogramKey creates a unique key for a histogram based on name, tags, and buckets
func makeHistogramKey(name string, tags []string, buckets []float64) string {
	key := name
	if len(tags) > 0 {
		sortedTags := make([]string, len(tags))
		copy(sortedTags, tags)
		sort.Strings(sortedTags)
		tagsJSON, _ := json.Marshal(sortedTags)
		key += ":" + string(tagsJSON)
	}
	if len(buckets) > 0 {
		bucketsJSON, _ := json.Marshal(buckets)
		key += ":" + string(bucketsJSON)
	}
	return key
}

// newHistogram creates a new histogram with the given buckets
func newHistogram(buckets []float64, tags []string, unit string, metadata map[string]interface{}) *Histogram {
	if len(buckets) == 0 {
		// Use default buckets
		buckets = make([]float64, len(DefaultHistogramBuckets))
		copy(buckets, DefaultHistogramBuckets)
	} else {
		// Ensure buckets are sorted
		buckets = append([]float64(nil), buckets...)
		sort.Float64s(buckets)
	}

	return &Histogram{
		Buckets:  buckets,
		Counts:   make([]int64, len(buckets)),
		Sum:      0,
		Count:    0,
		Tags:     tags,
		Unit:     unit,
		Metadata: metadata,
	}
}

// observe records a value in the histogram
func (h *Histogram) observe(value float64) {
	h.Count++
	h.Sum += value

	// Find the appropriate bucket (first bucket where value <= upperBound)
	for i, upperBound := range h.Buckets {
		if value <= upperBound {
			h.Counts[i]++
			return
		}
	}
	// Value exceeds all buckets, put in last bucket
	h.Counts[len(h.Counts)-1]++
}

// handleMetric handles metric logging requests
func (d *Daemon) handleMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string                 `json:"session_id"`
		Name      string                 `json:"name"`
		Type      string                 `json:"type"`
		Value     float64                `json:"value"`
		Delta     *float64               `json:"delta,omitempty"` // For counter increments
		Count     *int                   `json:"count,omitempty"`
		Unit      string                 `json:"unit,omitempty"`
		Tags      []string               `json:"tags"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
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

	// Timestamp when daemon receives the metric from client (this is effectively the client log time)
	now := time.Now()

	// Default to gauge if not specified (backward compatibility)
	metricType := req.Type
	if metricType == "" {
		metricType = "gauge"
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	// Handle counters specially - accumulate deltas instead of storing each value
	if metricType == "counter" && req.Delta != nil {
		// Incremental counter - accumulate in daemon
		key := makeCounterKey(req.Name, req.Tags)
		if buffer.Counters == nil {
			buffer.Counters = make(map[string]float64)
		}
		buffer.Counters[key] += *req.Delta
	} else if metricType == "histogram" {
		// Histogram - accumulate samples into buckets
		if buffer.Histograms == nil {
			buffer.Histograms = make(map[string]*Histogram)
		}

		// Extract buckets from metadata if provided
		var buckets []float64
		if req.Metadata != nil {
			if bucketsInterface, ok := req.Metadata["buckets"]; ok {
				// Handle both []interface{} and []float64
				switch v := bucketsInterface.(type) {
				case []interface{}:
					buckets = make([]float64, 0, len(v))
					for _, b := range v {
						switch num := b.(type) {
						case float64:
							buckets = append(buckets, num)
						case int:
							buckets = append(buckets, float64(num))
						case string:
							// Handle "inf" string
							if num == "inf" || num == "Infinity" {
								buckets = append(buckets, math.Inf(1))
							}
						}
					}
				case []float64:
					buckets = v
				}
			}
		}

		key := makeHistogramKey(req.Name, req.Tags, buckets)
		hist, exists := buffer.Histograms[key]
		if !exists {
			hist = newHistogram(buckets, req.Tags, req.Unit, req.Metadata)
			buffer.Histograms[key] = hist
		}
		hist.observe(req.Value)
	} else {
		// All other metrics (gauge, timer) or absolute counter values
		buffer.Metrics = append(buffer.Metrics, MetricData{
			Timestamp: now,
			Name:      req.Name,
			Type:      metricType,
			Value:     req.Value,
			Count:     req.Count,
			Unit:      req.Unit,
			Tags:      req.Tags,
			Metadata:  req.Metadata,
		})
	}

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
		remainingSessions := len(d.sessions)
		d.mu.Unlock()

		// If no sessions remain, trigger shutdown
		if remainingSessions == 0 {
			log.Printf("All sessions ended, initiating daemon shutdown")
			// Signal shutdown in a goroutine to avoid blocking the response
			go func() {
				time.Sleep(2 * time.Second) // Give time for response to be sent
				select {
				case d.shutdownChan <- struct{}{}:
				default:
				}
			}()
		}
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

// parseCounterKey extracts name and tags from a counter key
func parseCounterKey(key string) (string, []string) {
	parts := strings.SplitN(key, ":", 2)
	name := parts[0]
	var tags []string
	if len(parts) > 1 {
		json.Unmarshal([]byte(parts[1]), &tags)
	}
	return name, tags
}

// parseHistogramKey extracts name, tags, and buckets from a histogram key
func parseHistogramKey(key string) (string, []string, []float64) {
	parts := strings.Split(key, ":")
	name := parts[0]
	var tags []string
	var buckets []float64

	if len(parts) > 1 {
		json.Unmarshal([]byte(parts[1]), &tags)
	}
	if len(parts) > 2 {
		json.Unmarshal([]byte(parts[2]), &buckets)
	}

	return name, tags, buckets
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

	// Convert accumulated counters to metrics (but keep counter state)
	now := time.Now()
	counterMetrics := make([]MetricData, 0, len(buffer.Counters))
	for key, value := range buffer.Counters {
		name, tags := parseCounterKey(key)
		counterMetrics = append(counterMetrics, MetricData{
			Timestamp: now,
			Name:      name,
			Type:      "counter",
			Value:     value,
			Tags:      tags,
		})
	}
	// Don't reset counters - they keep accumulating!

	// Convert accumulated histograms to metrics (but keep histogram state)
	histogramMetrics := make([]MetricData, 0, len(buffer.Histograms))
	for key, hist := range buffer.Histograms {
		name, _, _ := parseHistogramKey(key)

		// Build buckets array for server
		buckets := make([]HistogramBucket, len(hist.Buckets))
		for i, upperBound := range hist.Buckets {
			buckets[i] = HistogramBucket{
				UpperBound: upperBound,
				Count:      hist.Counts[i],
			}
		}

		// Store histogram as metric with special metadata
		metadata := map[string]interface{}{
			"buckets": buckets,
			"sum":     hist.Sum,
			"count":   hist.Count,
		}
		// Merge any original metadata
		for k, v := range hist.Metadata {
			if k != "buckets" { // Don't override our structured buckets
				metadata[k] = v
			}
		}

		histogramMetrics = append(histogramMetrics, MetricData{
			Timestamp: now,
			Name:      name,
			Type:      "histogram",
			Value:     hist.Sum / float64(hist.Count), // Average as the value
			Tags:      hist.Tags,
			Unit:      hist.Unit,
			Metadata:  metadata,
		})
	}
	// Don't reset histograms - they keep accumulating!

	buffer.StateUpdates = make([]StateUpdate, 0)
	buffer.Events = make([]EventData, 0)
	buffer.Metrics = make([]MetricData, 0)
	buffer.mu.Unlock()

	// Send state updates
	for _, stateUpdate := range stateUpdates {
		d.sendStateUpdate(sessionID, stateUpdate)
	}

	// Send events
	for _, event := range events {
		d.sendEvent(sessionID, event)
	}

	// Send regular metrics
	for _, metric := range metrics {
		d.sendMetric(sessionID, metric)
	}

	// Send accumulated counter totals
	for _, metric := range counterMetrics {
		d.sendMetric(sessionID, metric)
	}

	// Send accumulated histogram buckets
	for _, metric := range histogramMetrics {
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
	resp, err := d.postToServer(endpoint, jsonData)
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
func (d *Daemon) sendStateUpdate(sessionID string, stateUpdate StateUpdate) {
	d.sendToServer(sessionID, "/api/sessions/"+sessionID+"/state", "state", stateUpdate)
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
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/heartbeat", []byte("{}"))
	if err != nil {
		log.Printf("Failed to send heartbeat to server: %v", err)
		return
	}
	defer resp.Body.Close()
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
	parentPID := buffer.ParentPID
	crashLogged := buffer.CrashLogged
	endedAt := buffer.EndedAt
	buffer.mu.Unlock()

	// Skip if no parent PID set, already logged, or session already ended
	if parentPID == 0 || crashLogged || endedAt != nil {
		return
	}

	// Check if process is still running
	if !isProcessRunning(parentPID) {
		// Parent process has crashed or exited abnormally
		buffer.mu.Lock()
		buffer.Events = append(buffer.Events, EventData{
			Name:     "parent_process_crashed",
			Level:    "critical",
			Category: "datacat.daemon",
			Labels:   []string{"crash", "process"},
			Message:  fmt.Sprintf("Parent process (PID %d) is no longer running", parentPID),
			Data: map[string]interface{}{
				"parent_pid": parentPID,
			},
		})
		buffer.CrashLogged = true
		now := time.Now()
		buffer.EndedAt = &now
		buffer.Active = false
		buffer.mu.Unlock()

		log.Printf("Session %s: parent process %d crashed/exited, marking as crashed", sessionID, parentPID)

		// Immediately flush this event
		d.flushSession(sessionID)

		// Mark session as crashed on server
		go func() {
		crashData, _ := json.Marshal(map[string]interface{}{
			"reason": "parent_process_terminated",
		})
		resp, err := d.postToServer("/api/sessions/"+sessionID+"/crash", crashData)
		if err != nil {
			log.Printf("Failed to mark session as crashed on server: %v", err)
			// Queue for retry
			d.queueMu.Lock()
			d.failedQueue = append(d.failedQueue, QueuedOperation{
				SessionID: sessionID,
				OpType:    "crash",
				Data:      map[string]interface{}{"reason": "parent_process_terminated"},
				Timestamp: time.Now(),
			})
			d.queueMu.Unlock()
		} else {
			resp.Body.Close()
				// Remove session from daemon after successfully marking as crashed
				d.mu.Lock()
				delete(d.sessions, sessionID)
				remainingSessions := len(d.sessions)
				d.mu.Unlock()

				log.Printf("Session %s removed after parent process crash", sessionID)

				// If no sessions remain, trigger shutdown
				if remainingSessions == 0 {
					log.Printf("All sessions ended, initiating daemon shutdown")
					time.Sleep(1 * time.Second)
					select {
					case d.shutdownChan <- struct{}{}:
					default:
					}
				}
			}
		}()
	}
}

// isProcessRunning checks if a process with the given PID is running
// This function is cross-platform compatible (Windows and Unix-like systems)
func isProcessRunning(pid int) bool {
	// Platform-specific process checking
	if runtime.GOOS == "windows" {
		// On Windows, use tasklist command to check if process exists
		// This is more reliable than os.FindProcess which always succeeds
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error checking process %d: %v", pid, err)
			return false
		}

		// If process exists, output will contain the PID
		// If not, output will be empty or contain "INFO: No tasks are running..."
		outputStr := strings.TrimSpace(string(output))
		if outputStr == "" || strings.Contains(outputStr, "INFO:") {
			return false
		}

		// Parse CSV output to verify PID matches
		// Format: "ImageName","PID","SessionName","SessionNumber","MemUsage"
		parts := strings.Split(outputStr, ",")
		if len(parts) >= 2 {
			// Remove quotes from PID field
			pidStr := strings.Trim(parts[1], "\"")
			foundPID, err := strconv.Atoi(pidStr)
			if err == nil && foundPID == pid {
				return true
			}
		}
		return false
	}

	// On Unix systems, use kill -0 signal to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// shutdownMonitor waits for shutdown signal and performs graceful shutdown
func (d *Daemon) shutdownMonitor() {
	<-d.shutdownChan
	log.Printf("Shutdown signal received, beginning graceful shutdown...")

	// Give time for any pending operations to complete
	time.Sleep(1 * time.Second)

	// Shut down the HTTP server
	if d.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}

	log.Printf("Daemon shutdown complete")
	os.Exit(0)
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
			if stateUpdate, ok := op.Data.(StateUpdate); ok {
				success = d.retrySendState(op.SessionID, stateUpdate)
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
		case "crash":
			if data, ok := op.Data.(map[string]interface{}); ok {
				success = d.retryCrashSession(op.SessionID, data)
			}
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

	resp, err := d.postToServer("/api/sessions", reqBody)
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
func (d *Daemon) retrySendState(sessionID string, stateUpdate StateUpdate) bool {
	data, _ := json.Marshal(stateUpdate)
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/state", data)
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
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/events", data)
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
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/metrics", data)
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
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/end", endData)
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

// retryCrashSession retries marking a session as crashed
func (d *Daemon) retryCrashSession(sessionID string, data map[string]interface{}) bool {
	crashData, _ := json.Marshal(data)
	resp, err := d.postToServer("/api/sessions/"+sessionID+"/crash", crashData)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to mark session as crashed on retry (status %d): %s", resp.StatusCode, string(body))
		return false
	}

	// Remove session from daemon after successfully marking as crashed
	d.mu.Lock()
	delete(d.sessions, sessionID)
	d.mu.Unlock()

	log.Printf("Session %s successfully marked as crashed on server (from queue)", sessionID)
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
	// Support instance-specific config files via environment variable
	configPath := os.Getenv("DATACAT_CONFIG")
	if configPath == "" {
		configPath = "daemon_config.json"
	}

	config := LoadConfig(configPath)
	daemon := NewDaemon(config)

	log.Fatal(daemon.Start())
}
