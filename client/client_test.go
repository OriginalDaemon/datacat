package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected BaseURL to be http://localhost:8080, got %s", client.BaseURL)
	}
	if client.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}
}

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/sessions" {
			t.Errorf("Expected path /api/sessions, got %s", r.URL.Path)
		}
		
		response := map[string]string{"session_id": "test-session-id"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessionID, err := client.CreateSession()
	
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if sessionID != "test-session-id" {
		t.Errorf("Expected session_id to be test-session-id, got %s", sessionID)
	}
}

func TestGetSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		session := &Session{
			ID:        "test-id",
			Active:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			State:     map[string]interface{}{"key": "value"},
			Events:    []Event{},
			Metrics:   []Metric{},
		}
		json.NewEncoder(w).Encode(session)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	session, err := client.GetSession("test-id")
	
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.ID != "test-id" {
		t.Errorf("Expected ID to be test-id, got %s", session.ID)
	}
	if !session.Active {
		t.Error("Expected session to be active")
	}
}

func TestUpdateState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type to be application/json")
		}
		
		var state map[string]interface{}
		json.NewDecoder(r.Body).Decode(&state)
		
		if state["key"] != "value" {
			t.Errorf("Expected state.key to be value, got %v", state["key"])
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.UpdateState("test-id", map[string]interface{}{"key": "value"})
	
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}
}

func TestLogEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		var eventData map[string]interface{}
		json.NewDecoder(r.Body).Decode(&eventData)
		
		if eventData["name"] != "test_event" {
			t.Errorf("Expected event name to be test_event, got %v", eventData["name"])
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.LogEvent("test-id", "test_event", map[string]interface{}{"data": "test"})
	
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}
}

func TestLogMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		var metricData map[string]interface{}
		json.NewDecoder(r.Body).Decode(&metricData)
		
		if metricData["name"] != "test_metric" {
			t.Errorf("Expected metric name to be test_metric, got %v", metricData["name"])
		}
		if metricData["value"] != 42.5 {
			t.Errorf("Expected metric value to be 42.5, got %v", metricData["value"])
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.LogMetric("test-id", "test_metric", 42.5, []string{"tag1"})
	
	if err != nil {
		t.Fatalf("LogMetric failed: %v", err)
	}
}

func TestEndSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.EndSession("test-id")
	
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}
}

func TestGetAllSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		sessions := []*Session{
			{ID: "session1", Active: true},
			{ID: "session2", Active: false},
		}
		json.NewEncoder(w).Encode(sessions)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessions, err := client.GetAllSessions()
	
	if err != nil {
		t.Fatalf("GetAllSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != "session1" {
		t.Errorf("Expected first session ID to be session1, got %s", sessions[0].ID)
	}
}
