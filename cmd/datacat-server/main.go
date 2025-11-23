package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
)

const (
	eventHangDetected   = "application_appears_hung"
	eventHangRecovered  = "application_recovered"
	defaultEventLevel   = "info"
	exceptionEventLevel = "error"
)

// Session represents a registered session
type Session struct {
	ID            string                 `json:"id"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
	LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
	Active        bool                   `json:"active"`
	Suspended     bool                   `json:"suspended"`            // True when heartbeats stopped but likely asleep/hibernating
	Crashed       bool                   `json:"crashed"`              // True when machine came back but session didn't resume
	Hung          bool                   `json:"hung"`                 // True if session ever had a hang event
	MachineID     string                 `json:"machine_id,omitempty"` // Unique machine identifier
	Hostname      string                 `json:"hostname,omitempty"`   // Machine hostname for display
	State         map[string]interface{} `json:"state"`
	StateHistory  []StateSnapshot        `json:"state_history"`
	Events        []Event                `json:"events"`
	Metrics       []Metric               `json:"metrics"`
}

// StateSnapshot represents the state at a specific point in time
type StateSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	State     map[string]interface{} `json:"state"`
}

// Event represents an event logged in a session
type Event struct {
	Timestamp  time.Time              `json:"timestamp"`
	Name       string                 `json:"name"`
	Category   string                 `json:"category"`             // User-defined category (e.g., debug, info, warning, error, critical, or custom)
	Group      string                 `json:"group"`                // Group/logger name (e.g., logger name, component name)
	Labels     []string               `json:"labels"`               // arbitrary tags for filtering
	Message    string                 `json:"message"`              // human-readable message
	Data       map[string]interface{} `json:"data"`                 // additional structured data
	Stacktrace []string               `json:"stacktrace,omitempty"` // Stack trace for any event (not just exceptions)

	// Exception-specific fields (when this is an exception event)
	ExceptionType  string `json:"exception_type,omitempty"`  // e.g., "ValueError", "NullPointerException"
	ExceptionMsg   string `json:"exception_msg,omitempty"`   // exception message
	SourceFile     string `json:"source_file,omitempty"`     // file where exception occurred
	SourceLine     int    `json:"source_line,omitempty"`     // line number where exception occurred
	SourceFunction string `json:"source_function,omitempty"` // function where exception occurred
}

// Metric represents a metric logged in a session
type Metric struct {
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // "gauge", "counter", "histogram", "timer"
	Value     float64                `json:"value"`
	Count     *int                   `json:"count,omitempty"` // For timers: number of iterations
	Unit      string                 `json:"unit,omitempty"`  // e.g., "seconds", "milliseconds", "bytes"
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Additional data for histograms, etc.
}

// Store manages all sessions with BadgerDB for persistence
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	db       *badger.DB
	config   *Config
}

// NewStore creates a new Store with BadgerDB
func NewStore(dbPath string, config *Config) (*Store, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // Disable BadgerDB logs

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	store := &Store{
		sessions: make(map[string]*Session),
		db:       db,
		config:   config,
	}

	// Load existing sessions from database
	if err := store.loadFromDB(); err != nil {
		return nil, fmt.Errorf("failed to load sessions: %v", err)
	}

	return store, nil
}

// Close closes the database
func (s *Store) Close() error {
	return s.db.Close()
}

// saveSessionToDB saves a single session to the database
func (s *Store) saveSessionToDB(session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	return s.saveSessionDataToDB(session.ID, data)
}

// saveSessionDataToDB saves pre-marshaled session data to the database
// This is used when marshaling is done while holding a lock to avoid race conditions
func (s *Store) saveSessionDataToDB(sessionID string, data []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("session:"+sessionID), data)
	})

	if err != nil {
		log.Printf("ERROR: Failed to save session %s to database: %v", sessionID, err)
		return fmt.Errorf("failed to save session to db: %v", err)
	}

	return nil
}

// loadFromDB loads all sessions from the database
func (s *Store) loadFromDB() error {
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("session:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var session Session
				if err := json.Unmarshal(val, &session); err != nil {
					return err
				}
				s.sessions[session.ID] = &session
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Loaded %d sessions from database", len(s.sessions))
	return nil
}

// CreateSession creates a new session with product and version
func (s *Store) CreateSession(product, version, machineID, hostname string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for suspended sessions from the same machine
	// If found, mark them as crashed (machine came back but session didn't resume)
	if machineID != "" {
		for _, existing := range s.sessions {
			// Update active status to check current suspension state
			s.updateActiveStatus(existing)

			if existing.MachineID == machineID && existing.Suspended && !existing.Crashed {
				existing.Crashed = true
				existing.Suspended = false
				log.Printf("Marked session %s as crashed (machine %s came back but session didn't resume)",
					existing.ID, machineID)

				// Add crash event
				existing.Events = append(existing.Events, Event{
					Timestamp: time.Now(),
					Name:      "session_crashed_detected",
					Data: map[string]interface{}{
						"reason":     "machine_returned_session_not_resumed",
						"machine_id": machineID,
					},
				})

				// Save the updated session synchronously to ensure event is immediately available
				s.saveSessionToDB(existing)
			}
		}
	}

	session := &Session{
		ID:           uuid.New().String(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Active:       true,
		MachineID:    machineID,
		Hostname:     hostname,
		State:        make(map[string]interface{}),
		StateHistory: []StateSnapshot{},
		Events:       []Event{},
		Metrics:      []Metric{},
	}

	// Set product and version in the initial state
	if product != "" {
		session.State["product"] = product
	}
	if version != "" {
		session.State["version"] = version
	}

	s.sessions[session.ID] = session

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		// Session is still in memory, continue anyway
	} else {
		// Save to database asynchronously
		sessionID := session.ID
		go func() {
			if err := s.saveSessionDataToDB(sessionID, sessionData); err != nil {
				// Error is already logged in saveSessionDataToDB
			}
		}()
	}

	return session
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}

	// Update active status based on current time before returning
	// Make a copy to avoid modifying the stored session without lock
	sessionCopy := *session
	s.updateActiveStatusReadOnly(&sessionCopy)

	return &sessionCopy, true
}

// updateActiveStatusReadOnly updates active status without modifying the original session
// Used when we have read lock only
func (s *Store) updateActiveStatusReadOnly(session *Session) {
	// If session is crashed, don't change status
	if session.Crashed {
		return
	}

	// If session is ended, it's not active and not suspended
	if session.EndedAt != nil {
		session.Active = false
		session.Suspended = false
		return
	}

	// If no heartbeat has been received yet, keep initial active status
	if session.LastHeartbeat == nil {
		session.Suspended = false
		return
	}

	// Check if heartbeat is within timeout
	timeout := time.Duration(s.config.HeartbeatTimeoutSeconds) * time.Second
	if time.Since(*session.LastHeartbeat) > timeout {
		// No heartbeats - likely suspended/sleeping
		session.Active = false
		session.Suspended = true
	} else {
		// Receiving heartbeats - active and not suspended
		session.Active = true
		session.Suspended = false
	}
}

// GetAllSessions retrieves all sessions
func (s *Store) GetAllSessions() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		// Make a copy and update active status based on current time
		sessionCopy := *session
		s.updateActiveStatusReadOnly(&sessionCopy)
		sessions = append(sessions, &sessionCopy)
	}
	return sessions
}

// deepMerge recursively merges src into dst
func deepMerge(dst, src map[string]interface{}) {
	for k, v := range src {
		// If value is nil, delete the key from destination
		if v == nil {
			delete(dst, k)
			continue
		}

		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := dst[k].(map[string]interface{}); ok {
				// Both are maps, merge recursively
				deepMerge(dstMap, srcMap)
			} else {
				// Destination is not a map, replace with source
				dst[k] = v
			}
		} else {
			// Not a map, just set the value
			dst[k] = v
		}
	}
}

// deepCopyState creates a deep copy of a state map
func deepCopyState(state map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range state {
		if vMap, ok := v.(map[string]interface{}); ok {
			copy[k] = deepCopyState(vMap)
		} else if vSlice, ok := v.([]interface{}); ok {
			copySlice := make([]interface{}, len(vSlice))
			for i, item := range vSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					copySlice[i] = deepCopyState(itemMap)
				} else {
					copySlice[i] = item
				}
			}
			copy[k] = copySlice
		} else {
			copy[k] = v
		}
	}
	return copy
}

// StateUpdateInput represents the input for updating state
type StateUpdateInput struct {
	Timestamp *time.Time             `json:"timestamp,omitempty"` // Optional timestamp from daemon
	State     map[string]interface{} `json:"state"`
}

// UpdateState updates the state of a session (merges new state with existing)
func (s *Store) UpdateState(id string, input StateUpdateInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Use timestamp from daemon if provided, otherwise use server time
	timestamp := time.Now()
	if input.Timestamp != nil {
		timestamp = *input.Timestamp
	}

	// Deep merge the new state into the existing state
	deepMerge(session.State, input.State)
	session.UpdatedAt = time.Now()

	// Create a snapshot of the current state
	snapshot := StateSnapshot{
		Timestamp: timestamp,
		State:     deepCopyState(session.State),
	}
	session.StateHistory = append(session.StateHistory, snapshot)

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Save to database asynchronously
	sessionID := session.ID
	go func() {
		if err := s.saveSessionDataToDB(sessionID, sessionData); err != nil {
			// Error is already logged in saveSessionDataToDB
		}
	}()

	return nil
}

// UpdateHeartbeat updates the last heartbeat time for a session
func (s *Store) UpdateHeartbeat(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	now := time.Now()
	session.LastHeartbeat = &now
	session.UpdatedAt = now

	// Update active status based on heartbeat
	s.updateActiveStatus(session)

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Save to database asynchronously
	sessionID := session.ID
	go func() {
		if err := s.saveSessionDataToDB(sessionID, sessionData); err != nil {
			// Error is already logged in saveSessionDataToDB
		}
	}()

	return nil
}

// updateActiveStatus updates the active status based on heartbeat and ended state
// This should be called with the mutex already locked
func (s *Store) updateActiveStatus(session *Session) {
	// If session is crashed, don't change status
	if session.Crashed {
		return
	}

	// If session is ended, it's not active and not suspended
	if session.EndedAt != nil {
		session.Active = false
		session.Suspended = false
		return
	}

	// If no heartbeat has been received yet, keep initial active status
	if session.LastHeartbeat == nil {
		session.Suspended = false
		return
	}

	// Check if heartbeat is within timeout
	timeout := time.Duration(s.config.HeartbeatTimeoutSeconds) * time.Second
	timeSinceHeartbeat := time.Since(*session.LastHeartbeat)

	if timeSinceHeartbeat > timeout {
		// No heartbeats - likely suspended/sleeping (system sleep, not crashed)
		// This allows the session to resume if heartbeats start again
		session.Active = false
		session.Suspended = true
	} else {
		// Receiving heartbeats - active and not suspended
		session.Active = true
		session.Suspended = false
	}
}

// EndSession marks a session as ended
func (s *Store) EndSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	now := time.Now()
	session.EndedAt = &now
	session.Active = false
	session.UpdatedAt = now

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Save to database asynchronously
	sessionID := session.ID
	go func() {
		if err := s.saveSessionDataToDB(sessionID, sessionData); err != nil {
			// Error is already logged in saveSessionDataToDB
		}
	}()

	return nil
}

// CrashSession marks a session as crashed (abnormal termination)
func (s *Store) CrashSession(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	now := time.Now()
	session.EndedAt = &now
	session.Active = false
	session.Crashed = true
	session.Suspended = false
	session.Hung = false // Clear hung flag when crashed
	session.UpdatedAt = now

	// Add crash event
	session.Events = append(session.Events, Event{
		Timestamp: now,
		Name:      "session_crashed_detected",
		Category:  "critical",
		Data: map[string]interface{}{
			"reason": reason,
		},
	})

	log.Printf("Session %s marked as crashed: %s", id, reason)

	// Save to database synchronously to ensure crash is recorded
	if err := s.saveSessionToDB(session); err != nil {
		log.Printf("Error saving crashed session to database: %v", err)
	}

	return nil
}

// CleanupOldSessions removes sessions older than retention period
func (s *Store) CleanupOldSessions() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	removed := 0

	// Find old sessions
	var toDelete []string
	for id, session := range s.sessions {
		if session.CreatedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	// Delete from database and memory
	for _, id := range toDelete {
		err := s.db.Update(func(txn *badger.Txn) error {
			return txn.Delete([]byte("session:" + id))
		})
		if err != nil {
			log.Printf("Failed to delete session %s from database: %v", id, err)
			continue
		}
		delete(s.sessions, id)
		removed++
	}

	if removed > 0 {
		log.Printf("Cleaned up %d sessions older than %d days", removed, s.config.RetentionDays)
	}

	return removed, nil
}

// StartCleanupRoutine starts background cleanup routine
func (s *Store) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(s.config.CleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			if _, err := s.CleanupOldSessions(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		}
	}()
}

// EventInput represents the input for adding an event
type EventInput struct {
	Timestamp      *time.Time             `json:"timestamp,omitempty"` // Optional timestamp from daemon
	Name           string                 `json:"name"`
	Category       string                 `json:"category"` // User-defined category (e.g., debug, info, warning, error, critical, or custom)
	Group          string                 `json:"group"`    // Group/logger name (e.g., logger name, component name)
	Labels         []string               `json:"labels"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data"`
	Stacktrace     []string               `json:"stacktrace,omitempty"` // Stack trace for any event
	ExceptionType  string                 `json:"exception_type,omitempty"`
	ExceptionMsg   string                 `json:"exception_msg,omitempty"`
	SourceFile     string                 `json:"source_file,omitempty"`
	SourceLine     int                    `json:"source_line,omitempty"`
	SourceFunction string                 `json:"source_function,omitempty"`
}

// AddEvent adds an event to a session
func (s *Store) AddEvent(sessionID string, input EventInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Default category to "info" if not specified
	category := input.Category
	if category == "" {
		category = defaultEventLevel
	}

	// For exceptions, default category to "error" if not specified
	if input.ExceptionType != "" && category == defaultEventLevel {
		category = exceptionEventLevel
	}

	// Use timestamp from daemon if provided, otherwise use server time
	timestamp := time.Now()
	if input.Timestamp != nil {
		timestamp = *input.Timestamp
	}

	event := Event{
		Timestamp:      timestamp,
		Name:           input.Name,
		Category:       category,
		Group:          input.Group,
		Labels:         input.Labels,
		Message:        input.Message,
		Data:           input.Data,
		Stacktrace:     input.Stacktrace,
		ExceptionType:  input.ExceptionType,
		ExceptionMsg:   input.ExceptionMsg,
		SourceFile:     input.SourceFile,
		SourceLine:     input.SourceLine,
		SourceFunction: input.SourceFunction,
	}
	session.Events = append(session.Events, event)
	session.UpdatedAt = time.Now()

	// Mark session as hung if we receive a hang event
	if input.Name == eventHangDetected {
		session.Hung = true
		log.Printf("Session %s marked as hung", sessionID)
	}

	// Clear hung flag if application recovers
	if input.Name == eventHangRecovered {
		session.Hung = false
		log.Printf("Session %s recovered from hang", sessionID)
	}

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Save to database asynchronously
	sid := sessionID // Capture for goroutine
	go func() {
		if err := s.saveSessionDataToDB(sid, sessionData); err != nil {
			// Error is already logged in saveSessionDataToDB
		}
	}()

	return nil
}

// MetricInput represents the input for adding a metric
type MetricInput struct {
	Timestamp *time.Time             `json:"timestamp,omitempty"` // Optional timestamp from daemon
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // "gauge", "counter", "histogram", "timer"
	Value     float64                `json:"value"`
	Count     *int                   `json:"count,omitempty"` // For timers
	Unit      string                 `json:"unit,omitempty"`  // e.g., "seconds", "milliseconds"
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AddMetric adds a metric to a session
func (s *Store) AddMetric(sessionID string, input MetricInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Use timestamp from daemon if provided, otherwise use server time
	timestamp := time.Now()
	if input.Timestamp != nil {
		timestamp = *input.Timestamp
	}

	// Default to gauge if type not specified (backward compatibility)
	metricType := input.Type
	if metricType == "" {
		metricType = "gauge"
	}

	metric := Metric{
		Timestamp: timestamp,
		Name:      input.Name,
		Type:      metricType,
		Value:     input.Value,
		Count:     input.Count,
		Unit:      input.Unit,
		Tags:      input.Tags,
		Metadata:  input.Metadata,
	}
	session.Metrics = append(session.Metrics, metric)
	session.UpdatedAt = time.Now()

	// Marshal session while holding lock to avoid race conditions
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: Failed to marshal session %s: %v", session.ID, err)
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Save to database asynchronously
	sid := sessionID // Capture for goroutine
	go func() {
		if err := s.saveSessionDataToDB(sid, sessionData); err != nil {
			// Error is already logged in saveSessionDataToDB
		}
	}()

	return nil
}

var store *Store
var serverConfig *Config

// gzipMiddleware automatically decompresses gzip-encoded requests
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request", http.StatusBadRequest)
				return
			}
			defer gzipReader.Close()
			r.Body = io.NopCloser(gzipReader)
			r.Header.Del("Content-Encoding")
		}
		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware validates API key if required
func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Only check API key if required
		if serverConfig.RequireAPIKey {
			authHeader := r.Header.Get("Authorization")
			expectedAuth := "Bearer " + serverConfig.APIKey

			if authHeader != expectedAuth {
				log.Printf("Unauthorized request from %s to %s", r.RemoteAddr, r.URL.Path)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// initLogging initializes file logging if configured
// Returns log file path, cleanup function, and error
func initLogging(config *Config) (string, func(), error) {
	if config.LogFile == "" {
		// No log file configured, use stdout only
		return "", func() {}, nil
	}

	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to open log file: %w", err)
	}

	// Set log output to both file and stdout
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	cleanup := func() {
		logFile.Close()
	}

	return config.LogFile, cleanup, nil
}

func main() {
	// Load configuration
	config := LoadConfig("./config.json")
	serverConfig = config // Store for middleware

	// Initialize file logging
	logPath, logCleanup, err := initLogging(config)
	if err != nil {
		// If logging init fails, continue with stdout only
		log.Printf("WARNING: Failed to initialize file logging: %v", err)
	} else {
		defer logCleanup()
		log.Printf("Logging to file: %s", logPath)
	}

	log.Printf("Configuration loaded: Data path=%s, Retention=%d days, Port=%s",
		config.DataPath, config.RetentionDays, config.ServerPort)

	// Initialize store with BadgerDB
	store, err = NewStore(config.DataPath, config)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Start cleanup routine
	store.StartCleanupRoutine()
	log.Printf("Started cleanup routine (interval: %v)", config.CleanupInterval)

	// Create a new mux to apply middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "datacat-server",
			"version": "1.0.0",
		})
	})
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSessionOperations)
	mux.HandleFunc("/api/data/sessions", handleGetAllSessions)

	// Apply middleware
	handler := apiKeyMiddleware(gzipMiddleware(mux))

	port := ":" + config.ServerPort

	// Check if TLS is configured
	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		log.Printf("Starting datacat server on %s (HTTPS)", port)
		if config.RequireAPIKey {
			log.Printf("API key authentication enabled")
		}
		log.Fatal(http.ListenAndServeTLS(port, config.TLSCertFile, config.TLSKeyFile, handler))
	} else {
		log.Printf("Starting datacat server on %s (HTTP)", port)
		if config.RequireAPIKey {
			log.Printf("API key authentication enabled")
		}
		log.Fatal(http.ListenAndServe(port, handler))
	}
}

// handleSessions handles POST /api/sessions to create a new session
func handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse request body to get product, version, and machine info
		var req struct {
			Product   string `json:"product"`
			Version   string `json:"version"`
			MachineID string `json:"machine_id,omitempty"`
			Hostname  string `json:"hostname,omitempty"`
		}

		// Try to decode the request body
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}

		// Validate that product and version are provided
		if req.Product == "" || req.Version == "" {
			http.Error(w, "Product and version are required fields", http.StatusBadRequest)
			return
		}

		session := store.CreateSession(req.Product, req.Version, req.MachineID, req.Hostname)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"session_id": session.ID})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleSessionOperations handles operations on specific sessions
func handleSessionOperations(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path
	path := r.URL.Path
	pathParts := strings.TrimPrefix(path, "/api/sessions/")

	// Split to get session ID and operation
	parts := strings.SplitN(pathParts, "/", 2)
	sessionID := parts[0]
	operation := ""
	if len(parts) > 1 {
		operation = parts[1]
	}

	if len(sessionID) == 0 {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// GET /api/sessions/{id} - Get session details
	if r.Method == "GET" && operation == "" {
		session, ok := store.GetSession(sessionID)
		if !ok {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
		return
	}

	// POST /api/sessions/{id}/state - Update state
	if r.Method == "POST" && operation == "state" {
		// Read the body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var input StateUpdateInput

		// Try to unmarshal as StateUpdateInput (new format with timestamp)
		if err := json.Unmarshal(bodyBytes, &input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Handle backward compatibility: if State is nil, treat the entire body as state
		if input.State == nil {
			var plainState map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &plainState); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			input.State = plainState
		}

		if err := store.UpdateState(sessionID, input); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/events - Add event
	if r.Method == "POST" && operation == "events" {
		// Read body to handle both EventData (from daemon) and EventInput formats
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: Failed to read event data for session %s: %v", sessionID, err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var eventData EventInput
		// Try to unmarshal as EventInput (with optional timestamp pointer)
		if err := json.Unmarshal(bodyBytes, &eventData); err != nil {
			log.Printf("ERROR: Failed to decode event data for session %s: %v", sessionID, err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// If timestamp is nil but JSON has a timestamp field, try to parse it
		if eventData.Timestamp == nil {
			var temp struct {
				Timestamp string `json:"timestamp"`
			}
			if err := json.Unmarshal(bodyBytes, &temp); err == nil && temp.Timestamp != "" {
				if t, err := time.Parse(time.RFC3339Nano, temp.Timestamp); err == nil {
					eventData.Timestamp = &t
				} else if t, err := time.Parse(time.RFC3339, temp.Timestamp); err == nil {
					eventData.Timestamp = &t
				}
			}
		}

		if err := store.AddEvent(sessionID, eventData); err != nil {
			log.Printf("ERROR: Failed to add event for session %s: %v", sessionID, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/metrics - Add metric
	if r.Method == "POST" && operation == "metrics" {
		var metricData MetricInput
		if err := json.NewDecoder(r.Body).Decode(&metricData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.AddMetric(sessionID, metricData); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/end - End session
	if r.Method == "POST" && operation == "end" {
		if err := store.EndSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/crash - Mark session as crashed
	if r.Method == "POST" && operation == "crash" {
		var crashData struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&crashData); err != nil {
			// Default reason if body is empty or invalid
			crashData.Reason = "abnormal_termination"
		}

		if err := store.CrashSession(sessionID, crashData.Reason); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/heartbeat - Update heartbeat
	if r.Method == "POST" && operation == "heartbeat" {
		if err := store.UpdateHeartbeat(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleGetAllSessions handles GET /api/data/sessions to export all sessions in JSON format
func handleGetAllSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := store.GetAllSessions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
