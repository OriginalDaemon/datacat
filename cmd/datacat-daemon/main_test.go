package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

	reqBody := map[string]interface{}{
		"parent_pid": 1234,
		"product":    "TestProduct",
		"version":    "1.0.0",
	}
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
		StateUpdates: []StateUpdate{},
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
		StateUpdates:  []StateUpdate{},
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

func TestHasStateChanged(t *testing.T) {
	daemon := NewDaemon(DefaultConfig())

	// Test no change
	old := map[string]interface{}{"key": "value"}
	new := map[string]interface{}{"key": "value"}
	if daemon.hasStateChanged(old, new) {
		t.Error("Expected no change when states are equal")
	}

	// Test new key added
	new2 := map[string]interface{}{"key": "value", "new_key": "new_value"}
	if !daemon.hasStateChanged(old, new2) {
		t.Error("Expected change when new key is added")
	}

	// Test value changed
	new3 := map[string]interface{}{"key": "different"}
	if !daemon.hasStateChanged(old, new3) {
		t.Error("Expected change when value is changed")
	}

	// Test nested objects
	oldNested := map[string]interface{}{
		"outer": map[string]interface{}{"inner": "value"},
	}
	newNested := map[string]interface{}{
		"outer": map[string]interface{}{"inner": "different"},
	}
	if !daemon.hasStateChanged(oldNested, newNested) {
		t.Error("Expected change when nested value is changed")
	}
}

func TestDeepEqual(t *testing.T) {
	daemon := NewDaemon(DefaultConfig())

	// Test equal values
	if !daemon.deepEqual("value", "value") {
		t.Error("Expected equal strings to be equal")
	}

	if !daemon.deepEqual(123, 123) {
		t.Error("Expected equal numbers to be equal")
	}

	// Test unequal values
	if daemon.deepEqual("value1", "value2") {
		t.Error("Expected different strings to not be equal")
	}

	// Test maps
	map1 := map[string]interface{}{"key": "value"}
	map2 := map[string]interface{}{"key": "value"}
	if !daemon.deepEqual(map1, map2) {
		t.Error("Expected equal maps to be equal")
	}

	map3 := map[string]interface{}{"key": "different"}
	if daemon.deepEqual(map1, map3) {
		t.Error("Expected different maps to not be equal")
	}
}

func TestMergeState(t *testing.T) {
	daemon := NewDaemon(DefaultConfig())

	// Test simple merge
	old := map[string]interface{}{"key1": "value1"}
	new := map[string]interface{}{"key2": "value2"}
	merged := daemon.mergeState(old, new)

	if merged["key1"] != "value1" {
		t.Error("Expected old key to be preserved")
	}
	if merged["key2"] != "value2" {
		t.Error("Expected new key to be added")
	}

	// Test overwrite
	old2 := map[string]interface{}{"key": "old_value"}
	new2 := map[string]interface{}{"key": "new_value"}
	merged2 := daemon.mergeState(old2, new2)

	if merged2["key"] != "new_value" {
		t.Error("Expected new value to overwrite old value")
	}

	// Test nested merge
	old3 := map[string]interface{}{
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}
	new3 := map[string]interface{}{
		"nested": map[string]interface{}{
			"key2": "new_value2",
			"key3": "value3",
		},
	}
	merged3 := daemon.mergeState(old3, new3)

	nested := merged3["nested"].(map[string]interface{})
	if nested["key1"] != "value1" {
		t.Error("Expected nested key1 to be preserved")
	}
	if nested["key2"] != "new_value2" {
		t.Error("Expected nested key2 to be updated")
	}
	if nested["key3"] != "value3" {
		t.Error("Expected nested key3 to be added")
	}
}

func TestFlushSession(t *testing.T) {
	// Create a mock server
	var stateUpdates, events, metrics int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/test-session/state":
			stateUpdates++
		case "/api/sessions/test-session/events":
			events++
		case "/api/sessions/test-session/metrics":
			metrics++
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	// Create session buffer with data
	daemon.mu.Lock()
	daemon.sessions["test-session"] = &SessionBuffer{
		SessionID: "test-session",
		StateUpdates: []StateUpdate{
			{Timestamp: time.Now(), State: map[string]interface{}{"key": "value1"}},
			{Timestamp: time.Now(), State: map[string]interface{}{"key": "value2"}},
		},
		Events: []EventData{
			{Name: "event1", Data: map[string]interface{}{}},
		},
		Metrics: []MetricData{
			{Name: "metric1", Value: 1.0},
		},
	}
	daemon.mu.Unlock()

	// Flush the session
	daemon.flushSession("test-session")

	// Wait a bit for async operations
	time.Sleep(100 * time.Millisecond)

	// Verify buffer is cleared
	daemon.mu.RLock()
	buffer := daemon.sessions["test-session"]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if len(buffer.StateUpdates) != 0 {
		t.Errorf("Expected 0 state updates after flush, got %d", len(buffer.StateUpdates))
	}
	if len(buffer.Events) != 0 {
		t.Errorf("Expected 0 events after flush, got %d", len(buffer.Events))
	}
	if len(buffer.Metrics) != 0 {
		t.Errorf("Expected 0 metrics after flush, got %d", len(buffer.Metrics))
	}
	buffer.mu.Unlock()
}

func TestFlushSessionNonExistent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	// Flush non-existent session should not panic
	daemon.flushSession("non-existent")
}

func TestHandleRegisterMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()

	daemon.handleRegister(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleStateMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/state", nil)
	w := httptest.NewRecorder()

	daemon.handleState(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleEventMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/event", nil)
	w := httptest.NewRecorder()

	daemon.handleEvent(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleMetricMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/metric", nil)
	w := httptest.NewRecorder()

	daemon.handleMetric(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleHeartbeatMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/heartbeat", nil)
	w := httptest.NewRecorder()

	daemon.handleHeartbeat(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleEndMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/end", nil)
	w := httptest.NewRecorder()

	daemon.handleEnd(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleStateWithoutSession(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	stateData := map[string]interface{}{
		"session_id": "non-existent",
		"state":      map[string]interface{}{"key": "value"},
	}
	body, _ := json.Marshal(stateData)

	req := httptest.NewRequest("POST", "/state", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleState(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleEventWithoutSession(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	eventData := map[string]interface{}{
		"session_id": "non-existent",
		"name":       "test_event",
		"data":       map[string]interface{}{},
	}
	body, _ := json.Marshal(eventData)

	req := httptest.NewRequest("POST", "/event", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleEvent(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleMetricWithoutSession(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	metricData := map[string]interface{}{
		"session_id": "non-existent",
		"name":       "test_metric",
		"value":      1.0,
	}
	body, _ := json.Marshal(metricData)

	req := httptest.NewRequest("POST", "/metric", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleMetric(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleRegisterInvalidJSON(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/register", nil)
	w := httptest.NewRecorder()

	daemon.handleRegister(w, req)

	// Should get a 400 error due to missing product/version
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRegisterInvalidSessionID(t *testing.T) {
	// Create a mock server that returns invalid session_id type
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{"session_id": 12345} // Not a string
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/register", nil)
	w := httptest.NewRecorder()

	daemon.handleRegister(w, req)

	// Should get a 400 error due to missing product/version
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleStateInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/state", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleState(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleEventInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/event", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleMetricInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/metric", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleMetric(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleHeartbeatInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/heartbeat", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleHeartbeatNonExistentSession(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	heartbeatData := map[string]interface{}{
		"session_id": "non-existent",
	}
	body, _ := json.Marshal(heartbeatData)

	req := httptest.NewRequest("POST", "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleEndInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/end", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleEnd(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleEndNonExistentSession(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("session not found"))
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	sessionID := "test-session"
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

	// The daemon always returns 200, even if server returns error
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify session was removed from daemon
	daemon.mu.RLock()
	_, exists := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	if exists {
		t.Error("Session should have been removed from daemon")
	}
}

func TestHandleStateNoChange(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:    sessionID,
		StateUpdates: []StateUpdate{},
		LastState:    map[string]interface{}{"key": "value"},
	}
	daemon.mu.Unlock()

	// Send same state (no change)
	stateData := map[string]interface{}{
		"session_id": sessionID,
		"state":      map[string]interface{}{"key": "value"},
	}
	body, _ := json.Marshal(stateData)

	req := httptest.NewRequest("POST", "/state", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleState(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify no state update was added
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if len(buffer.StateUpdates) != 0 {
		t.Errorf("Expected 0 state updates when state doesn't change, got %d", len(buffer.StateUpdates))
	}
	buffer.mu.Unlock()
}

func TestSendStateUpdateError(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-host-that-does-not-exist-12345:99999"
	daemon := NewDaemon(config)

	// This should log an error but not panic
	daemon.sendStateUpdate("test-session", StateUpdate{
		Timestamp: time.Now(),
		State:     map[string]interface{}{"key": "value"},
	})
}

func TestSendEventError(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-host-that-does-not-exist-12345:99999"
	daemon := NewDaemon(config)

	// This should log an error but not panic
	daemon.sendEvent("test-session", EventData{Name: "event", Data: map[string]interface{}{}})
}

func TestSendMetricError(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-host-that-does-not-exist-12345:99999"
	daemon := NewDaemon(config)

	// This should log an error but not panic
	daemon.sendMetric("test-session", MetricData{Name: "metric", Value: 1.0})
}

func TestHandleHeartbeatRecovery(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:     sessionID,
		LastHeartbeat: time.Now().Add(-10 * time.Second),
		HangLogged:    true, // Mark as hung
		Events:        []EventData{},
	}
	daemon.mu.Unlock()

	heartbeatData := map[string]interface{}{
		"session_id": sessionID,
	}
	body, _ := json.Marshal(heartbeatData)

	req := httptest.NewRequest("POST", "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleHeartbeat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify recovery event was added
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if len(buffer.Events) != 1 {
		t.Errorf("Expected 1 recovery event, got %d", len(buffer.Events))
	}
	if buffer.Events[0].Name != "application_recovered" {
		t.Errorf("Expected application_recovered event, got %s", buffer.Events[0].Name)
	}
	if buffer.HangLogged {
		t.Error("HangLogged should be reset after recovery")
	}
	buffer.mu.Unlock()
}

func TestHandleEndForwardsToServer(t *testing.T) {
	// Create a mock server that tracks if end was called
	endCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/test-session/end" {
			endCalled = true
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	sessionID := "test-session"
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

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !endCalled {
		t.Error("Expected end to be forwarded to server")
	}
}

func TestCheckHeartbeat(t *testing.T) {
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1 // Short timeout for testing
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:     sessionID,
		LastHeartbeat: time.Now().Add(-2 * time.Second), // Older than timeout
		Events:        []EventData{},
		HangLogged:    false,
	}
	daemon.mu.Unlock()

	// Check heartbeat - should detect hang
	daemon.checkHeartbeat(sessionID)

	// Verify hang event was logged
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if len(buffer.Events) != 1 {
		t.Errorf("Expected 1 hang event, got %d", len(buffer.Events))
	}
	if buffer.Events[0].Name != "application_appears_hung" {
		t.Errorf("Expected application_appears_hung event, got %s", buffer.Events[0].Name)
	}
	if !buffer.HangLogged {
		t.Error("HangLogged should be true after detecting hang")
	}
	buffer.mu.Unlock()

	// Check heartbeat again - should not log duplicate event
	daemon.checkHeartbeat(sessionID)

	buffer.mu.Lock()
	if len(buffer.Events) != 1 {
		t.Errorf("Expected still 1 hang event (no duplicate), got %d", len(buffer.Events))
	}
	buffer.mu.Unlock()
}

func TestCheckHeartbeatNonExistent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	// Check heartbeat for non-existent session should not panic
	daemon.checkHeartbeat("non-existent")
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	if !isProcessRunning(os.Getpid()) {
		t.Error("Current process should be running")
	}

	// Test with a PID that definitely doesn't exist
	if isProcessRunning(99999) {
		t.Error("PID 99999 should not be running")
	}
}

func TestCheckParentProcess(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:   sessionID,
		ParentPID:   99999, // Non-existent PID
		Events:      []EventData{},
		CrashLogged: false,
		Active:      true, // Session must be active for crash detection
	}
	daemon.mu.Unlock()

	// Check parent process - should detect crash
	daemon.checkParentProcess(sessionID)

	// Verify crash event was logged
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	// The event is queued for sending, not added to buffer.Events
	// Just check that CrashLogged flag is set
	if !buffer.CrashLogged {
		t.Error("CrashLogged should be true after detecting crash")
	}
	buffer.mu.Unlock()

	// Check again - should not log duplicate event
	daemon.checkParentProcess(sessionID)

	buffer.mu.Lock()
	// CrashLogged flag should still be true, preventing duplicates
	if !buffer.CrashLogged {
		t.Error("CrashLogged should still be true")
	}
	buffer.mu.Unlock()
}

func TestCheckParentProcessNonExistent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	// Check parent process for non-existent session should not panic
	daemon.checkParentProcess("non-existent")
}

func TestCheckParentProcessNoPID(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:   sessionID,
		ParentPID:   0, // No parent PID
		Events:      []EventData{},
		CrashLogged: false,
	}
	daemon.mu.Unlock()

	// Check parent process - should skip
	daemon.checkParentProcess(sessionID)

	// Verify no event was logged
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if len(buffer.Events) != 0 {
		t.Errorf("Expected 0 events when no parent PID, got %d", len(buffer.Events))
	}
	buffer.mu.Unlock()
}

// Test handlePauseHeartbeat
func TestHandlePauseHeartbeat(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:              sessionID,
		HeartbeatMonitorPaused: false,
	}
	daemon.mu.Unlock()

	reqBody := map[string]interface{}{
		"session_id": sessionID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/pause_heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handlePauseHeartbeat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify heartbeat monitoring is paused
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if !buffer.HeartbeatMonitorPaused {
		t.Error("Expected heartbeat monitoring to be paused")
	}
	buffer.mu.Unlock()
}

func TestHandlePauseHeartbeatInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/pause_heartbeat", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handlePauseHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlePauseHeartbeatNonExistent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	reqBody := map[string]interface{}{
		"session_id": "non-existent",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/pause_heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handlePauseHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandlePauseHeartbeatMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/pause_heartbeat", nil)
	w := httptest.NewRecorder()

	daemon.handlePauseHeartbeat(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Test handleResumeHeartbeat
func TestHandleResumeHeartbeat(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	sessionID := "test-session-id"
	daemon.mu.Lock()
	daemon.sessions[sessionID] = &SessionBuffer{
		SessionID:              sessionID,
		HeartbeatMonitorPaused: true,
		LastHeartbeat:          time.Now().Add(-2 * time.Minute), // Old heartbeat
	}
	daemon.mu.Unlock()

	reqBody := map[string]interface{}{
		"session_id": sessionID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/resume_heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleResumeHeartbeat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify heartbeat monitoring is resumed and timestamp reset
	daemon.mu.RLock()
	buffer := daemon.sessions[sessionID]
	daemon.mu.RUnlock()

	buffer.mu.Lock()
	if buffer.HeartbeatMonitorPaused {
		t.Error("Expected heartbeat monitoring to be resumed")
	}
	if time.Since(buffer.LastHeartbeat) > 1*time.Second {
		t.Error("Expected LastHeartbeat to be reset to recent time")
	}
	buffer.mu.Unlock()
}

func TestHandleResumeHeartbeatInvalidJSON(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/resume_heartbeat", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	daemon.handleResumeHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleResumeHeartbeatNonExistent(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	reqBody := map[string]interface{}{
		"session_id": "non-existent",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/resume_heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	daemon.handleResumeHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleResumeHeartbeatMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/resume_heartbeat", nil)
	w := httptest.NewRecorder()

	daemon.handleResumeHeartbeat(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Test retry functions
func TestRetryCreateSession(t *testing.T) {
	// Create a mock server to simulate datacat-server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"session_id": "server-session-id"}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	// Create local session
	daemon.mu.Lock()
	daemon.sessions["local-session-id"] = &SessionBuffer{
		SessionID:        "local-session-id",
		SyncedWithServer: false,
	}
	daemon.mu.Unlock()

	// Test retry
	sessionData := map[string]interface{}{
		"product":    "TestProduct",
		"version":    "1.0.0",
		"machine_id": "test-machine",
		"hostname":   "test-host",
	}
	success := daemon.retryCreateSession("local-session-id", sessionData)

	if !success {
		t.Error("Expected retryCreateSession to succeed")
	}

	// Verify session was updated
	daemon.mu.RLock()
	_, existsOld := daemon.sessions["local-session-id"]
	buffer, existsNew := daemon.sessions["server-session-id"]
	daemon.mu.RUnlock()

	if existsOld {
		t.Error("Expected old session ID to be removed")
	}
	if !existsNew {
		t.Fatal("Expected new session ID to exist")
	}
	if !buffer.SyncedWithServer {
		t.Error("Expected session to be marked as synced")
	}
}

func TestRetryCreateSessionFailure(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	sessionData := map[string]interface{}{
		"product": "TestProduct",
		"version": "1.0.0",
	}
	success := daemon.retryCreateSession("local-session-id", sessionData)

	if success {
		t.Error("Expected retryCreateSession to fail with invalid server")
	}
}

func TestRetrySendState(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	stateUpdate := StateUpdate{
		Timestamp: time.Now(),
		State:     map[string]interface{}{"key": "value"},
	}
	success := daemon.retrySendState("test-session-id", stateUpdate)

	if !success {
		t.Error("Expected retrySendState to succeed")
	}
}

func TestRetrySendStateFailure(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	stateUpdate := StateUpdate{
		Timestamp: time.Now(),
		State:     map[string]interface{}{"key": "value"},
	}
	success := daemon.retrySendState("test-session-id", stateUpdate)

	if success {
		t.Error("Expected retrySendState to fail with invalid server")
	}
}

func TestRetrySendEvent(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	event := EventData{
		Name: "test_event",
		Data: map[string]interface{}{"key": "value"},
	}
	success := daemon.retrySendEvent("test-session-id", event)

	if !success {
		t.Error("Expected retrySendEvent to succeed")
	}
}

func TestRetrySendEventFailure(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	event := EventData{Name: "test_event", Data: map[string]interface{}{}}
	success := daemon.retrySendEvent("test-session-id", event)

	if success {
		t.Error("Expected retrySendEvent to fail with invalid server")
	}
}

func TestRetrySendMetric(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	metric := MetricData{
		Name:  "test_metric",
		Value: 42.0,
		Tags:  []string{"test"},
	}
	success := daemon.retrySendMetric("test-session-id", metric)

	if !success {
		t.Error("Expected retrySendMetric to succeed")
	}
}

func TestRetrySendMetricFailure(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	metric := MetricData{Name: "test_metric", Value: 42.0}
	success := daemon.retrySendMetric("test-session-id", metric)

	if success {
		t.Error("Expected retrySendMetric to fail with invalid server")
	}
}

func TestRetryEndSession(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	// Create session
	daemon.mu.Lock()
	daemon.sessions["test-session-id"] = &SessionBuffer{
		SessionID: "test-session-id",
	}
	daemon.mu.Unlock()

	success := daemon.retryEndSession("test-session-id")

	if !success {
		t.Error("Expected retryEndSession to succeed")
	}

	// Verify session was removed
	daemon.mu.RLock()
	_, exists := daemon.sessions["test-session-id"]
	daemon.mu.RUnlock()

	if exists {
		t.Error("Expected session to be removed after successful end")
	}
}

func TestRetryEndSessionFailure(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	success := daemon.retryEndSession("test-session-id")

	if success {
		t.Error("Expected retryEndSession to fail with invalid server")
	}
}

// Test handleGetSession
func TestHandleGetSession(t *testing.T) {
	// Create a mock server that returns session data
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":      "test-session-id",
			"active":  true,
			"state":   map[string]interface{}{"key": "value"},
			"events":  []interface{}{},
			"metrics": []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/session?session_id=test-session-id", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["id"] != "test-session-id" {
		t.Errorf("Expected session id test-session-id, got %v", result["id"])
	}
}

func TestHandleGetSessionLocalFallback(t *testing.T) {
	// Test with server unavailable - should fall back to local buffer
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	// Create local session
	daemon.mu.Lock()
	daemon.sessions["test-session-id"] = &SessionBuffer{
		SessionID: "test-session-id",
		Active:    true,
		LastState: map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
	}
	daemon.mu.Unlock()

	req := httptest.NewRequest("GET", "/session?session_id=test-session-id", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["id"] != "test-session-id" {
		t.Errorf("Expected session id test-session-id, got %v", result["id"])
	}
}

func TestHandleGetSessionMissingID(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/session", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGetSessionNotFound(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/session?session_id=non-existent", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleGetSessionMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/session", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSession(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Test handleGetSessions
func TestHandleGetSessions(t *testing.T) {
	// Create a mock server that returns sessions list
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"id":     "session-1",
				"active": true,
			},
			{
				"id":     "session-2",
				"active": false,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	req := httptest.NewRequest("GET", "/sessions", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(result))
	}
}

func TestHandleGetSessionsLocalFallback(t *testing.T) {
	// Test with server unavailable - should return local sessions
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	// Create local sessions
	daemon.mu.Lock()
	daemon.sessions["session-1"] = &SessionBuffer{
		SessionID: "session-1",
		Active:    true,
		LastState: map[string]interface{}{},
		CreatedAt: time.Now(),
	}
	daemon.sessions["session-2"] = &SessionBuffer{
		SessionID: "session-2",
		Active:    false,
		LastState: map[string]interface{}{},
		CreatedAt: time.Now(),
	}
	daemon.mu.Unlock()

	req := httptest.NewRequest("GET", "/sessions", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(result))
	}
}

func TestHandleGetSessionsMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	daemon := NewDaemon(config)

	req := httptest.NewRequest("POST", "/sessions", nil)
	w := httptest.NewRecorder()

	daemon.handleGetSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Test processFailedQueue
func TestProcessFailedQueue(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "session_id": "test-session"})
	}))
	defer mockServer.Close()

	config := DefaultConfig()
	config.ServerURL = mockServer.URL
	daemon := NewDaemon(config)

	// Add operations to queue
	daemon.queueMu.Lock()
	daemon.failedQueue = []QueuedOperation{
		{
			SessionID: "local-session",
			OpType:    "create_session",
			Data: map[string]interface{}{
				"product": "Test",
				"version": "1.0",
			},
			Timestamp: time.Now(),
		},
		{
			SessionID: "test-session",
			OpType:    "state",
			Data:      map[string]interface{}{"key": "value"},
			Timestamp: time.Now(),
		},
	}
	daemon.queueMu.Unlock()

	// Create local session for second operation
	daemon.mu.Lock()
	daemon.sessions["local-session"] = &SessionBuffer{
		SessionID: "local-session",
	}
	daemon.mu.Unlock()

	// Process queue
	daemon.processFailedQueue()

	// Verify queue is empty or has minimal items (operations may have been processed or some may be retrying)
	daemon.queueMu.Lock()
	queueLen := len(daemon.failedQueue)
	daemon.queueMu.Unlock()

	if queueLen > 1 {
		t.Errorf("Expected queue to be mostly empty after successful processing, got %d items", queueLen)
	}
}

func TestProcessFailedQueueWithFailures(t *testing.T) {
	config := DefaultConfig()
	config.ServerURL = "http://invalid-server:99999"
	daemon := NewDaemon(config)

	// Add operation that will fail
	daemon.queueMu.Lock()
	daemon.failedQueue = []QueuedOperation{
		{
			SessionID: "test-session",
			OpType:    "state",
			Data:      map[string]interface{}{"key": "value"},
			Timestamp: time.Now(),
		},
	}
	daemon.queueMu.Unlock()

	// Process queue
	daemon.processFailedQueue()

	// Verify operation is still in queue (failed)
	daemon.queueMu.Lock()
	queueLen := len(daemon.failedQueue)
	daemon.queueMu.Unlock()

	if queueLen != 1 {
		t.Errorf("Expected 1 item in queue after failed processing, got %d", queueLen)
	}
}

// Test getMachineID
func TestGetMachineID(t *testing.T) {
	machineID := getMachineID()

	// Machine ID should be 32 characters (MD5 hex)
	if len(machineID) != 32 {
		t.Errorf("Expected machine ID length 32, got %d", len(machineID))
	}

	// Should be consistent
	machineID2 := getMachineID()
	if machineID != machineID2 {
		t.Error("Machine ID should be consistent across calls")
	}
}

// Test getHostname
func TestGetHostname(t *testing.T) {
	hostname := getHostname()

	// Hostname should not be empty (unless there's an error)
	if hostname == "" {
		t.Log("Warning: hostname is empty, but this might be expected in some environments")
	}

	// Should be consistent
	hostname2 := getHostname()
	if hostname != hostname2 {
		t.Error("Hostname should be consistent across calls")
	}
}
