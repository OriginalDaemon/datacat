package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
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
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"`
	Level     string                 `json:"level"`    // debug, info, warning, error, critical
	Category  string                 `json:"category"` // e.g., logger name, component name
	Labels    []string               `json:"labels"`   // arbitrary tags for filtering
	Message   string                 `json:"message"`  // human-readable message
	Data      map[string]interface{} `json:"data"`     // additional structured data

	// Exception-specific fields (when this is an exception event)
	ExceptionType  string   `json:"exception_type,omitempty"`  // e.g., "ValueError", "NullPointerException"
	ExceptionMsg   string   `json:"exception_msg,omitempty"`   // exception message
	Stacktrace     []string `json:"stacktrace,omitempty"`      // array of stack trace lines
	SourceFile     string   `json:"source_file,omitempty"`     // file where exception occurred
	SourceLine     int      `json:"source_line,omitempty"`     // line number where exception occurred
	SourceFunction string   `json:"source_function,omitempty"` // function where exception occurred
}

// Metric represents a metric logged in a session
type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Tags      []string  `json:"tags,omitempty"`
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
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("session:"+session.ID), data)
	})

	if err != nil {
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

	// Save to database asynchronously
	go s.saveSessionToDB(session)

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

// UpdateState updates the state of a session (merges new state with existing)
func (s *Store) UpdateState(id string, state map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Deep merge the new state into the existing state
	deepMerge(session.State, state)
	session.UpdatedAt = time.Now()

	// Create a snapshot of the current state
	snapshot := StateSnapshot{
		Timestamp: time.Now(),
		State:     deepCopyState(session.State),
	}
	session.StateHistory = append(session.StateHistory, snapshot)

	// Save to database asynchronously
	go s.saveSessionToDB(session)

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

	// Save to database asynchronously
	go s.saveSessionToDB(session)

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

	// Save to database asynchronously
	go s.saveSessionToDB(session)

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

// AddEvent adds an event to a session
func (s *Store) AddEvent(id string, name string, level string, category string, labels []string, message string, data map[string]interface{}, exceptionType string, exceptionMsg string, stacktrace []string, sourceFile string, sourceLine int, sourceFunction string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Default level to "info" if not specified
	if level == "" {
		level = "info"
	}

	// For exceptions, default level to "error" if not specified
	if exceptionType != "" && level == "info" {
		level = "error"
	}

	event := Event{
		Timestamp:      time.Now(),
		Name:           name,
		Level:          level,
		Category:       category,
		Labels:         labels,
		Message:        message,
		Data:           data,
		ExceptionType:  exceptionType,
		ExceptionMsg:   exceptionMsg,
		Stacktrace:     stacktrace,
		SourceFile:     sourceFile,
		SourceLine:     sourceLine,
		SourceFunction: sourceFunction,
	}
	session.Events = append(session.Events, event)
	session.UpdatedAt = time.Now()

	// Mark session as hung if we receive a hang event
	if name == "application_appears_hung" {
		session.Hung = true
		log.Printf("Session %s marked as hung", id)
	}

	// Clear hung flag if application recovers
	if name == "application_recovered" {
		session.Hung = false
		log.Printf("Session %s recovered from hang", id)
	}

	// Save to database asynchronously
	go s.saveSessionToDB(session)

	return nil
}

// AddMetric adds a metric to a session
func (s *Store) AddMetric(id string, name string, value float64, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	metric := Metric{
		Timestamp: time.Now(),
		Name:      name,
		Value:     value,
		Tags:      tags,
	}
	session.Metrics = append(session.Metrics, metric)
	session.UpdatedAt = time.Now()

	// Save to database asynchronously
	go s.saveSessionToDB(session)

	return nil
}

var store *Store

func main() {
	// Load configuration
	config := LoadConfig("./config.json")
	log.Printf("Configuration loaded: Data path=%s, Retention=%d days, Port=%s",
		config.DataPath, config.RetentionDays, config.ServerPort)

	// Initialize store with BadgerDB
	var err error
	store, err = NewStore(config.DataPath, config)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Start cleanup routine
	store.StartCleanupRoutine()
	log.Printf("Started cleanup routine (interval: %v)", config.CleanupInterval)

	http.HandleFunc("/api/sessions", handleSessions)
	http.HandleFunc("/api/sessions/", handleSessionOperations)
	http.HandleFunc("/api/data/sessions", handleGetAllSessions)

	port := ":" + config.ServerPort
	log.Printf("Starting datacat server on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
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
		var state map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.UpdateState(sessionID, state); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/events - Add event
	if r.Method == "POST" && operation == "events" {
		var eventData struct {
			Name           string                 `json:"name"`
			Level          string                 `json:"level"`           // optional: debug, info, warning, error, critical
			Category       string                 `json:"category"`        // optional: logger name, component
			Labels         []string               `json:"labels"`          // optional: tags for filtering
			Message        string                 `json:"message"`         // optional: human-readable message
			Data           map[string]interface{} `json:"data"`            // optional: additional structured data
			ExceptionType  string                 `json:"exception_type"`  // optional: for exceptions
			ExceptionMsg   string                 `json:"exception_msg"`   // optional: for exceptions
			Stacktrace     []string               `json:"stacktrace"`      // optional: for exceptions
			SourceFile     string                 `json:"source_file"`     // optional: for exceptions
			SourceLine     int                    `json:"source_line"`     // optional: for exceptions
			SourceFunction string                 `json:"source_function"` // optional: for exceptions
		}
		if err := json.NewDecoder(r.Body).Decode(&eventData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.AddEvent(sessionID, eventData.Name, eventData.Level, eventData.Category, eventData.Labels, eventData.Message, eventData.Data, eventData.ExceptionType, eventData.ExceptionMsg, eventData.Stacktrace, eventData.SourceFile, eventData.SourceLine, eventData.SourceFunction); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// POST /api/sessions/{id}/metrics - Add metric
	if r.Method == "POST" && operation == "metrics" {
		var metricData struct {
			Name  string   `json:"name"`
			Value float64  `json:"value"`
			Tags  []string `json:"tags,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&metricData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.AddMetric(sessionID, metricData.Name, metricData.Value, metricData.Tags); err != nil {
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
