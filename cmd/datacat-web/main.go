package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OriginalDaemon/datacat/client"
)

//go:embed templates/* static/*
var content embed.FS

var datacatClient *client.Client

// PageData represents the data passed to HTML templates
type PageData struct {
	Title         string
	Sessions      []*client.Session
	Session       *client.Session
	ServerOffline bool
	ErrorMessage  string
}

// TimeseriesPoint represents a single point in a time series
type TimeseriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	SessionID string    `json:"session_id"`
}

// TimeseriesData represents aggregated time series data for a metric
type TimeseriesData struct {
	MetricName      string            `json:"metric_name"`
	Points          []TimeseriesPoint `json:"points"`
	Peak            float64           `json:"peak"`
	Average         float64           `json:"average"`
	Min             float64           `json:"min"`
	SessionsMatched int               `json:"sessions_matched"`
	AggregationType string            `json:"aggregation_type"`
}

// SessionMetrics represents metric statistics for a single session
type SessionMetrics struct {
	SessionID string
	Peak      float64
	Average   float64
	Min       float64
	Values    []float64
}

func main() {
	// Initialize datacat client
	datacatClient = client.NewClient("http://localhost:9090")

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/sessions", handleSessions)
	http.HandleFunc("/session/", handleSessionDetail)
	http.HandleFunc("/api/timeseries", handleTimeseriesAPI)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/api/server-status", handleServerStatus)
	
	// HTMX live update endpoints
	http.HandleFunc("/api/stats-cards", handleStatsCards)
	http.HandleFunc("/api/sessions-table", handleSessionsTable)
	http.HandleFunc("/api/session-info/", handleSessionInfo)

	port := ":8080"
	log.Printf("Starting datacat web UI on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}
	tmpl, err := template.New("base.html").Funcs(funcMap).ParseFS(content, "templates/base.html", "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessions, err := datacatClient.GetAllSessions()
	data := PageData{
		Title: "Datacat Dashboard",
	}

	if err != nil {
		// Server is offline - render UI anyway but with warning
		data.ServerOffline = true
		data.ErrorMessage = "Cannot connect to datacat server. Please start the server."
		data.Sessions = []*client.Session{}
	} else {
		// Sort sessions by created_at descending
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
		})
		data.Sessions = sessions
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := 1
	perPage := 20
	sortBy := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	stateFilter := r.URL.Query().Get("state_filter")

	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Get page number
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply state filter if provided
	if stateFilter != "" {
		sessions = filterSessionsByState(sessions, stateFilter)
	}

	// Sort sessions
	sortSessions(sessions, sortBy, sortOrder)

	// Paginate
	totalSessions := len(sessions)
	totalPages := (totalSessions + perPage - 1) / perPage
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalSessions {
		end = totalSessions
	}
	if start > totalSessions {
		start = totalSessions
	}

	paginatedSessions := sessions[start:end]

	// Prepare pagination data
	type SessionsData struct {
		Sessions    []*client.Session
		CurrentPage int
		TotalPages  int
		TotalCount  int
		SortBy      string
		SortOrder   string
		StateFilter string
		HasPrev     bool
		HasNext     bool
	}

	data := SessionsData{
		Sessions:    paginatedSessions,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalSessions,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
		StateFilter: stateFilter,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"eq":  func(a, b string) bool { return a == b },
	}
	t, err := template.New("sessions_enhanced.html").Funcs(funcMap).ParseFS(content, "templates/sessions_enhanced.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/session/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	funcMap := template.FuncMap{
		"replace": strings.ReplaceAll,
	}
	tmpl, err := template.New("base.html").Funcs(funcMap).ParseFS(content, "templates/base.html", "templates/session.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:   "Session Detail",
		Session: session,
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "templates/base.html", "templates/metrics.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Metrics Visualization",
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
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

func sortSessions(sessions []*client.Session, sortBy, sortOrder string) {
	sort.Slice(sessions, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "created_at":
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		case "updated_at":
			less = sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt)
		case "status":
			less = sessions[i].Active && !sessions[j].Active
		case "events":
			less = len(sessions[i].Events) < len(sessions[j].Events)
		case "metrics":
			less = len(sessions[i].Metrics) < len(sessions[j].Metrics)
		default:
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func filterSessionsByState(sessions []*client.Session, filterJSON string) []*client.Session {
	// Parse the JSON filter
	var filterState map[string]interface{}
	if err := json.Unmarshal([]byte(filterJSON), &filterState); err != nil {
		// If JSON is invalid, return all sessions
		return sessions
	}

	var filtered []*client.Session
	for _, session := range sessions {
		if matchesStateHistory(session, filterState) {
			filtered = append(filtered, session)
		}
	}
	return filtered
}

func matchesStateHistory(session *client.Session, filterState map[string]interface{}) bool {
	// Check if the session's current state or any historical state matches the filter
	// For simplicity, we'll check if the current state contains all keys/values from the filter
	return stateContainsAll(session.State, filterState)
}

func stateContainsAll(state, filter map[string]interface{}) bool {
	for key, filterValue := range filter {
		stateValue, exists := state[key]
		if !exists {
			return false
		}

		// If filter value is a map, recurse
		if filterMap, ok := filterValue.(map[string]interface{}); ok {
			if stateMap, ok := stateValue.(map[string]interface{}); ok {
				if !stateContainsAll(stateMap, filterMap) {
					return false
				}
			} else {
				return false
			}
		} else if filterArray, ok := filterValue.([]interface{}); ok {
			// If filter value is an array, check if state array contains all elements
			if stateArray, ok := stateValue.([]interface{}); ok {
				if !arrayContainsAll(stateArray, filterArray) {
					return false
				}
			} else {
				return false
			}
		} else {
			// Direct comparison
			if fmt.Sprintf("%v", stateValue) != fmt.Sprintf("%v", filterValue) {
				return false
			}
		}
	}
	return true
}

func arrayContainsAll(stateArray, filterArray []interface{}) bool {
	for _, filterItem := range filterArray {
		found := false
		for _, stateItem := range stateArray {
			if fmt.Sprintf("%v", stateItem) == fmt.Sprintf("%v", filterItem) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// handleServerStatus checks if the server is online and returns status
func handleServerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	// Try to get sessions to check if server is online
	_, err := datacatClient.GetAllSessions()
	
	if err != nil {
		// Server is offline
		w.Write([]byte(`<div id="server-status" class="alert alert-danger" hx-get="/api/server-status" hx-trigger="every 5s" hx-swap="outerHTML">
			<strong>⚠️ Server Offline:</strong> Cannot connect to datacat server at http://localhost:9090. Please start the server.
		</div>`))
	} else {
		// Server is online
		w.Write([]byte(`<div id="server-status" class="alert alert-success" hx-get="/api/server-status" hx-trigger="every 10s" hx-swap="outerHTML">
			<strong>✓ Server Online:</strong> Connected to datacat server at http://localhost:9090
		</div>`))
	}
}

// handleStatsCards returns the stats cards with live session counts
func handleStatsCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		// Return error state
		w.Write([]byte(`<div class="stats-grid" id="stats-cards" hx-get="/api/stats-cards" hx-trigger="every 5s" hx-swap="outerHTML">
			<div class="stat-card">
				<h3>Error</h3>
				<div class="value">N/A</div>
			</div>
		</div>`))
		return
	}
	
	// Calculate stats
	totalSessions := len(sessions)
	activeSessions := 0
	totalEvents := 0
	totalMetrics := 0
	
	for _, session := range sessions {
		if session.Active {
			activeSessions++
		}
		totalEvents += len(session.Events)
		totalMetrics += len(session.Metrics)
	}
	
	// Determine poll interval based on active sessions
	pollInterval := "10s"
	if activeSessions > 0 {
		pollInterval = "5s"
	}
	
	html := fmt.Sprintf(`<div class="stats-grid" id="stats-cards" hx-get="/api/stats-cards" hx-trigger="every %s" hx-swap="outerHTML">
		<div class="stat-card">
			<h3>Total Sessions</h3>
			<div class="value">%d</div>
		</div>
		<div class="stat-card">
			<h3>Active Sessions</h3>
			<div class="value">%d</div>
		</div>
		<div class="stat-card">
			<h3>Total Events</h3>
			<div class="value">%d</div>
		</div>
		<div class="stat-card">
			<h3>Total Metrics</h3>
			<div class="value">%d</div>
		</div>
	</div>`, pollInterval, totalSessions, activeSessions, totalEvents, totalMetrics)
	
	w.Write([]byte(html))
}

// handleSessionsTable returns the sessions table for HTMX updates
func handleSessionsTable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		w.Write([]byte(`<div id="sessions-table" hx-get="/api/sessions-table" hx-trigger="every 10s" hx-swap="outerHTML">
			<p style="color: var(--error-color);">Error loading sessions</p>
		</div>`))
		return
	}
	
	// Sort sessions by created_at descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})
	
	// Check if there are active sessions to determine poll interval
	hasActiveSessions := false
	for _, session := range sessions {
		if session.Active {
			hasActiveSessions = true
			break
		}
	}
	
	pollInterval := "30s"
	if hasActiveSessions {
		pollInterval = "10s"
	}
	
	// Build the table HTML
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="sessions-table" hx-get="/api/sessions-table" hx-trigger="every %s" hx-swap="outerHTML">`, pollInterval))
	html.WriteString(`<table>
		<thead>
			<tr>
				<th>Session ID</th>
				<th>Created</th>
				<th>Updated</th>
				<th>Status</th>
				<th>Events</th>
				<th>Metrics</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>`)
	
	if len(sessions) == 0 {
		html.WriteString(`<tr>
			<td colspan="7" style="text-align: center; padding: 40px; color: var(--text-secondary);">
				No sessions found
			</td>
		</tr>`)
	} else {
		for _, session := range sessions {
			statusBadge := `<span class="badge badge-inactive">Ended</span>`
			if session.Active {
				statusBadge = `<span class="badge badge-active">Active</span>`
			}
			
			sessionIDShort := session.ID
			if len(sessionIDShort) > 12 {
				sessionIDShort = sessionIDShort[:12] + "..."
			}
			
			html.WriteString(fmt.Sprintf(`<tr>
				<td>
					<a href="/session/%s" style="color: var(--accent-primary); text-decoration: none;">
						%s
					</a>
				</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%d</td>
				<td>%d</td>
				<td>
					<a href="/session/%s" class="btn btn-secondary" style="padding: 6px 12px; font-size: 13px;">View</a>
				</td>
			</tr>`,
				session.ID,
				sessionIDShort,
				session.CreatedAt.Format("2006-01-02 15:04:05"),
				session.UpdatedAt.Format("2006-01-02 15:04:05"),
				statusBadge,
				len(session.Events),
				len(session.Metrics),
				session.ID))
		}
	}
	
	html.WriteString(`</tbody>
	</table>
	</div>`)
	
	w.Write([]byte(html.String()))
}

// handleSessionInfo returns updated session info for active sessions
func handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session-info/")
	
	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		w.Write([]byte(`<div id="session-info">
			<p style="color: var(--error-color);">Error loading session</p>
		</div>`))
		return
	}
	
	// Determine poll interval based on session status
	pollInterval := ""
	if session.Active {
		pollInterval = fmt.Sprintf(` hx-get="/api/session-info/%s" hx-trigger="every 10s" hx-swap="outerHTML"`, sessionID)
	}
	
	// Build session info HTML
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="session-info"%s>`, pollInterval))
	html.WriteString(`<table style="margin-bottom: 20px;">
		<tr>
			<th style="width: 200px;">Created</th>
			<td>` + session.CreatedAt.Format("2006-01-02 15:04:05") + `</td>
		</tr>
		<tr>
			<th>Updated</th>
			<td>` + session.UpdatedAt.Format("2006-01-02 15:04:05") + `</td>
		</tr>`)
	
	if session.EndedAt != nil {
		html.WriteString(`<tr>
			<th>Ended</th>
			<td>` + session.EndedAt.Format("2006-01-02 15:04:05") + `</td>
		</tr>`)
	}
	
	statusBadge := `<span class="badge badge-inactive">Ended</span>`
	if session.Active {
		statusBadge = `<span class="badge badge-active">Active</span>`
	}
	
	html.WriteString(`<tr>
		<th>Status</th>
		<td>` + statusBadge + `</td>
	</tr>
	<tr>
		<th>State Changes</th>
		<td>` + fmt.Sprintf("%d", len(session.StateHistory)) + ` updates recorded</td>
	</tr>
	<tr>
		<th>Events</th>
		<td>` + fmt.Sprintf("%d", len(session.Events)) + ` events logged</td>
	</tr>
	<tr>
		<th>Metrics</th>
		<td>` + fmt.Sprintf("%d", len(session.Metrics)) + ` metrics recorded</td>
	</tr>
	</table>
	</div>`)
	
	w.Write([]byte(html.String()))
}

