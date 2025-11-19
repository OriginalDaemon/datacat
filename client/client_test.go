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

func TestNewDaemonManager(t *testing.T) {
	// Test with specified binary
	dm := NewDaemonManager("8081", "http://localhost:8080", "/path/to/daemon")
	if dm == nil {
		t.Fatal("NewDaemonManager returned nil")
	}
	if dm.daemonPort != "8081" {
		t.Errorf("Expected port 8081, got %s", dm.daemonPort)
	}
	if dm.serverURL != "http://localhost:8080" {
		t.Errorf("Expected serverURL http://localhost:8080, got %s", dm.serverURL)
	}
	if dm.daemonBinary != "/path/to/daemon" {
		t.Errorf("Expected binary /path/to/daemon, got %s", dm.daemonBinary)
	}
	
	// Test with empty binary (should call findDaemonBinary)
	dm2 := NewDaemonManager("8081", "http://localhost:8080", "")
	if dm2 == nil {
		t.Fatal("NewDaemonManager returned nil")
	}
	if dm2.daemonBinary == "" {
		t.Error("daemonBinary should not be empty after findDaemonBinary call")
	}
}

func TestDaemonManagerIsRunning(t *testing.T) {
	dm := NewDaemonManager("8081", "http://localhost:8080", "test-binary")
	
	// Initially not running
	if dm.IsRunning() {
		t.Error("Daemon should not be running initially")
	}
	
	// Simulate started state
	dm.started = true
	if dm.IsRunning() {
		t.Error("Daemon should not be running without process")
	}
}

func TestClose(t *testing.T) {
	// Test Close without daemon
	client := NewClient("http://localhost:8080")
	err := client.Close()
	if err != nil {
		t.Errorf("Close should not fail without daemon: %v", err)
	}
	
	// Test Close with daemon manager (but not started)
	client2 := &Client{
		BaseURL:       "http://localhost:8081",
		HTTPClient:    &http.Client{},
		UseDaemon:     true,
		DaemonManager: NewDaemonManager("8081", "http://localhost:8080", "test-binary"),
	}
	err = client2.Close()
	if err != nil {
		t.Errorf("Close should not fail with stopped daemon: %v", err)
	}
}

func TestHeartbeat(t *testing.T) {
	// Test heartbeat with non-daemon client (should be no-op)
	client := NewClient("http://localhost:8080")
	err := client.Heartbeat("test-session")
	if err != nil {
		t.Errorf("Heartbeat should not fail for non-daemon client: %v", err)
	}
	
	// Test heartbeat with daemon client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/heartbeat" {
			t.Errorf("Expected path /heartbeat, got %s", r.URL.Path)
		}
		
		var data map[string]interface{}
		json.NewDecoder(r.Body).Decode(&data)
		
		if data["session_id"] != "test-session" {
			t.Errorf("Expected session_id test-session, got %v", data["session_id"])
		}
		
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	
	daemonClient := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	err = daemonClient.Heartbeat("test-session")
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
}

func TestCreateSessionWithDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" {
			var data map[string]interface{}
			json.NewDecoder(r.Body).Decode(&data)
			
			// Verify parent_pid is sent
			if _, ok := data["parent_pid"]; !ok {
				t.Error("Expected parent_pid in request")
			}
			
			response := map[string]string{"session_id": "daemon-session-id"}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()
	
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	sessionID, err := client.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession with daemon failed: %v", err)
	}
	if sessionID != "daemon-session-id" {
		t.Errorf("Expected session_id daemon-session-id, got %s", sessionID)
	}
}

func TestUpdateStateWithDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/state" {
			var data map[string]interface{}
			json.NewDecoder(r.Body).Decode(&data)
			
			if data["session_id"] != "test-id" {
				t.Errorf("Expected session_id test-id, got %v", data["session_id"])
			}
			
			state, ok := data["state"].(map[string]interface{})
			if !ok {
				t.Error("Expected state to be a map")
			}
			if state["key"] != "value" {
				t.Errorf("Expected state.key to be value, got %v", state["key"])
			}
			
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()
	
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	err := client.UpdateState("test-id", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("UpdateState with daemon failed: %v", err)
	}
}

func TestLogEventWithDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/event" {
			var data map[string]interface{}
			json.NewDecoder(r.Body).Decode(&data)
			
			if data["session_id"] != "test-id" {
				t.Errorf("Expected session_id test-id, got %v", data["session_id"])
			}
			if data["name"] != "test_event" {
				t.Errorf("Expected name test_event, got %v", data["name"])
			}
			
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()
	
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	err := client.LogEvent("test-id", "test_event", map[string]interface{}{"data": "test"})
	if err != nil {
		t.Fatalf("LogEvent with daemon failed: %v", err)
	}
}

func TestLogMetricWithDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metric" {
			var data map[string]interface{}
			json.NewDecoder(r.Body).Decode(&data)
			
			if data["session_id"] != "test-id" {
				t.Errorf("Expected session_id test-id, got %v", data["session_id"])
			}
			if data["name"] != "test_metric" {
				t.Errorf("Expected name test_metric, got %v", data["name"])
			}
			if data["value"] != 42.5 {
				t.Errorf("Expected value 42.5, got %v", data["value"])
			}
			
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()
	
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	err := client.LogMetric("test-id", "test_metric", 42.5, []string{"tag1"})
	if err != nil {
		t.Fatalf("LogMetric with daemon failed: %v", err)
	}
}

func TestEndSessionWithDaemon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/end" {
			var data map[string]interface{}
			json.NewDecoder(r.Body).Decode(&data)
			
			if data["session_id"] != "test-id" {
				t.Errorf("Expected session_id test-id, got %v", data["session_id"])
			}
			
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()
	
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UseDaemon:  true,
	}
	
	err := client.EndSession("test-id")
	if err != nil {
		t.Fatalf("EndSession with daemon failed: %v", err)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test CreateSession error
	client := NewClient("http://invalid-url-that-does-not-exist:99999")
	_, err := client.CreateSession()
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test GetSession error
	_, err = client.GetSession("test-id")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test UpdateState error
	err = client.UpdateState("test-id", map[string]interface{}{"key": "value"})
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test LogEvent error
	err = client.LogEvent("test-id", "event", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test LogMetric error
	err = client.LogMetric("test-id", "metric", 1.0, nil)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test EndSession error
	err = client.EndSession("test-id")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test GetAllSessions error
	_, err = client.GetAllSessions()
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test Heartbeat error
	daemonClient := &Client{
		BaseURL:    "http://invalid-url-that-does-not-exist:99999",
		HTTPClient: &http.Client{Timeout: 1 * time.Second},
		UseDaemon:  true,
	}
	err = daemonClient.Heartbeat("test-id")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestHTTPErrorResponses(t *testing.T) {
	// Test CreateSession with error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	
	_, err := client.CreateSession()
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	_, err = client.GetSession("test-id")
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	err = client.UpdateState("test-id", map[string]interface{}{"key": "value"})
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	err = client.LogEvent("test-id", "event", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	err = client.LogMetric("test-id", "metric", 1.0, nil)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	err = client.EndSession("test-id")
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	
	_, err = client.GetAllSessions()
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestInvalidJSONResponses(t *testing.T) {
	// Test CreateSession with invalid JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	
	_, err := client.CreateSession()
	if err == nil {
		t.Error("Expected error for invalid JSON response in CreateSession")
	}
	
	_, err = client.GetSession("test-id")
	if err == nil {
		t.Error("Expected error for invalid JSON response in GetSession")
	}
	
	_, err = client.GetAllSessions()
	if err == nil {
		t.Error("Expected error for invalid JSON response in GetAllSessions")
	}
}

func TestDaemonManagerStop(t *testing.T) {
	dm := NewDaemonManager("8081", "http://localhost:8080", "test-binary")
	
	// Stopping a non-started daemon should not error
	err := dm.Stop()
	if err != nil {
		t.Errorf("Stop should not fail for non-started daemon: %v", err)
	}
}
