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
		StateUpdates: []map[string]interface{}{
			{"key": "value1"},
			{"key": "value2"},
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
