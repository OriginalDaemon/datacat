// Package client provides a Go client for the datacat REST API
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client is a datacat API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Session represents a datacat session
type Session struct {
	ID           string                 `json:"id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	Active       bool                   `json:"active"`
	State        map[string]interface{} `json:"state"`
	StateHistory []StateSnapshot        `json:"state_history"`
	Events       []Event                `json:"events"`
	Metrics      []Metric               `json:"metrics"`
}

// StateSnapshot represents the state at a specific point in time
type StateSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	State     map[string]interface{} `json:"state"`
}

// Event represents an event in a session
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data"`
}

// Metric represents a metric in a session
type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Tags      []string  `json:"tags,omitempty"`
}

// NewClient creates a new datacat client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession creates a new session
func (c *Client) CreateSession() (string, error) {
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/sessions", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("create session failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result["session_id"], nil
}

// GetSession retrieves a session by ID
func (c *Client) GetSession(sessionID string) (*Session, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/sessions/" + sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("get session failed with status %d: %s", resp.StatusCode, string(body))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// UpdateState updates session state
func (c *Client) UpdateState(sessionID string, state map[string]interface{}) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/sessions/"+sessionID+"/state", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("update state failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LogEvent logs an event
func (c *Client) LogEvent(sessionID, name string, data map[string]interface{}) error {
	eventData := map[string]interface{}{
		"name": name,
		"data": data,
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/sessions/"+sessionID+"/events", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("log event failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LogMetric logs a metric
func (c *Client) LogMetric(sessionID, name string, value float64, tags []string) error {
	metricData := map[string]interface{}{
		"name":  name,
		"value": value,
		"tags":  tags,
	}

	jsonData, err := json.Marshal(metricData)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/sessions/"+sessionID+"/metrics", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to log metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("log metric failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EndSession ends a session
func (c *Client) EndSession(sessionID string) error {
	req, err := http.NewRequest("POST", c.BaseURL+"/api/sessions/"+sessionID+"/end", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("end session failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAllSessions retrieves all sessions
func (c *Client) GetAllSessions() ([]*Session, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/grafana/sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("get sessions failed with status %d: %s", resp.StatusCode, string(body))
	}

	var sessions []*Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}
