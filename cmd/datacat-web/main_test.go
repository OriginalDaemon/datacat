package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/OriginalDaemon/datacat/client"
)

func TestShouldIncludeSession(t *testing.T) {
	session := &client.Session{
		State: map[string]interface{}{
			"status": "running",
			"tags":   []interface{}{"tag1", "tag2"},
		},
	}
	
	// Test "none" filter mode
	if !shouldIncludeSession(session, "none", "", "") {
		t.Error("Expected session to be included with 'none' filter")
	}
	
	// Test empty filter mode
	if !shouldIncludeSession(session, "", "", "") {
		t.Error("Expected session to be included with empty filter")
	}
	
	// Test current_state filter
	if !shouldIncludeSession(session, "current_state", "status", "running") {
		t.Error("Expected session to be included when state matches")
	}
	
	if shouldIncludeSession(session, "current_state", "status", "stopped") {
		t.Error("Expected session to be excluded when state doesn't match")
	}
	
	// Test state_array_contains filter
	if !shouldIncludeSession(session, "state_array_contains", "tags", "tag1") {
		t.Error("Expected session to be included when array contains value")
	}
	
	if shouldIncludeSession(session, "state_array_contains", "tags", "tag3") {
		t.Error("Expected session to be excluded when array doesn't contain value")
	}
}

func TestStateArrayContains(t *testing.T) {
	state := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2", "tag3"},
		"nested": map[string]interface{}{
			"values": []interface{}{"a", "b", "c"},
		},
	}
	
	// Test direct array
	if !stateArrayContains(state, "tags", "tag1") {
		t.Error("Expected to find tag1 in tags array")
	}
	
	if stateArrayContains(state, "tags", "tag4") {
		t.Error("Expected not to find tag4 in tags array")
	}
	
	// Test nested array
	if !stateArrayContains(state, "nested.values", "b") {
		t.Error("Expected to find b in nested.values array")
	}
	
	if stateArrayContains(state, "nested.values", "d") {
		t.Error("Expected not to find d in nested.values array")
	}
	
	// Test non-existent path
	if stateArrayContains(state, "nonexistent", "value") {
		t.Error("Expected false for non-existent path")
	}
	
	// Test non-array value
	if stateArrayContains(state, "nested", "value") {
		t.Error("Expected false for non-array value")
	}
}

func TestMatchesStateFilter(t *testing.T) {
	state := map[string]interface{}{
		"status": "running",
		"app": map[string]interface{}{
			"name":    "test-app",
			"version": "1.0",
		},
	}
	
	// Test simple key
	if !matchesStateFilter(state, "status", "running") {
		t.Error("Expected to match status=running")
	}
	
	if matchesStateFilter(state, "status", "stopped") {
		t.Error("Expected not to match status=stopped")
	}
	
	// Test nested key
	if !matchesStateFilter(state, "app.name", "test-app") {
		t.Error("Expected to match app.name=test-app")
	}
	
	if matchesStateFilter(state, "app.name", "other-app") {
		t.Error("Expected not to match app.name=other-app")
	}
	
	// Test non-existent key
	if matchesStateFilter(state, "nonexistent", "value") {
		t.Error("Expected false for non-existent key")
	}
	
	// Test invalid path
	if matchesStateFilter(state, "status.nested", "value") {
		t.Error("Expected false for invalid nested path")
	}
}

func TestSortSessions(t *testing.T) {
	now := time.Now()
	sessions := []*client.Session{
		{ID: "1", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour), Active: false, Events: []client.Event{{}, {}}, Metrics: []client.Metric{{}}},
		{ID: "2", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour), Active: true, Events: []client.Event{{}}, Metrics: []client.Metric{{}, {}}},
		{ID: "3", CreatedAt: now, UpdatedAt: now, Active: true, Events: []client.Event{{}, {}, {}}, Metrics: []client.Metric{{}, {}, {}}},
	}
	
	// Test sort by created_at ascending
	sortSessions(sessions, "created_at", "asc")
	if sessions[0].ID != "1" {
		t.Errorf("Expected first session to be 1, got %s", sessions[0].ID)
	}
	
	// Test sort by created_at descending
	sortSessions(sessions, "created_at", "desc")
	if sessions[0].ID != "3" {
		t.Errorf("Expected first session to be 3, got %s", sessions[0].ID)
	}
	
	// Test sort by updated_at ascending
	sortSessions(sessions, "updated_at", "asc")
	if sessions[0].ID != "2" {
		t.Errorf("Expected first session to be 2, got %s", sessions[0].ID)
	}
	
	// Test sort by status
	sortSessions(sessions, "status", "asc")
	if sessions[0].Active == false {
		t.Error("Expected active sessions to come first")
	}
	
	// Test sort by events count
	sortSessions(sessions, "events", "asc")
	if len(sessions[0].Events) != 1 {
		t.Errorf("Expected first session to have 1 event, got %d", len(sessions[0].Events))
	}
	
	// Test sort by metrics count
	sortSessions(sessions, "metrics", "desc")
	if len(sessions[0].Metrics) != 3 {
		t.Errorf("Expected first session to have 3 metrics, got %d", len(sessions[0].Metrics))
	}
	
	// Test default sort
	sortSessions(sessions, "invalid", "asc")
	// Should default to created_at
}

func TestFilterSessionsByState(t *testing.T) {
	sessions := []*client.Session{
		{ID: "1", State: map[string]interface{}{"status": "running", "app": "app1"}},
		{ID: "2", State: map[string]interface{}{"status": "stopped", "app": "app2"}},
		{ID: "3", State: map[string]interface{}{"status": "running", "app": "app1"}},
	}
	
	// Test valid JSON filter
	filterJSON := `{"status": "running"}`
	filtered := filterSessionsByState(sessions, filterJSON)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered sessions, got %d", len(filtered))
	}
	
	// Test more specific filter
	filterJSON2 := `{"status": "running", "app": "app1"}`
	filtered2 := filterSessionsByState(sessions, filterJSON2)
	if len(filtered2) != 2 {
		t.Errorf("Expected 2 filtered sessions, got %d", len(filtered2))
	}
	
	// Test invalid JSON (should return all)
	invalidJSON := `{invalid json}`
	filteredAll := filterSessionsByState(sessions, invalidJSON)
	if len(filteredAll) != 3 {
		t.Errorf("Expected all 3 sessions for invalid JSON, got %d", len(filteredAll))
	}
}

func TestMatchesStateHistory(t *testing.T) {
	session := &client.Session{
		State: map[string]interface{}{
			"status": "running",
			"app":    "test-app",
		},
	}
	
	// Test matching filter
	filter := map[string]interface{}{"status": "running"}
	if !matchesStateHistory(session, filter) {
		t.Error("Expected session to match filter")
	}
	
	// Test non-matching filter
	filter2 := map[string]interface{}{"status": "stopped"}
	if matchesStateHistory(session, filter2) {
		t.Error("Expected session not to match filter")
	}
	
	// Test partial match
	filter3 := map[string]interface{}{"status": "running", "app": "test-app"}
	if !matchesStateHistory(session, filter3) {
		t.Error("Expected session to match partial filter")
	}
}

func TestStateContainsAll(t *testing.T) {
	state := map[string]interface{}{
		"status": "running",
		"config": map[string]interface{}{
			"debug": true,
			"level": "info",
		},
		"tags": []interface{}{"tag1", "tag2"},
	}
	
	// Test simple match
	filter := map[string]interface{}{"status": "running"}
	if !stateContainsAll(state, filter) {
		t.Error("Expected state to contain filter")
	}
	
	// Test nested map match
	filter2 := map[string]interface{}{
		"config": map[string]interface{}{
			"debug": true,
		},
	}
	if !stateContainsAll(state, filter2) {
		t.Error("Expected state to contain nested filter")
	}
	
	// Test array match
	filter3 := map[string]interface{}{
		"tags": []interface{}{"tag1"},
	}
	if !stateContainsAll(state, filter3) {
		t.Error("Expected state to contain array filter")
	}
	
	// Test non-matching filter
	filter4 := map[string]interface{}{"status": "stopped"}
	if stateContainsAll(state, filter4) {
		t.Error("Expected state not to contain non-matching filter")
	}
	
	// Test missing key
	filter5 := map[string]interface{}{"missing": "value"}
	if stateContainsAll(state, filter5) {
		t.Error("Expected state not to contain filter with missing key")
	}
	
	// Test nested map mismatch
	filter6 := map[string]interface{}{
		"config": map[string]interface{}{
			"debug": false,
		},
	}
	if stateContainsAll(state, filter6) {
		t.Error("Expected state not to contain mismatched nested filter")
	}
	
	// Test type mismatch (filter is map, state is not)
	filter7 := map[string]interface{}{
		"status": map[string]interface{}{"nested": "value"},
	}
	if stateContainsAll(state, filter7) {
		t.Error("Expected state not to match when types differ")
	}
	
	// Test type mismatch (filter is array, state is not)
	filter8 := map[string]interface{}{
		"status": []interface{}{"value"},
	}
	if stateContainsAll(state, filter8) {
		t.Error("Expected state not to match when array expected but not present")
	}
}

func TestArrayContainsAll(t *testing.T) {
	stateArray := []interface{}{"a", "b", "c", "d"}
	
	// Test all elements present
	filterArray := []interface{}{"a", "c"}
	if !arrayContainsAll(stateArray, filterArray) {
		t.Error("Expected state array to contain all filter elements")
	}
	
	// Test missing element
	filterArray2 := []interface{}{"a", "e"}
	if arrayContainsAll(stateArray, filterArray2) {
		t.Error("Expected state array not to contain all filter elements")
	}
	
	// Test empty filter
	filterArray3 := []interface{}{}
	if !arrayContainsAll(stateArray, filterArray3) {
		t.Error("Expected state array to contain empty filter")
	}
	
	// Test exact match
	filterArray4 := []interface{}{"a", "b", "c", "d"}
	if !arrayContainsAll(stateArray, filterArray4) {
		t.Error("Expected state array to match exactly")
	}
}

func TestHandleTimeseriesAPI(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessions := []*client.Session{
			{
				ID: "session1",
				Metrics: []client.Metric{
					{Name: "cpu", Value: 50.0, Timestamp: time.Now()},
					{Name: "cpu", Value: 75.0, Timestamp: time.Now()},
					{Name: "memory", Value: 80.0, Timestamp: time.Now()},
				},
			},
			{
				ID: "session2",
				Metrics: []client.Metric{
					{Name: "cpu", Value: 60.0, Timestamp: time.Now()},
					{Name: "cpu", Value: 90.0, Timestamp: time.Now()},
				},
			},
		}
		json.NewEncoder(w).Encode(sessions)
	}))
	defer mockServer.Close()
	
	// Set up the global client
	datacatClient = client.NewClient(mockServer.URL)
	
	// Test missing metric parameter
	t.Run("MissingMetric", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
	
	// Test with metric parameter (all aggregation)
	t.Run("AllAggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries?metric=cpu&aggregation=all", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var result TimeseriesData
		json.NewDecoder(w.Body).Decode(&result)
		
		if result.MetricName != "cpu" {
			t.Errorf("Expected metric name cpu, got %s", result.MetricName)
		}
		if len(result.Points) != 4 {
			t.Errorf("Expected 4 points, got %d", len(result.Points))
		}
		if result.Peak != 90.0 {
			t.Errorf("Expected peak 90.0, got %f", result.Peak)
		}
		if result.Min != 50.0 {
			t.Errorf("Expected min 50.0, got %f", result.Min)
		}
	})
	
	// Test peak aggregation
	t.Run("PeakAggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries?metric=cpu&aggregation=peak", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var result TimeseriesData
		json.NewDecoder(w.Body).Decode(&result)
		
		if len(result.Points) != 2 {
			t.Errorf("Expected 2 points (one per session), got %d", len(result.Points))
		}
		if result.Peak != 90.0 {
			t.Errorf("Expected peak 90.0, got %f", result.Peak)
		}
	})
	
	// Test average aggregation
	t.Run("AverageAggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries?metric=cpu&aggregation=average", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var result TimeseriesData
		json.NewDecoder(w.Body).Decode(&result)
		
		if len(result.Points) != 2 {
			t.Errorf("Expected 2 points, got %d", len(result.Points))
		}
		if result.AggregationType != "average" {
			t.Errorf("Expected aggregation type average, got %s", result.AggregationType)
		}
	})
	
	// Test min aggregation
	t.Run("MinAggregation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries?metric=cpu&aggregation=min", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var result TimeseriesData
		json.NewDecoder(w.Body).Decode(&result)
		
		if len(result.Points) != 2 {
			t.Errorf("Expected 2 points, got %d", len(result.Points))
		}
	})
	
	// Test with non-existent metric
	t.Run("NonExistentMetric", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/timeseries?metric=nonexistent", nil)
		w := httptest.NewRecorder()
		
		handleTimeseriesAPI(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var result TimeseriesData
		json.NewDecoder(w.Body).Decode(&result)
		
		if len(result.Points) != 0 {
			t.Errorf("Expected 0 points for non-existent metric, got %d", len(result.Points))
		}
		if result.Peak != 0 {
			t.Errorf("Expected peak 0, got %f", result.Peak)
		}
	})
}

func TestHandleTimeseriesAPIError(t *testing.T) {
	// Create a mock server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer mockServer.Close()
	
	datacatClient = client.NewClient(mockServer.URL)
	
	req := httptest.NewRequest("GET", "/api/timeseries?metric=cpu", nil)
	w := httptest.NewRecorder()
	
	handleTimeseriesAPI(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
