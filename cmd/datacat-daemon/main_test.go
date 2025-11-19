package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewDaemon(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	if daemon == nil {
		t.Fatal("NewDaemon returned nil")
	}
	if daemon.config != config {
		t.Error("Config not set correctly")
	}
	if daemon.sessions == nil {
		t.Error("Sessions map is nil")
	}
}

func TestHandleRegister(t *testing.T) {
	// Create a mock server to simulate datacat-server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"session_id": "test-session-id"}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()
	
	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)
	
	reqBody := map[string]int{"parent_pid": 1234}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleRegister(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	
	if response["session_id"] == "" {
		t.Error("session_id should not be empty")
	}
}

func TestHandleState(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	// Create a session buffer
	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:    sessionID,
		StateUpdates: []map[string]interface{}{},
	}
	daemon.mu.Unlock()
	
	stateData := map[string]interface{}{
		"session_id": sessionID,
		"state":      map[string]interface{}{"key": "value"},
	}
	body, _ := json.Marshal(stateData)
	
	req := httptest.NewRequest("POST", "/state", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleState(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Verify state was buffered
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()
	
	if len(buffer.StateUpdates) != 1 {
		t.Errorf("Expected 1 state update, got %d", len(buffer.StateUpdates))
	}
}

func TestHandleEvent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID: sessionID,
		Events:    []EventData{},
	}
	daemon.mu.Unlock()
	
	eventData := map[string]interface{}{
		"session_id": sessionID,
		"name":       "test_event",
		"data":       map[string]interface{}{"msg": "hello"},
	}
	body, _ := json.Marshal(eventData)
	
	req := httptest.NewRequest("POST", "/event", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleEvent(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Verify event was buffered
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()
	
	if len(buffer.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(buffer.Events))
	}
	if buffer.Events[0].Name != "test_event" {
		t.Errorf("Expected event name test_event, got %s", buffer.Events[0].Name)
	}
}

func TestHandleMetric(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID: sessionID,
		Metrics:   []MetricData{},
	}
	daemon.mu.Unlock()
	
	metricData := map[string]interface{}{
		"session_id": sessionID,
		"name":       "cpu_usage",
		"value":      75.5,
		"tags":       []interface{}{"tag1"},
	}
	body, _ := json.Marshal(metricData)
	
	req := httptest.NewRequest("POST", "/metric", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleMetric(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Verify metric was buffered
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()
	
	if len(buffer.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(buffer.Metrics))
	}
	if buffer.Metrics[0].Name != "cpu_usage" {
		t.Errorf("Expected metric name cpu_usage, got %s", buffer.Metrics[0].Name)
	}
}

func TestHandleHeartbeat(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:     sessionID,
		LastHeartbeat: time.Now().Add(-10 * time.Second),
	}
	daemon.mu.Unlock()
	
	heartbeatData := map[string]interface{}{
		"session_id": sessionID,
	}
	body, _ := json.Marshal(heartbeatData)
	
	req := httptest.NewRequest("POST", "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleHeartbeat(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Verify heartbeat was updated
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()
	
	if time.Since(buffer.LastHeartbeat) > 2*time.Second {
		t.Error("Heartbeat timestamp should have been updated")
	}
}

func TestHandleEnd(t *testing.T) {
	// Create a mock server to simulate datacat-server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()
	
	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)
	
	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID: sessionID,
	}
	daemon.mu.Unlock()
	
	endData := map[string]interface{}{
		"session_id": sessionID,
	}
	body, _ := json.Marshal(endData)
	
	req := httptest.NewRequest("POST", "/end", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	daemon.handleEnd(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandleHealth(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	daemon.handleHealth(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	
	if response["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", response["status"])
	}
}

func TestSessionBuffer(t *testing.T) {
	buffer := &SessionBuffer{
		SessionID:     "test-id",
		StateUpdates:  []map[string]interface{}{},
		Events:        []EventData{},
		Metrics:       []MetricData{},
		LastHeartbeat: time.Now(),
		ParentPID:     1234,
	}
	
	if buffer.SessionID != "test-id" {
		t.Errorf("Expected SessionID test-id, got %s", buffer.SessionID)
	}
	if buffer.ParentPID != 1234 {
		t.Errorf("Expected ParentPID 1234, got %d", buffer.ParentPID)
	}
	if buffer.HangLogged {
		t.Error("HangLogged should be false initially")
	}
	if buffer.CrashLogged {
		t.Error("CrashLogged should be false initially")
	}
}
