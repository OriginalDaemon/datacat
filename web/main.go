package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/OriginalDaemon/datacat/client"
)

//go:embed templates/* static/*
var content embed.FS

var datacatClient *client.Client

type PageData struct {
	Title    string
	Sessions []*client.Session
	Session  *client.Session
}

type TimeseriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	SessionID string    `json:"session_id"`
}

type TimeseriesData struct {
	MetricName       string            `json:"metric_name"`
	Points           []TimeseriesPoint `json:"points"`
	Peak             float64           `json:"peak"`
	Average          float64           `json:"average"`
	Min              float64           `json:"min"`
	SessionsMatched  int               `json:"sessions_matched"`
	AggregationType  string            `json:"aggregation_type"`
}

type SessionMetrics struct {
	SessionID string
	Peak      float64
	Average   float64
	Min       float64
	Values    []float64
}

func main() {
	// Initialize datacat client
	datacatClient = client.NewClient("http://localhost:8080")

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/sessions", handleSessions)
	http.HandleFunc("/session/", handleSessionDetail)
	http.HandleFunc("/api/timeseries", handleTimeseriesAPI)
	http.HandleFunc("/metrics", handleMetrics)

	port := ":8081"
	log.Printf("Starting datacat web UI on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "templates/index.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort sessions by created_at descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	data := PageData{
		Title:    "Datacat Dashboard",
		Sessions: sessions,
	}

	tmpl.Execute(w, data)
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort sessions by created_at descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	tmpl, err := template.ParseFS(content, "templates/sessions.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, sessions)
}

func handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/session/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFS(content, "templates/session.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:   "Session Detail",
		Session: session,
	}

	tmpl.Execute(w, data)
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "templates/metrics.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Metrics Visualization",
	}

	tmpl.Execute(w, data)
}

func handleTimeseriesAPI(w http.ResponseWriter, r *http.Request) {
	metricName := r.URL.Query().Get("metric")
	aggregation := r.URL.Query().Get("aggregation")
	filterMode := r.URL.Query().Get("filter_mode")
	filterPath := r.URL.Query().Get("filter_path")
	filterValue := r.URL.Query().Get("filter_value")

	if metricName == "" {
		http.Error(w, "metric parameter required", http.StatusBadRequest)
		return
	}

	if aggregation == "" {
		aggregation = "all"
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter sessions based on filter mode
	var filteredSessions []*client.Session
	for _, session := range sessions {
		if shouldIncludeSession(session, filterMode, filterPath, filterValue) {
			filteredSessions = append(filteredSessions, session)
		}
	}

	// Collect metrics based on aggregation mode
	var points []TimeseriesPoint
	var sessionMetricsMap = make(map[string]*SessionMetrics)
	
	for _, session := range filteredSessions {
		sessionMetrics := &SessionMetrics{
			SessionID: session.ID,
			Min:       -1,
		}

		for _, metric := range session.Metrics {
			if metric.Name == metricName {
				sessionMetrics.Values = append(sessionMetrics.Values, metric.Value)

				// Update peak
				if metric.Value > sessionMetrics.Peak {
					sessionMetrics.Peak = metric.Value
				}

				// Update min
				if sessionMetrics.Min < 0 || metric.Value < sessionMetrics.Min {
					sessionMetrics.Min = metric.Value
				}
			}
		}

		if len(sessionMetrics.Values) > 0 {
			// Calculate average
			var sum float64
			for _, v := range sessionMetrics.Values {
				sum += v
			}
			sessionMetrics.Average = sum / float64(len(sessionMetrics.Values))
			sessionMetricsMap[session.ID] = sessionMetrics
		}
	}

	// Generate points based on aggregation
	var overallPeak, overallMin float64 = 0, -1
	var overallSum float64
	var overallCount int

	switch aggregation {
	case "peak":
		// One point per session with peak value
		for _, sessionMetrics := range sessionMetricsMap {
			points = append(points, TimeseriesPoint{
				Timestamp: time.Now(), // Could use session created time
				Value:     sessionMetrics.Peak,
				SessionID: sessionMetrics.SessionID,
			})
			
			if sessionMetrics.Peak > overallPeak {
				overallPeak = sessionMetrics.Peak
			}
			if overallMin < 0 || sessionMetrics.Peak < overallMin {
				overallMin = sessionMetrics.Peak
			}
			overallSum += sessionMetrics.Peak
			overallCount++
		}

	case "average":
		// One point per session with average value
		for _, sessionMetrics := range sessionMetricsMap {
			points = append(points, TimeseriesPoint{
				Timestamp: time.Now(),
				Value:     sessionMetrics.Average,
				SessionID: sessionMetrics.SessionID,
			})
			
			if sessionMetrics.Average > overallPeak {
				overallPeak = sessionMetrics.Average
			}
			if overallMin < 0 || sessionMetrics.Average < overallMin {
				overallMin = sessionMetrics.Average
			}
			overallSum += sessionMetrics.Average
			overallCount++
		}

	case "min":
		// One point per session with min value
		for _, sessionMetrics := range sessionMetricsMap {
			points = append(points, TimeseriesPoint{
				Timestamp: time.Now(),
				Value:     sessionMetrics.Min,
				SessionID: sessionMetrics.SessionID,
			})
			
			if sessionMetrics.Min > overallPeak {
				overallPeak = sessionMetrics.Min
			}
			if overallMin < 0 || sessionMetrics.Min < overallMin {
				overallMin = sessionMetrics.Min
			}
			overallSum += sessionMetrics.Min
			overallCount++
		}

	default: // "all"
		// All metric values from all sessions
		for _, session := range filteredSessions {
			for _, metric := range session.Metrics {
				if metric.Name == metricName {
					points = append(points, TimeseriesPoint{
						Timestamp: metric.Timestamp,
						Value:     metric.Value,
						SessionID: session.ID,
					})

					if metric.Value > overallPeak {
						overallPeak = metric.Value
					}
					if overallMin < 0 || metric.Value < overallMin {
						overallMin = metric.Value
					}
					overallSum += metric.Value
					overallCount++
				}
			}
		}
	}

	// Sort points by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	var overallAverage float64
	if overallCount > 0 {
		overallAverage = overallSum / float64(overallCount)
	}

	if overallMin < 0 {
		overallMin = 0
	}

	result := TimeseriesData{
		MetricName:      metricName,
		Points:          points,
		Peak:            overallPeak,
		Average:         overallAverage,
		Min:             overallMin,
		SessionsMatched: len(filteredSessions),
		AggregationType: aggregation,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func shouldIncludeSession(session *client.Session, filterMode, filterPath, filterValue string) bool {
	if filterMode == "none" || filterMode == "" {
		return true
	}

	switch filterMode {
	case "current_state":
		// Check current state
		return matchesStateFilter(session.State, filterPath, filterValue)

	case "state_history":
		// Check if state ever had this value (would need event tracking)
		// For now, just check current state
		return matchesStateFilter(session.State, filterPath, filterValue)

	case "state_array_contains":
		// Check if array in state contains value
		return stateArrayContains(session.State, filterPath, filterValue)
	}

	return true
}

func stateArrayContains(state map[string]interface{}, path, value string) bool {
	// Navigate to the array using dot notation
	parts := strings.Split(path, ".")
	var current interface{} = state

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return false
		}
	}

	// Check if current is an array/slice
	if arr, ok := current.([]interface{}); ok {
		for _, item := range arr {
			if fmt.Sprintf("%v", item) == value {
				return true
			}
		}
	}

	return false
}

func matchesStateFilter(state map[string]interface{}, filter, value string) bool {
	// Support nested state filtering with dot notation (e.g., "application.status")
	parts := strings.Split(filter, ".")

	var current interface{} = state
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return false
		}
	}

	// Convert current to string and compare
	currentStr := fmt.Sprintf("%v", current)
	return currentStr == value
}
