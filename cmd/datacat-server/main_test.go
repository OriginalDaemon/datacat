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

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()

	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.sessions == nil {
		t.Fatal("sessions map is nil")
	}
	if store.db == nil {
		t.Fatal("database is nil")
	}
}

func TestCreateSession(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")
	if session == nil {
		t.Fatal("CreateSession returned nil")
	}
	if session.ID == "" {
		t.Error("Session ID is empty")
	}
	if !session.Active {
		t.Error("Session should be active")
	}
	if session.State == nil {
		t.Error("Session state is nil")
	}

	// Verify session is in store
	retrieved, ok := store.GetSession(session.ID)
	if !ok {
		t.Error("Session not found in store")
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

func TestGetSession(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Get existing session
	retrieved, ok := store.GetSession(session.ID)
	if !ok {
		t.Error("Session not found")
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}

	// Try to get non-existent session
	_, ok = store.GetSession("non-existent")
	if ok {
		t.Error("Expected non-existent session to not be found")
	}
}

func TestGetAllSessions(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create multiple sessions
	session1 := store.CreateSession("TestProduct", "1.0.0", "", "")
	session2 := store.CreateSession("TestProduct", "1.0.0", "", "")

	sessions := store.GetAllSessions()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify both sessions are present
	found1, found2 := false, false
	for _, s := range sessions {
		if s.ID == session1.ID {
			found1 = true
		}
		if s.ID == session2.ID {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Not all sessions found")
	}
}

func TestUpdateState(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	newState := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	err = store.UpdateState(session.ID, StateUpdateInput{
		Timestamp: nil,
		State:     newState,
	})
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Verify state was updated
	retrieved, _ := store.GetSession(session.ID)
	if retrieved.State["key1"] != "value1" {
		t.Errorf("Expected key1 to be value1, got %v", retrieved.State["key1"])
	}
	// Since we directly set the state without JSON marshaling, it remains as int
	if retrieved.State["key2"] != 123 {
		t.Errorf("Expected key2 to be 123, got %v (type %T)", retrieved.State["key2"], retrieved.State["key2"])
	}

	// Check state history
	if len(retrieved.StateHistory) == 0 {
		t.Error("State history should not be empty")
	}
}

func TestAddEvent(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	eventData := map[string]interface{}{
		"message": "test event",
	}

	err = store.AddEvent(session.ID, EventInput{
		Name:     "test_event",
		Category: "info",
		Group:    "test",
		Labels:   []string{"tag1"},
		Message:  "test message",
		Data:     eventData,
	})
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Verify event was logged
	retrieved, _ := store.GetSession(session.ID)
	if len(retrieved.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(retrieved.Events))
	}
	if retrieved.Events[0].Name != "test_event" {
		t.Errorf("Expected event name test_event, got %s", retrieved.Events[0].Name)
	}
}

func TestAddMetric(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	err = store.AddMetric(session.ID, MetricInput{
		Name:  "cpu_usage",
		Value: 75.5,
		Tags:  []string{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("AddMetric failed: %v", err)
	}

	// Verify metric was logged
	retrieved, _ := store.GetSession(session.ID)
	if len(retrieved.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(retrieved.Metrics))
	}
	if retrieved.Metrics[0].Name != "cpu_usage" {
		t.Errorf("Expected metric name cpu_usage, got %s", retrieved.Metrics[0].Name)
	}
	if retrieved.Metrics[0].Value != 75.5 {
		t.Errorf("Expected metric value 75.5, got %f", retrieved.Metrics[0].Value)
	}
}

func TestEndSession(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	err = store.EndSession(session.ID)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	// Verify session was ended
	retrieved, _ := store.GetSession(session.ID)
	if retrieved.Active {
		t.Error("Session should not be active")
	}
	if retrieved.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestDeepMerge(t *testing.T) {
	dst := map[string]interface{}{
		"a": "value_a",
		"b": map[string]interface{}{
			"b1": "value_b1",
			"b2": "value_b2",
		},
	}

	src := map[string]interface{}{
		"b": map[string]interface{}{
			"b2": "new_value_b2",
			"b3": "value_b3",
		},
		"c": "value_c",
	}

	deepMerge(dst, src)

	if dst["a"] != "value_a" {
		t.Error("Expected a to remain unchanged")
	}
	if dst["c"] != "value_c" {
		t.Error("Expected c to be added")
	}

	b := dst["b"].(map[string]interface{})
	if b["b1"] != "value_b1" {
		t.Error("Expected b.b1 to remain unchanged")
	}
	if b["b2"] != "new_value_b2" {
		t.Error("Expected b.b2 to be updated")
	}
	if b["b3"] != "value_b3" {
		t.Error("Expected b.b3 to be added")
	}
}

func TestDeepMergeNonMap(t *testing.T) {
	dst := map[string]interface{}{
		"a": "value_a",
		"b": "value_b",
	}

	src := map[string]interface{}{
		"b": map[string]interface{}{
			"nested": "value",
		},
	}

	deepMerge(dst, src)

	// When src value is a map and dst value is not, src should replace dst
	if _, ok := dst["b"].(map[string]interface{}); !ok {
		t.Error("Expected b to be replaced with map")
	}
}

func TestDeepCopyState(t *testing.T) {
	original := map[string]interface{}{
		"a": "value",
		"b": map[string]interface{}{
			"b1": "nested",
		},
	}

	copied := deepCopyState(original)

	// Modify copy
	copied["a"] = "modified"
	copied["b"].(map[string]interface{})["b1"] = "modified_nested"

	// Original should remain unchanged
	if original["a"] != "value" {
		t.Error("Original should not be modified")
	}
	if original["b"].(map[string]interface{})["b1"] != "nested" {
		t.Error("Nested value in original should not be modified")
	}
}

func TestDeepCopyStateWithSlices(t *testing.T) {
	original := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
		"nested": []interface{}{
			map[string]interface{}{"key": "value1"},
			map[string]interface{}{"key": "value2"},
		},
	}

	copied := deepCopyState(original)

	// Modify copy
	copied["items"].([]interface{})[0] = "modified"
	copied["nested"].([]interface{})[0].(map[string]interface{})["key"] = "modified"

	// Original should remain unchanged
	if original["items"].([]interface{})[0] != "a" {
		t.Error("Original array should not be modified")
	}
	if original["nested"].([]interface{})[0].(map[string]interface{})["key"] != "value1" {
		t.Error("Original nested array should not be modified")
	}
}

func TestCleanupOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.RetentionDays = 1
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create an old session
	oldSession := store.CreateSession("TestProduct", "1.0.0", "", "")
	oldTime := time.Now().AddDate(0, 0, -2)
	store.mu.Lock()
	oldSession.CreatedAt = oldTime
	oldSession.EndedAt = &oldTime
	oldSession.UpdatedAt = oldTime
	oldSession.Active = false
	store.mu.Unlock()

	// Create a recent session
	recentSession := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Run cleanup
	removed, err := store.CleanupOldSessions()
	if err != nil {
		t.Fatalf("CleanupOldSessions failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("Expected 1 session to be removed, got %d", removed)
	}

	// Check that old session is removed
	_, ok := store.GetSession(oldSession.ID)
	if ok {
		t.Error("Old session should have been removed")
	}

	// Check that recent session still exists
	_, ok = store.GetSession(recentSession.ID)
	if !ok {
		t.Error("Recent session should still exist")
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()

	// Create store and session
	store1, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	session := store1.CreateSession("TestProduct", "1.0.0", "", "")
	sessionID := session.ID

	// Update state
	store1.UpdateState(sessionID, StateUpdateInput{
		State: map[string]interface{}{"key": "value"},
	})

	// Wait for async save
	time.Sleep(200 * time.Millisecond)

	// Close store
	store1.Close()

	// Reopen store
	store2, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	// Verify session persisted
	retrieved, ok := store2.GetSession(sessionID)
	if !ok {
		t.Fatal("Session not found after reload")
	}
	if retrieved.ID != sessionID {
		t.Errorf("Expected ID %s, got %s", sessionID, retrieved.ID)
	}
	if retrieved.State["key"] != "value" {
		t.Errorf("Expected state to be persisted")
	}
}

func TestComprehensivePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()

	// Create store and session
	store1, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	session := store1.CreateSession("TestProduct", "1.0.0", "", "")
	sessionID := session.ID

	// Update state with nested data
	store1.UpdateState(sessionID, StateUpdateInput{
		State: map[string]interface{}{
			"status": "running",
			"config": map[string]interface{}{
				"mode":  "test",
				"count": 42,
			},
		},
	})

	// Add event
	store1.AddEvent(sessionID, EventInput{
		Name:     "test_event",
		Category: "info",
		Group:    "test",
		Labels:   []string{"tag1"},
		Message:  "test message",
		Data:     map[string]interface{}{"extra": "data"},
	})

	// Add metric
	store1.AddMetric(sessionID, MetricInput{
		Name:  "cpu_usage",
		Value: 75.5,
		Tags:  []string{"host:server1"},
	})

	// Wait for async saves
	time.Sleep(500 * time.Millisecond)

	// Close store
	store1.Close()

	// Reopen store
	store2, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	// Verify session persisted
	retrieved, ok := store2.GetSession(sessionID)
	if !ok {
		t.Fatal("Session not found after reload")
	}
	if retrieved.ID != sessionID {
		t.Errorf("Expected ID %s, got %s", sessionID, retrieved.ID)
	}

	// Verify state persisted
	if retrieved.State["status"] != "running" {
		t.Errorf("Expected status 'running', got %v", retrieved.State["status"])
	}
	configMap, ok := retrieved.State["config"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected config to be a map")
	}
	if configMap["mode"] != "test" {
		t.Errorf("Expected config.mode 'test', got %v", configMap["mode"])
	}

	// Verify event persisted
	if len(retrieved.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(retrieved.Events))
	}
	if retrieved.Events[0].Name != "test_event" {
		t.Errorf("Expected event name 'test_event', got %s", retrieved.Events[0].Name)
	}
	if retrieved.Events[0].Data["extra"] != "data" {
		t.Errorf("Expected event data to be persisted")
	}

	// Verify metric persisted
	if len(retrieved.Metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(retrieved.Metrics))
	}
	if retrieved.Metrics[0].Name != "cpu_usage" {
		t.Errorf("Expected metric name 'cpu_usage', got %s", retrieved.Metrics[0].Name)
	}
	if retrieved.Metrics[0].Value != 75.5 {
		t.Errorf("Expected metric value 75.5, got %f", retrieved.Metrics[0].Value)
	}

	// Add more data after reopening
	store2.UpdateState(sessionID, StateUpdateInput{
		State: map[string]interface{}{"status": "completed"},
	})
	store2.AddEvent(sessionID, EventInput{
		Name:     "after_reload",
		Category: "info",
		Group:    "test",
		Message:  "after reload",
		Data:     map[string]interface{}{},
	})

	// Wait for async saves
	time.Sleep(500 * time.Millisecond)

	// Verify new data was added
	retrieved2, _ := store2.GetSession(sessionID)
	if retrieved2.State["status"] != "completed" {
		t.Errorf("Expected status 'completed' after update, got %v", retrieved2.State["status"])
	}
	if len(retrieved2.Events) != 2 {
		t.Errorf("Expected 2 events after adding new one, got %d", len(retrieved2.Events))
	}
}

func TestAddEventErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Try to add event to non-existent session
	err = store.AddEvent("non-existent", EventInput{
		Name: "test",
		Data: map[string]interface{}{},
	})
	if err == nil {
		t.Error("Expected error when adding event to non-existent session")
	}
}

func TestAddMetricErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Try to add metric to non-existent session
	err = store.AddMetric("non-existent", MetricInput{
		Name:  "test",
		Value: 0.0,
	})
	if err == nil {
		t.Error("Expected error when adding metric to non-existent session")
	}
}

func TestUpdateStateErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Try to update state of non-existent session
	err = store.UpdateState("non-existent", StateUpdateInput{
		State: map[string]interface{}{},
	})
	if err == nil {
		t.Error("Expected error when updating state of non-existent session")
	}
}

func TestEndSessionErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Try to end non-existent session
	err = store.EndSession("non-existent")
	if err == nil {
		t.Error("Expected error when ending non-existent session")
	}
}

func TestHTTPHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	var err error
	store, err = NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Test handleSessions POST - create session
	t.Run("CreateSession", func(t *testing.T) {
		reqBody := map[string]string{
			"product": "TestProduct",
			"version": "1.0.0",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/sessions", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["session_id"] == "" {
			t.Error("Expected session_id in response")
		}
	})

	// Test handleSessions with invalid method
	t.Run("CreateSessionInvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sessions", nil)
		w := httptest.NewRecorder()

		handleSessions(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	// Create a session for subsequent tests
	session := store.CreateSession("TestProduct", "1.0.0", "", "")
	sessionID := session.ID

	// Test handleSessionOperations GET - get session
	t.Run("GetSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sessions/"+sessionID, nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var retrieved Session
		json.NewDecoder(w.Body).Decode(&retrieved)

		if retrieved.ID != sessionID {
			t.Errorf("Expected session ID %s, got %s", sessionID, retrieved.ID)
		}
	})

	// Test get non-existent session
	t.Run("GetNonExistentSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sessions/non-existent", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test update state
	t.Run("UpdateState", func(t *testing.T) {
		stateData := map[string]interface{}{"key": "value"}
		body, _ := json.Marshal(stateData)

		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/state", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test update state with invalid JSON
	t.Run("UpdateStateInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/state", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test update state for non-existent session
	t.Run("UpdateStateNonExistent", func(t *testing.T) {
		stateData := map[string]interface{}{"key": "value"}
		body, _ := json.Marshal(stateData)

		req := httptest.NewRequest("POST", "/api/sessions/non-existent/state", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test add event
	t.Run("AddEvent", func(t *testing.T) {
		eventData := map[string]interface{}{
			"name": "test_event",
			"data": map[string]interface{}{"msg": "hello"},
		}
		body, _ := json.Marshal(eventData)

		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/events", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test add event with invalid JSON
	t.Run("AddEventInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/events", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test add event for non-existent session
	t.Run("AddEventNonExistent", func(t *testing.T) {
		eventData := map[string]interface{}{
			"name": "test_event",
			"data": map[string]interface{}{},
		}
		body, _ := json.Marshal(eventData)

		req := httptest.NewRequest("POST", "/api/sessions/non-existent/events", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test add metric
	t.Run("AddMetric", func(t *testing.T) {
		metricData := map[string]interface{}{
			"name":  "cpu_usage",
			"value": 75.5,
			"tags":  []string{"tag1"},
		}
		body, _ := json.Marshal(metricData)

		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/metrics", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test add metric with invalid JSON
	t.Run("AddMetricInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/metrics", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test add metric for non-existent session
	t.Run("AddMetricNonExistent", func(t *testing.T) {
		metricData := map[string]interface{}{
			"name":  "metric",
			"value": 1.0,
		}
		body, _ := json.Marshal(metricData)

		req := httptest.NewRequest("POST", "/api/sessions/non-existent/metrics", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test end session
	t.Run("EndSession", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/end", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test end non-existent session
	t.Run("EndNonExistentSession", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/non-existent/end", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test invalid operation
	t.Run("InvalidOperation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/invalid", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test missing session ID
	t.Run("MissingSessionID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sessions//state", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test handleGetAllSessions
	t.Run("GetAllSessions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data/sessions", nil)
		w := httptest.NewRecorder()

		handleGetAllSessions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var sessions []*Session
		json.NewDecoder(w.Body).Decode(&sessions)

		if len(sessions) == 0 {
			t.Error("Expected at least one session")
		}
	})

	// Test handleGetAllSessions with invalid method
	t.Run("GetAllSessionsInvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/data/sessions", nil)
		w := httptest.NewRecorder()

		handleGetAllSessions(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestStartCleanupRoutine(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.CleanupIntervalHours = 0 // Set to 0 for immediate cleanup in test
	var err error
	store, err = NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Start cleanup routine (it will run once since interval is 0)
	// We can't easily test the goroutine, but we can test that it doesn't panic
	go store.StartCleanupRoutine()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
}

func TestNewStoreError(t *testing.T) {
	// Try to create store with invalid path (use a path with invalid characters)
	// On Windows, paths with certain characters like * are invalid
	// On Unix, we use a path that requires root permissions
	invalidPath := "\x00invalid\x00path" // Null bytes are invalid in paths on all systems
	_, err := NewStore(invalidPath, DefaultConfig())
	if err == nil {
		t.Error("Expected error when creating store with invalid path")
	}
}

func TestCleanupOldSessionsEmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Cleanup with no sessions
	removed, err := store.CleanupOldSessions()
	if err != nil {
		t.Fatalf("CleanupOldSessions failed: %v", err)
	}

	if removed != 0 {
		t.Errorf("Expected 0 sessions removed, got %d", removed)
	}
}

func TestGetSessionNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	_, ok := store.GetSession("non-existent")
	if ok {
		t.Error("Expected false for non-existent session")
	}
}

func TestSaveSessionToDBError(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Close the database to cause errors
	store.db.Close()

	session := &Session{
		ID:        "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
		State:     map[string]interface{}{},
	}

	// This should fail because DB is closed
	err = store.saveSessionToDB(session)
	if err == nil {
		t.Error("Expected error when saving to closed database")
	}
}

func TestCloseStore(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Errorf("Close should not fail: %v", err)
	}

	// Closing again should not panic
	_ = store.Close()
	// BadgerDB returns error when closing an already closed DB, which is expected
}

// Test StartCleanupRoutine execution
func TestStartCleanupRoutineExecution(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.CleanupInterval = 50 * time.Millisecond // Short interval for testing
	config.RetentionDays = 0                       // Clean up sessions immediately

	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create an old session
	session := store.CreateSession("TestProduct", "1.0.0", "", "")
	session.CreatedAt = time.Now().Add(-2 * 24 * time.Hour)

	// Start cleanup routine
	store.StartCleanupRoutine()

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Session should be cleaned up
	store.mu.RLock()
	_, exists := store.sessions[session.ID]
	store.mu.RUnlock()

	if exists {
		t.Error("Expected old session to be cleaned up")
	}
}

func TestUpdateHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Update heartbeat
	err = store.UpdateHeartbeat(session.ID)
	if err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	// Verify heartbeat was updated
	retrieved, _ := store.GetSession(session.ID)
	if retrieved.LastHeartbeat == nil {
		t.Error("LastHeartbeat should be set")
	}
	if !retrieved.Active {
		t.Error("Session should be active after heartbeat")
	}
}

func TestUpdateHeartbeatNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Try to update heartbeat of non-existent session
	err = store.UpdateHeartbeat("non-existent")
	if err == nil {
		t.Error("Expected error when updating heartbeat of non-existent session")
	}
}

func TestActiveStatusBasedOnHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1 // Short timeout for testing
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Send heartbeat
	store.UpdateHeartbeat(session.ID)

	// Session should be active
	retrieved, _ := store.GetSession(session.ID)
	if !retrieved.Active {
		t.Error("Session should be active after recent heartbeat")
	}

	// Wait for heartbeat timeout
	time.Sleep(2 * time.Second)

	// Session should now be inactive
	retrieved, _ = store.GetSession(session.ID)
	if retrieved.Active {
		t.Error("Session should be inactive after heartbeat timeout")
	}
}

func TestActiveStatusWithoutHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Session without heartbeat should still show initial active status
	retrieved, _ := store.GetSession(session.ID)
	if !retrieved.Active {
		t.Error("Session should be active initially even without heartbeat")
	}
}

func TestActiveStatusAfterEnd(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Send heartbeat
	store.UpdateHeartbeat(session.ID)

	// End session
	store.EndSession(session.ID)

	// Session should be inactive even with recent heartbeat
	retrieved, _ := store.GetSession(session.ID)
	if retrieved.Active {
		t.Error("Session should be inactive after being ended")
	}
}

func TestHeartbeatHTTPHandler(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	var err error
	store, err = NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")
	sessionID := session.ID

	// Test heartbeat endpoint
	t.Run("Heartbeat", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/"+sessionID+"/heartbeat", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify heartbeat was recorded
		retrieved, _ := store.GetSession(sessionID)
		if retrieved.LastHeartbeat == nil {
			t.Error("LastHeartbeat should be set after heartbeat call")
		}
	})

	// Test heartbeat for non-existent session
	t.Run("HeartbeatNonExistent", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sessions/non-existent/heartbeat", nil)
		w := httptest.NewRecorder()

		handleSessionOperations(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestMachineTracking(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	machineID := "test-machine-123"
	hostname := "test-host"

	// Create session with machine info
	session := store.CreateSession("TestProduct", "1.0.0", machineID, hostname)

	if session.MachineID != machineID {
		t.Errorf("Expected MachineID %s, got %s", machineID, session.MachineID)
	}
	if session.Hostname != hostname {
		t.Errorf("Expected Hostname %s, got %s", hostname, session.Hostname)
	}
}

func TestCrashDetection(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1 // Short timeout for testing
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	machineID := "test-machine-456"

	// Create first session
	session1 := store.CreateSession("TestProduct", "1.0.0", machineID, "host1")

	// Send heartbeat then stop
	store.UpdateHeartbeat(session1.ID)

	// Wait for heartbeat timeout to mark as suspended
	time.Sleep(2 * time.Second)

	retrieved1, _ := store.GetSession(session1.ID)
	if !retrieved1.Suspended {
		t.Error("Session should be marked as suspended after heartbeat timeout")
	}

	// Create new session from same machine
	session2 := store.CreateSession("TestProduct", "1.0.0", machineID, "host1")

	// First session should now be marked as crashed
	retrieved1Again, _ := store.GetSession(session1.ID)
	if !retrieved1Again.Crashed {
		t.Error("Old session should be marked as crashed when machine returns with new session")
	}
	if retrieved1Again.Suspended {
		t.Error("Crashed session should not be suspended")
	}

	// Verify crash event was logged
	hasCrashEvent := false
	for _, event := range retrieved1Again.Events {
		if event.Name == "session_crashed_detected" {
			hasCrashEvent = true
			break
		}
	}
	if !hasCrashEvent {
		t.Error("Expected session_crashed_detected event to be logged")
	}

	// New session should not be crashed
	if session2.Crashed {
		t.Error("New session should not be marked as crashed")
	}
}

func TestSuspendedStatus(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Send heartbeat
	store.UpdateHeartbeat(session.ID)

	// Session should be active
	retrieved, _ := store.GetSession(session.ID)
	if !retrieved.Active {
		t.Error("Session should be active after recent heartbeat")
	}
	if retrieved.Suspended {
		t.Error("Session should not be suspended with recent heartbeat")
	}

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Session should now be suspended
	retrieved, _ = store.GetSession(session.ID)
	if retrieved.Active {
		t.Error("Session should not be active after heartbeat timeout")
	}
	if !retrieved.Suspended {
		t.Error("Session should be suspended after heartbeat timeout")
	}

	// Resume heartbeats
	store.UpdateHeartbeat(session.ID)

	// Session should be active again and not suspended
	retrieved, _ = store.GetSession(session.ID)
	if !retrieved.Active {
		t.Error("Session should be active after heartbeat resumes")
	}
	if retrieved.Suspended {
		t.Error("Session should not be suspended after heartbeat resumes")
	}
}

func TestHungTracking(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Session should not be hung initially
	if session.Hung {
		t.Error("Session should not be hung initially")
	}

	// Log hung event
	err = store.AddEvent(session.ID, EventInput{
		Name:     "application_appears_hung",
		Category: "error",
		Group:    "datacat.daemon",
		Labels:   []string{"heartbeat", "hung"},
		Message:  "Application appears hung",
		Data: map[string]interface{}{
			"last_heartbeat": time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Session should now be marked as hung
	retrieved, _ := store.GetSession(session.ID)
	if !retrieved.Hung {
		t.Error("Session should be marked as hung after application_appears_hung event")
	}

	// Log recovery event
	err = store.AddEvent(session.ID, EventInput{
		Name:     "application_recovered",
		Category: "info",
		Group:    "datacat.daemon",
		Labels:   []string{"heartbeat", "recovery"},
		Message:  "Application recovered",
		Data:     map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Session should no longer be hung
	retrieved, _ = store.GetSession(session.ID)
	if retrieved.Hung {
		t.Error("Session should not be hung after application_recovered event")
	}

	// Verify both events were logged
	if len(retrieved.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(retrieved.Events))
	}
}

func TestHungWhileCrashed(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	machineID := "test-machine-789"

	// Create session
	session1 := store.CreateSession("TestProduct", "1.0.0", machineID, "host1")

	// Send heartbeat
	store.UpdateHeartbeat(session1.ID)

	// Log hung event
	store.AddEvent(session1.ID, EventInput{
		Name:     "application_appears_hung",
		Category: "error",
		Group:    "datacat.daemon",
		Labels:   []string{"heartbeat", "hung"},
		Message:  "Application appears hung",
		Data:     map[string]interface{}{},
	})

	// Verify it's hung
	retrieved, _ := store.GetSession(session1.ID)
	if !retrieved.Hung {
		t.Error("Session should be hung")
	}

	// Wait for heartbeat timeout
	time.Sleep(2 * time.Second)

	// Create new session from same machine (simulating crash)
	store.CreateSession("TestProduct", "1.0.0", machineID, "host1")

	// Original session should be both crashed AND hung
	retrieved1, _ := store.GetSession(session1.ID)
	if !retrieved1.Crashed {
		t.Error("Session should be marked as crashed")
	}
	if !retrieved1.Hung {
		t.Error("Session should still be hung (crashed while hung)")
	}
}

func TestSessionWithoutMachineID(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create sessions without machine ID
	session1 := store.CreateSession("TestProduct", "1.0.0", "", "")
	session2 := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Neither should be marked as crashed
	if session1.Crashed {
		t.Error("Session without machine ID should not be marked as crashed")
	}
	if session2.Crashed {
		t.Error("Session without machine ID should not trigger crash detection")
	}
}

func TestMultipleMachines(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.HeartbeatTimeoutSeconds = 1
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	machine1ID := "machine-001"
	machine2ID := "machine-002"

	// Create sessions on different machines
	session1 := store.CreateSession("TestProduct", "1.0.0", machine1ID, "host1")
	session2 := store.CreateSession("TestProduct", "1.0.0", machine2ID, "host2")

	// Send heartbeats
	store.UpdateHeartbeat(session1.ID)
	store.UpdateHeartbeat(session2.ID)

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Both should be suspended
	r1, _ := store.GetSession(session1.ID)
	r2, _ := store.GetSession(session2.ID)
	if !r1.Suspended || !r2.Suspended {
		t.Error("Both sessions should be suspended")
	}

	// New session on machine1
	store.CreateSession("TestProduct", "1.0.0", machine1ID, "host1")

	// Only session1 should be crashed, not session2
	r1, _ = store.GetSession(session1.ID)
	r2, _ = store.GetSession(session2.ID)

	if !r1.Crashed {
		t.Error("Session on machine1 should be crashed")
	}
	if r2.Crashed {
		t.Error("Session on machine2 should not be crashed")
	}
	if !r2.Suspended {
		t.Error("Session on machine2 should still be suspended")
	}
}

func TestCreateSessionHTTPWithMachineInfo(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	var err error
	store, err = NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	reqData := map[string]interface{}{
		"product":    "TestProduct",
		"version":    "1.0.0",
		"machine_id": "test-machine-http",
		"hostname":   "test-host-http",
	}
	body, _ := json.Marshal(reqData)

	req := httptest.NewRequest("POST", "/api/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	sessionID := response["session_id"]

	if sessionID == "" {
		t.Fatal("Expected session_id in response")
	}

	// Verify machine info was stored
	session, _ := store.GetSession(sessionID)
	if session.MachineID != "test-machine-http" {
		t.Errorf("Expected MachineID test-machine-http, got %s", session.MachineID)
	}
	if session.Hostname != "test-host-http" {
		t.Errorf("Expected Hostname test-host-http, got %s", session.Hostname)
	}
}

func TestExceptionEvent(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Add an exception event with all fields
	stacktrace := []string{
		"Traceback (most recent call last):",
		"  File \"test.py\", line 10, in <module>",
		"    raise ValueError(\"test error\")",
		"ValueError: test error",
	}

	err = store.AddEvent(session.ID, EventInput{
		Name:           "exception",
		Category:       "error",
		Group:          "exception",
		Labels:         []string{"exception", "ValueError"},
		Message:        "test error",
		Data:           map[string]interface{}{"extra": "data"},
		ExceptionType:  "ValueError",
		ExceptionMsg:   "test error",
		Stacktrace:     stacktrace,
		SourceFile:     "test.py",
		SourceLine:     10,
		SourceFunction: "<module>",
	})
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Verify exception event was logged with all fields
	retrieved, _ := store.GetSession(session.ID)
	if len(retrieved.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(retrieved.Events))
	}

	event := retrieved.Events[0]
	if event.Name != "exception" {
		t.Errorf("Expected event name exception, got %s", event.Name)
	}
	if event.ExceptionType != "ValueError" {
		t.Errorf("Expected ExceptionType ValueError, got %s", event.ExceptionType)
	}
	if event.ExceptionMsg != "test error" {
		t.Errorf("Expected ExceptionMsg 'test error', got %s", event.ExceptionMsg)
	}
	if len(event.Stacktrace) != 4 {
		t.Errorf("Expected 4 stacktrace lines, got %d", len(event.Stacktrace))
	}
	if event.SourceFile != "test.py" {
		t.Errorf("Expected SourceFile test.py, got %s", event.SourceFile)
	}
	if event.SourceLine != 10 {
		t.Errorf("Expected SourceLine 10, got %d", event.SourceLine)
	}
	if event.SourceFunction != "<module>" {
		t.Errorf("Expected SourceFunction <module>, got %s", event.SourceFunction)
	}
}

func TestStateDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Set initial state
	initialState := map[string]interface{}{
		"user":  "alice",
		"count": 5,
		"data":  map[string]interface{}{"key1": "value1", "key2": "value2"},
	}
	err = store.UpdateState(session.ID, StateUpdateInput{
		State: initialState,
	})
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Verify initial state
	retrieved, _ := store.GetSession(session.ID)
	if retrieved.State["user"] != "alice" {
		t.Errorf("Expected user alice, got %v", retrieved.State["user"])
	}
	// Compare count value (could be int or float64 depending on storage)
	countVal, ok := retrieved.State["count"]
	if !ok {
		t.Errorf("Expected count key to exist")
	} else {
		// Convert to float64 for comparison
		var countFloat float64
		switch v := countVal.(type) {
		case int:
			countFloat = float64(v)
		case float64:
			countFloat = v
		default:
			t.Errorf("Expected count to be numeric, got %T", v)
		}
		if countFloat != 5.0 {
			t.Errorf("Expected count 5, got %v", countFloat)
		}
	}

	// Delete a key by setting it to nil
	deleteUpdate := map[string]interface{}{
		"user": nil,
	}
	err = store.UpdateState(session.ID, StateUpdateInput{
		State: deleteUpdate,
	})
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Verify key was deleted
	retrieved, _ = store.GetSession(session.ID)
	if _, exists := retrieved.State["user"]; exists {
		t.Errorf("Expected user key to be deleted, but it still exists")
	}
	// Verify count remains unchanged
	countVal, ok = retrieved.State["count"]
	if !ok {
		t.Errorf("Expected count key to still exist")
	} else {
		var countFloat float64
		switch v := countVal.(type) {
		case int:
			countFloat = float64(v)
		case float64:
			countFloat = v
		default:
			t.Errorf("Expected count to be numeric, got %T", v)
		}
		if countFloat != 5.0 {
			t.Errorf("Expected count to remain 5, got %v", countFloat)
		}
	}

	// Delete nested key
	nestedDeleteUpdate := map[string]interface{}{
		"data": map[string]interface{}{
			"key1": nil,
		},
	}
	err = store.UpdateState(session.ID, StateUpdateInput{
		State: nestedDeleteUpdate,
	})
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Verify nested key was deleted
	retrieved, _ = store.GetSession(session.ID)
	dataMap, ok := retrieved.State["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data to be a map")
	}
	if _, exists := dataMap["key1"]; exists {
		t.Errorf("Expected data.key1 to be deleted, but it still exists")
	}
	if dataMap["key2"] != "value2" {
		t.Errorf("Expected data.key2 to remain value2, got %v", dataMap["key2"])
	}
}

func TestEventWithLevelCategoryLabels(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	store, err := NewStore(tmpDir, config)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	session := store.CreateSession("TestProduct", "1.0.0", "", "")

	// Add event with level, category, labels, and message
	err = store.AddEvent(session.ID, EventInput{
		Name:     "custom_event",
		Category: "warning",
		Group:    "my.component",
		Labels:   []string{"tag1", "tag2", "important"},
		Message:  "This is a warning message",
		Data:     map[string]interface{}{"detail": "some detail"},
	})
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Verify event was logged with all fields
	retrieved, _ := store.GetSession(session.ID)
	if len(retrieved.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(retrieved.Events))
	}

	event := retrieved.Events[0]
	if event.Name != "custom_event" {
		t.Errorf("Expected event name custom_event, got %s", event.Name)
	}
	if event.Category != "warning" {
		t.Errorf("Expected category warning, got %s", event.Category)
	}
	if event.Group != "my.component" {
		t.Errorf("Expected group my.component, got %s", event.Group)
	}
	if len(event.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(event.Labels))
	}
	if event.Labels[0] != "tag1" || event.Labels[1] != "tag2" || event.Labels[2] != "important" {
		t.Errorf("Unexpected labels: %v", event.Labels)
	}
	if event.Message != "This is a warning message" {
		t.Errorf("Expected message 'This is a warning message', got %s", event.Message)
	}
}

func TestInitLogging(t *testing.T) {
	// Test with empty log file (should create default log file)
	config := &Config{
		DataPath:                "./test_data",
		RetentionDays:           7,
		CleanupIntervalHours:    24,
		ServerPort:              "9090",
		HeartbeatTimeoutSeconds: 60,
		LogFile:                 "",
	}

	logPath, cleanup, err := initLogging(config)

	if err != nil {
		t.Errorf("Expected no error with empty log file, got %v", err)
	}

	// When LogFile is empty, initLogging creates a default log file with timestamp and PID
	if logPath == "" {
		t.Errorf("Expected non-empty log path (default log file), got empty")
	}

	// Cleanup should not panic
	cleanup()
	
	// Clean up the created log file
	if logPath != "" {
		os.Remove(logPath)
	}
}
