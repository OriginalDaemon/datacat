package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
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

// ProductVersion represents a unique product and version combination
type ProductVersion struct {
	Product      string
	Version      string
	SessionCount int
}

// ProductsPageData represents data for the products listing page
type ProductsPageData struct {
	Title         string
	Products      []ProductInfo
	ServerOffline bool
	ErrorMessage  string
}

// ProductInfo represents aggregated info about a product
type ProductInfo struct {
	Name     string
	Versions []VersionInfo
}

// VersionInfo represents info about a specific version
type VersionInfo struct {
	Version      string
	SessionCount int
	ActiveCount  int
	EndedCount   int
	CrashedCount int
	HungCount    int
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
	http.HandleFunc("/api/server-status", handleServerStatus)

	// HTMX live update endpoints
	http.HandleFunc("/api/stats-cards", handleStatsCards)
	http.HandleFunc("/api/sessions-table", handleSessionsTable)
	http.HandleFunc("/api/session-info/", handleSessionInfo)
	http.HandleFunc("/api/sessions-metrics", handleSessionsMetrics)
	http.HandleFunc("/api/metric-data/", handleMetricData)
	http.HandleFunc("/api/products-grid", handleProductsGrid)
	http.HandleFunc("/api/session-timeline/", handleSessionTimeline)
	http.HandleFunc("/api/session-metrics-list/", handleSessionMetricsList)
	http.HandleFunc("/api/session-events/", handleSessionEvents)
	http.HandleFunc("/api/session-metrics-table/", handleSessionMetricsTable)

	port := ":8080"
	log.Printf("Starting datacat web UI on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "templates/base.html", "templates/products.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessions, err := datacatClient.GetAllSessions()
	data := ProductsPageData{
		Title: "Datacat Dashboard",
	}

	if err != nil {
		// Server is offline - render UI anyway but with warning
		data.ServerOffline = true
		data.ErrorMessage = "Cannot connect to datacat server. Please start the server."
		data.Products = []ProductInfo{}
	} else {
		// Extract unique products and versions
		data.Products = extractProductInfo(sessions)
	}

	err = tmpl.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

// extractProductInfo extracts unique products and versions from sessions
func extractProductInfo(sessions []*client.Session) []ProductInfo {
	productMap := make(map[string]map[string]*VersionInfo)

	for _, session := range sessions {
		// Extract product and version from state
		product, _ := session.State["product"].(string)
		version, _ := session.State["version"].(string)

		if product == "" {
			product = "Unknown"
		}
		if version == "" {
			version = "Unknown"
		}

		// Initialize product map if needed
		if productMap[product] == nil {
			productMap[product] = make(map[string]*VersionInfo)
		}

		// Initialize version info if needed
		if productMap[product][version] == nil {
			productMap[product][version] = &VersionInfo{
				Version: version,
			}
		}

		vi := productMap[product][version]
		vi.SessionCount++

		// Categorize session status
		if session.Active {
			// Check for hung/crashed status in state or via heartbeat
			status, _ := session.State["status"].(string)
			if status == "crashed" {
				vi.CrashedCount++
			} else if status == "hung" {
				vi.HungCount++
			} else {
				vi.ActiveCount++
			}
		} else {
			vi.EndedCount++
		}
	}

	// Convert to sorted list
	var products []ProductInfo
	for productName, versions := range productMap {
		var versionList []VersionInfo
		for _, vi := range versions {
			versionList = append(versionList, *vi)
		}
		// Sort versions
		sort.Slice(versionList, func(i, j int) bool {
			return versionList[i].Version < versionList[j].Version
		})
		products = append(products, ProductInfo{
			Name:     productName,
			Versions: versionList,
		})
	}

	// Sort products by name
	sort.Slice(products, func(i, j int) bool {
		return products[i].Name < products[j].Name
	})

	return products
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := 1
	perPage := 20
	sortBy := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	product := r.URL.Query().Get("product")
	version := r.URL.Query().Get("version")
	statusFilter := r.URL.Query().Get("status")
	stateKey := r.URL.Query().Get("state_key")
	stateValue := r.URL.Query().Get("state_value")
	eventName := r.URL.Query().Get("event_name")
	timeRange := r.URL.Query().Get("time_range")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if timeRange == "" {
		timeRange = "2w"
	}

	// Get page number
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Calculate time range boundaries
	var startTime, endTime time.Time
	now := time.Now()

	switch timeRange {
	case "1d":
		startTime = now.Add(-24 * time.Hour)
		endTime = now
	case "1w":
		startTime = now.Add(-7 * 24 * time.Hour)
		endTime = now
	case "2w":
		startTime = now.Add(-14 * 24 * time.Hour)
		endTime = now
	case "1m":
		startTime = now.AddDate(0, -1, 0)
		endTime = now
	case "3m":
		startTime = now.AddDate(0, -3, 0)
		endTime = now
	case "all":
		// No time filter
	case "custom":
		if startTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", startTimeStr); err == nil {
				startTime = t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", endTimeStr); err == nil {
				endTime = t
			}
		}
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filters
	var filteredSessions []*client.Session
	for _, session := range sessions {
		// Apply time range filter
		if timeRange != "all" && !startTime.IsZero() {
			if session.CreatedAt.Before(startTime) || session.CreatedAt.After(endTime) {
				continue
			}
		}

		if !matchesFilters(session, product, version, statusFilter, stateKey, stateValue, eventName) {
			continue
		}
		filteredSessions = append(filteredSessions, session)
	}

	// Sort sessions
	sortSessions(filteredSessions, sortBy, sortOrder)

	// Paginate
	totalSessions := len(filteredSessions)
	totalPages := (totalSessions + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalSessions {
		end = totalSessions
	}
	if start > totalSessions {
		start = totalSessions
	}

	paginatedSessions := []*client.Session{}
	if start < end {
		paginatedSessions = filteredSessions[start:end]
	}

	// Extract unique products and versions for filter dropdowns
	uniqueProducts := extractUniqueProducts(sessions)
	uniqueVersions := extractUniqueVersions(sessions, product)

	// Format time range for display
	var timeRangeDisplay string
	switch timeRange {
	case "1d":
		timeRangeDisplay = "Last 24 Hours"
	case "1w":
		timeRangeDisplay = "Last Week"
	case "2w":
		timeRangeDisplay = "Last 2 Weeks"
	case "1m":
		timeRangeDisplay = "Last Month"
	case "3m":
		timeRangeDisplay = "Last 3 Months"
	case "all":
		timeRangeDisplay = "All Time"
	case "custom":
		if !startTime.IsZero() && !endTime.IsZero() {
			timeRangeDisplay = fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
		}
	}

	// Build query string for HTMX requests
	queryParams := url.Values{}
	queryParams.Add("time_range", timeRange)
	if timeRange == "custom" {
		queryParams.Add("start_time", startTimeStr)
		queryParams.Add("end_time", endTimeStr)
	}
	if product != "" {
		queryParams.Add("product", product)
	}
	if version != "" {
		queryParams.Add("version", version)
	}
	if statusFilter != "" {
		queryParams.Add("status", statusFilter)
	}
	if stateKey != "" {
		queryParams.Add("state_key", stateKey)
	}
	if stateValue != "" {
		queryParams.Add("state_value", stateValue)
	}
	if eventName != "" {
		queryParams.Add("event_name", eventName)
	}

	// Prepare pagination data
	type SessionsData struct {
		Sessions         []*client.Session
		CurrentPage      int
		TotalPages       int
		TotalCount       int
		SortBy           string
		SortOrder        string
		Product          string
		Version          string
		StatusFilter     string
		StateKey         string
		StateValue       string
		EventName        string
		TimeRange        string
		StartTime        string
		EndTime          string
		TimeRangeDisplay string
		HasPrev          bool
		HasNext          bool
		Products         []string
		Versions         []string
		QueryString      string
	}

	data := SessionsData{
		Sessions:         paginatedSessions,
		CurrentPage:      page,
		TotalPages:       totalPages,
		TotalCount:       totalSessions,
		SortBy:           sortBy,
		SortOrder:        sortOrder,
		Product:          product,
		Version:          version,
		StatusFilter:     statusFilter,
		StateKey:         stateKey,
		StateValue:       stateValue,
		EventName:        eventName,
		TimeRange:        timeRange,
		StartTime:        startTimeStr,
		EndTime:          endTimeStr,
		TimeRangeDisplay: timeRangeDisplay,
		HasPrev:          page > 1,
		HasNext:          page < totalPages,
		Products:         uniqueProducts,
		Versions:         uniqueVersions,
		QueryString:      queryParams.Encode(),
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"eq":  func(a, b string) bool { return a == b },
	}
	t, err := template.New("sessions_filtered.html").Funcs(funcMap).ParseFS(content, "templates/sessions_filtered.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// matchesFilters checks if a session matches all specified filters
func matchesFilters(session *client.Session, product, version, statusFilter, stateKey, stateValue, eventName string) bool {
	// Product filter
	if product != "" {
		sessionProduct, _ := session.State["product"].(string)
		if sessionProduct != product {
			return false
		}
	}

	// Version filter
	if version != "" {
		sessionVersion, _ := session.State["version"].(string)
		if sessionVersion != version {
			return false
		}
	}

	// Status filter
	if statusFilter != "" {
		switch statusFilter {
		case "active":
			if !session.Active {
				return false
			}
		case "ended":
			if session.Active {
				return false
			}
		case "crashed":
			if !session.Crashed {
				return false
			}
		case "suspended":
			if !session.Suspended {
				return false
			}
		case "hung":
			// Sessions currently hung (Hung=true)
			if !session.Hung {
				return false
			}
		case "ever_hung":
			// Sessions that were hung at any point (check events)
			hasHangEvent := false
			for _, event := range session.Events {
				if event.Name == "application_appears_hung" {
					hasHangEvent = true
					break
				}
			}
			if !hasHangEvent {
				return false
			}
		case "hung_crashed":
			// Sessions that were hung when they crashed
			if !session.Crashed {
				return false
			}
			// Check if there was a hang event before crash
			hasHangEvent := false
			for _, event := range session.Events {
				if event.Name == "application_appears_hung" {
					hasHangEvent = true
					break
				}
			}
			if !hasHangEvent {
				return false
			}
		case "hung_ended":
			// Sessions that were hung when they ended normally
			if session.Active || session.Crashed || session.EndedAt == nil {
				return false
			}
			// Check if there was a hang event
			hasHangEvent := false
			for _, event := range session.Events {
				if event.Name == "application_appears_hung" {
					hasHangEvent = true
					break
				}
			}
			if !hasHangEvent {
				return false
			}
		case "hung_recovered":
			// Sessions that had a hang event and recovered (Hung=false after being true)
			if session.Hung {
				return false // Still hung, not recovered
			}
			// Check for both hang and recovery events
			hasHangEvent := false
			hasRecoveryEvent := false
			for _, event := range session.Events {
				if event.Name == "application_appears_hung" {
					hasHangEvent = true
				}
				if event.Name == "application_recovered" {
					hasRecoveryEvent = true
				}
			}
			if !hasHangEvent || !hasRecoveryEvent {
				return false
			}
		}
	}

	// State key/value filter
	if stateKey != "" && stateValue != "" {
		if !hasStateValue(session, stateKey, stateValue) {
			return false
		}
	}

	// Event name filter
	if eventName != "" {
		if !hasEvent(session, eventName) {
			return false
		}
	}

	return true
}

// hasStateValue checks if the session ever had a specific state value
func hasStateValue(session *client.Session, key, value string) bool {
	// Check current state
	if val, ok := session.State[key]; ok {
		if fmt.Sprintf("%v", val) == value {
			return true
		}
	}

	// Check state history
	for _, snapshot := range session.StateHistory {
		if val, ok := snapshot.State[key]; ok {
			if fmt.Sprintf("%v", val) == value {
				return true
			}
		}
	}

	return false
}

// hasEvent checks if the session has a specific event
func hasEvent(session *client.Session, eventName string) bool {
	for _, event := range session.Events {
		if event.Name == eventName {
			return true
		}
	}
	return false
}

// extractUniqueProducts extracts unique product names from sessions
func extractUniqueProducts(sessions []*client.Session) []string {
	productSet := make(map[string]bool)
	for _, session := range sessions {
		if product, ok := session.State["product"].(string); ok && product != "" {
			productSet[product] = true
		}
	}

	var products []string
	for product := range productSet {
		products = append(products, product)
	}
	sort.Strings(products)
	return products
}

// extractUniqueVersions extracts unique versions for a given product
func extractUniqueVersions(sessions []*client.Session, product string) []string {
	versionSet := make(map[string]bool)
	for _, session := range sessions {
		sessionProduct, _ := session.State["product"].(string)
		if product == "" || sessionProduct == product {
			if version, ok := session.State["version"].(string); ok && version != "" {
				versionSet[version] = true
			}
		}
	}

	var versions []string
	for version := range versionSet {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
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
		"toJSONSafe": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return template.JS(b)
		},
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

	// Build a session lookup map for timestamps
	sessionLookup := make(map[string]*client.Session)
	for _, session := range filteredSessions {
		sessionLookup[session.ID] = session
	}

	switch aggregation {
	case "peak":
		// One point per session with peak value
		for _, sessionMetrics := range sessionMetricsMap {
			session := sessionLookup[sessionMetrics.SessionID]
			points = append(points, TimeseriesPoint{
				Timestamp: session.CreatedAt,
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
			session := sessionLookup[sessionMetrics.SessionID]
			points = append(points, TimeseriesPoint{
				Timestamp: session.CreatedAt,
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
			session := sessionLookup[sessionMetrics.SessionID]
			points = append(points, TimeseriesPoint{
				Timestamp: session.CreatedAt,
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

// handleSessionsMetrics returns the list of unique metrics for filtered sessions
func handleSessionsMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Parse same filters as handleSessions
	product := r.URL.Query().Get("product")
	version := r.URL.Query().Get("version")
	statusFilter := r.URL.Query().Get("status")
	stateKey := r.URL.Query().Get("state_key")
	stateValue := r.URL.Query().Get("state_value")
	eventName := r.URL.Query().Get("event_name")
	timeRange := r.URL.Query().Get("time_range")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	// Calculate time range boundaries
	var startTime, endTime time.Time
	now := time.Now()

	if timeRange == "" {
		timeRange = "2w"
	}

	switch timeRange {
	case "1d":
		startTime = now.Add(-24 * time.Hour)
		endTime = now
	case "1w":
		startTime = now.Add(-7 * 24 * time.Hour)
		endTime = now
	case "2w":
		startTime = now.Add(-14 * 24 * time.Hour)
		endTime = now
	case "1m":
		startTime = now.AddDate(0, -1, 0)
		endTime = now
	case "3m":
		startTime = now.AddDate(0, -3, 0)
		endTime = now
	case "all":
		// No time filter
	case "custom":
		if startTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", startTimeStr); err == nil {
				startTime = t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", endTimeStr); err == nil {
				endTime = t
			}
		}
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		w.Write([]byte(`<p style="color: var(--error-color);">Error loading sessions</p>`))
		return
	}

	// Apply filters
	var filteredSessions []*client.Session
	for _, session := range sessions {
		// Apply time range filter
		if timeRange != "all" && !startTime.IsZero() {
			if session.CreatedAt.Before(startTime) || session.CreatedAt.After(endTime) {
				continue
			}
		}

		if !matchesFilters(session, product, version, statusFilter, stateKey, stateValue, eventName) {
			continue
		}
		filteredSessions = append(filteredSessions, session)
	}

	// Extract unique metric names
	metricNames := make(map[string]bool)
	for _, session := range filteredSessions {
		for _, metric := range session.Metrics {
			metricNames[metric.Name] = true
		}
	}

	// Sort metric names
	var sortedMetrics []string
	for name := range metricNames {
		sortedMetrics = append(sortedMetrics, name)
	}
	sort.Strings(sortedMetrics)

	// Build query string
	queryParams := url.Values{}
	queryParams.Add("time_range", timeRange)
	if timeRange == "custom" {
		queryParams.Add("start_time", startTimeStr)
		queryParams.Add("end_time", endTimeStr)
	}
	if product != "" {
		queryParams.Add("product", product)
	}
	if version != "" {
		queryParams.Add("version", version)
	}
	if statusFilter != "" {
		queryParams.Add("status", statusFilter)
	}
	if stateKey != "" {
		queryParams.Add("state_key", stateKey)
	}
	if stateValue != "" {
		queryParams.Add("state_value", stateValue)
	}
	if eventName != "" {
		queryParams.Add("event_name", eventName)
	}
	queryString := queryParams.Encode()

	if len(sortedMetrics) == 0 {
		w.Write([]byte(`<p style="text-align: center; padding: 20px; color: var(--text-secondary);">No metrics found in filtered sessions</p>`))
		return
	}

	// Build HTML for metric cards
	var html strings.Builder
	for _, metricName := range sortedMetrics {
		html.WriteString(fmt.Sprintf(`
		<div class="metric-card">
			<div class="collapsible-header" onclick="toggleMetric('%s')">
				<h4 style="margin: 0; color: var(--text-primary);">%s</h4>
				<span class="collapse-icon collapsed" id="metric-%s-icon">▼</span>
			</div>
			<div class="collapsible-content collapsed" id="metric-%s-content"
				 hx-get="/api/metric-data/%s?%s"
				 hx-trigger="loadMetric"
				 hx-swap="innerHTML">
				<p style="text-align: center; padding: 20px; color: var(--text-secondary);">Click to expand and load data...</p>
			</div>
		</div>
		`, metricName, metricName, metricName, metricName, url.PathEscape(metricName), queryString))
	}

	w.Write([]byte(html.String()))
}

// handleMetricData returns the timeseries data and chart for a specific metric
func handleMetricData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Extract metric name from URL path
	metricName := strings.TrimPrefix(r.URL.Path, "/api/metric-data/")
	metricName, _ = url.PathUnescape(metricName)

	// Parse same filters as handleSessions
	product := r.URL.Query().Get("product")
	version := r.URL.Query().Get("version")
	statusFilter := r.URL.Query().Get("status")
	stateKey := r.URL.Query().Get("state_key")
	stateValue := r.URL.Query().Get("state_value")
	eventName := r.URL.Query().Get("event_name")
	timeRange := r.URL.Query().Get("time_range")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	// Calculate time range boundaries
	var startTime, endTime time.Time
	now := time.Now()

	if timeRange == "" {
		timeRange = "2w"
	}

	switch timeRange {
	case "1d":
		startTime = now.Add(-24 * time.Hour)
		endTime = now
	case "1w":
		startTime = now.Add(-7 * 24 * time.Hour)
		endTime = now
	case "2w":
		startTime = now.Add(-14 * 24 * time.Hour)
		endTime = now
	case "1m":
		startTime = now.AddDate(0, -1, 0)
		endTime = now
	case "3m":
		startTime = now.AddDate(0, -3, 0)
		endTime = now
	case "all":
		// No time filter
	case "custom":
		if startTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", startTimeStr); err == nil {
				startTime = t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse("2006-01-02T15:04", endTimeStr); err == nil {
				endTime = t
			}
		}
	}

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		w.Write([]byte(`<p style="color: var(--error-color);">Error loading sessions</p>`))
		return
	}

	// Apply filters
	var filteredSessions []*client.Session
	for _, session := range sessions {
		// Apply time range filter
		if timeRange != "all" && !startTime.IsZero() {
			if session.CreatedAt.Before(startTime) || session.CreatedAt.After(endTime) {
				continue
			}
		}

		if !matchesFilters(session, product, version, statusFilter, stateKey, stateValue, eventName) {
			continue
		}
		filteredSessions = append(filteredSessions, session)
	}

	// Find the overall time range across all sessions
	var globalMinTime, globalMaxTime time.Time
	for _, session := range filteredSessions {
		if globalMinTime.IsZero() || session.CreatedAt.Before(globalMinTime) {
			globalMinTime = session.CreatedAt
		}
		if globalMaxTime.IsZero() || session.UpdatedAt.After(globalMaxTime) {
			globalMaxTime = session.UpdatedAt
		}
		// Also check metric timestamps
		for _, metric := range session.Metrics {
			if globalMinTime.IsZero() || metric.Timestamp.Before(globalMinTime) {
				globalMinTime = metric.Timestamp
			}
			if metric.Timestamp.After(globalMaxTime) {
				globalMaxTime = metric.Timestamp
			}
		}
	}

	// Collect all metric values
	var allValues []float64
	var points []TimeseriesPoint

	for _, session := range filteredSessions {
		for _, metric := range session.Metrics {
			if metric.Name == metricName {
				// Apply time range filter to metric timestamp
				if timeRange != "all" && !startTime.IsZero() {
					if metric.Timestamp.Before(startTime) || metric.Timestamp.After(endTime) {
						continue
					}
				}

				allValues = append(allValues, metric.Value)
				points = append(points, TimeseriesPoint{
					Timestamp: metric.Timestamp,
					Value:     metric.Value,
					SessionID: session.ID,
				})
			}
		}
	}

	// Sort points by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	if len(allValues) == 0 {
		w.Write([]byte(`<p style="text-align: center; padding: 20px; color: var(--text-secondary);">No data found for this metric</p>`))
		return
	}

	// Calculate statistics
	avg, max, min, median, stdDev, mode := calculateStats(allValues)

	// Generate unique chart ID
	chartID := fmt.Sprintf("chart-%s-%d", metricName, time.Now().UnixNano())

	// Build HTML with chart and stats
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`
		<div class="chart-container">
			<canvas id="%s"></canvas>
		</div>
		<div class="stats-row">
			<div class="stat-item">
				<div class="stat-label">Average</div>
				<div class="stat-value">%.2f</div>
			</div>
			<div class="stat-item">
				<div class="stat-label">Maximum</div>
				<div class="stat-value">%.2f</div>
			</div>
			<div class="stat-item">
				<div class="stat-label">Minimum</div>
				<div class="stat-value">%.2f</div>
			</div>
			<div class="stat-item">
				<div class="stat-label">Median</div>
				<div class="stat-value">%.2f</div>
			</div>
			<div class="stat-item">
				<div class="stat-label">Mode</div>
				<div class="stat-value">%.2f</div>
			</div>
			<div class="stat-item">
				<div class="stat-label">Std Dev</div>
				<div class="stat-value">%.2f</div>
			</div>
		</div>
		<script>
		(function() {
			const ctx = document.getElementById('%s').getContext('2d');
			const data = %s;

			// Calculate point radius based on data density
			const pointRadius = data.length > 100 ? 1 : (data.length > 50 ? 2 : 3);

			new Chart(ctx, {
				type: 'line',
				data: {
					datasets: [{
						label: '%s',
						data: data.map(p => ({
							x: new Date(p.timestamp),
							y: p.value
						})),
						borderColor: '#819BFC',
						backgroundColor: 'rgba(129, 155, 252, 0.1)',
						tension: 0.1,
						fill: true,
						pointRadius: pointRadius,
						pointHoverRadius: pointRadius + 2,
						pointBackgroundColor: '#819BFC',
						pointBorderColor: '#ffffff',
						pointBorderWidth: 1
					}]
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					interaction: {
						mode: 'nearest',
						axis: 'x',
						intersect: false
					},
					plugins: {
						legend: {
							display: false
						},
						tooltip: {
							callbacks: {
								title: function(context) {
									return new Date(context[0].parsed.x).toLocaleString();
								},
								label: function(context) {
									return 'Value: ' + context.parsed.y.toFixed(2);
								},
								afterLabel: function(context) {
									const point = data[context.dataIndex];
									return 'Session: ' + point.session_id.substring(0, 8) + '...';
								}
							}
						}
					},
					scales: {
						y: {
							beginAtZero: true,
							title: {
								display: true,
								text: 'Value'
							},
							grid: {
								color: 'rgba(160, 174, 192, 0.1)'
							},
							ticks: {
								color: '#a0aec0'
							}
						},
						x: {
							type: 'time',
							min: %d,
							max: %d,
							time: {
								displayFormats: {
									millisecond: 'HH:mm:ss.SSS',
									second: 'HH:mm:ss',
									minute: 'HH:mm',
									hour: 'HH:mm',
									day: 'MMM dd'
								},
								tooltipFormat: 'yyyy-MM-dd HH:mm:ss'
							},
							title: {
								display: true,
								text: 'Time'
							},
							grid: {
								color: 'rgba(160, 174, 192, 0.1)'
							},
							ticks: {
								color: '#a0aec0',
								maxRotation: 45,
								minRotation: 0,
								autoSkip: true,
								maxTicksLimit: 10
							}
						}
					}
				}
			});
		})();
		</script>
	`, chartID, avg, max, min, median, mode, stdDev, chartID, toJSON(points), metricName, globalMinTime.UnixMilli(), globalMaxTime.UnixMilli()))

	w.Write([]byte(html.String()))
}

// calculateStats calculates statistical measures for a slice of values
func calculateStats(values []float64) (avg, max, min, median, stdDev, mode float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	// Average and sum for std dev
	var sum float64
	max = values[0]
	min = values[0]

	for _, v := range values {
		sum += v
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}

	avg = sum / float64(len(values))

	// Standard deviation
	var variance float64
	for _, v := range values {
		diff := v - avg
		variance += diff * diff
	}
	variance /= float64(len(values))
	stdDev = math.Sqrt(variance)

	// Median and Mode
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	if len(sortedValues)%2 == 0 {
		median = (sortedValues[len(sortedValues)/2-1] + sortedValues[len(sortedValues)/2]) / 2
	} else {
		median = sortedValues[len(sortedValues)/2]
	}

	// Mode - most frequently occurring value
	frequencyMap := make(map[float64]int)
	maxFrequency := 0
	for _, v := range values {
		frequencyMap[v]++
		if frequencyMap[v] > maxFrequency {
			maxFrequency = frequencyMap[v]
			mode = v
		}
	}

	return
}

// toJSON converts data to JSON string for embedding in HTML
func toJSON(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "[]"
	}
	return string(b)
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

	// Determine status badge
	statusBadge := `<span class="badge badge-inactive">Ended</span>`
	if session.Active {
		statusBadge = `<span class="badge badge-active">Active</span>`
	} else if session.Crashed {
		statusBadge = `<span class="badge badge-crashed">Crashed</span>`
	} else if session.Suspended {
		statusBadge = `<span class="badge badge-suspended">Suspended</span>`
	}

	// Add hung badge if applicable
	if session.Hung {
		statusBadge += ` <span class="badge badge-hung" style="margin-left: 8px;">Hung</span>`
	}

	html.WriteString(`<tr>
		<th>Status</th>
		<td>` + statusBadge + `</td>
	</tr>`)

	// Add machine/hostname if available
	if session.Hostname != "" {
		html.WriteString(`<tr>
			<th>Machine</th>
			<td>` + session.Hostname + `</td>
		</tr>`)
	}

	// Add last heartbeat row
	if session.LastHeartbeat != nil {
		html.WriteString(`<tr>
			<th>Last Heartbeat</th>
			<td>` + session.LastHeartbeat.Format("2006-01-02 15:04:05") + `</td>
		</tr>`)
	} else {
		html.WriteString(`<tr>
			<th>Last Heartbeat</th>
			<td style="color: var(--text-secondary);">No heartbeats received</td>
		</tr>`)
	}

	html.WriteString(`<tr>
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

// handleProductsGrid returns the products grid with live session counts
func handleProductsGrid(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sessions, err := datacatClient.GetAllSessions()
	if err != nil {
		// Return error state with retry
		w.Write([]byte(`<div id="products-grid-container" hx-get="/api/products-grid" hx-trigger="every 10s" hx-swap="outerHTML">
<p style="color: var(--error-color); padding: 20px; text-align: center;">
Error loading products. Retrying...
</p>
</div>`))
		return
	}

	products := extractProductInfo(sessions)

	// Check if there are any active sessions to adjust polling
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

	// Build HTML
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="products-grid-container" hx-get="/api/products-grid" hx-trigger="every %s" hx-swap="outerHTML">`, pollInterval))

	if len(products) == 0 {
		html.WriteString(`<p style="color: var(--text-secondary); padding: 20px; text-align: center;">
No sessions found. Start logging data to see products here.
</p>`)
	} else {
		html.WriteString(`<div class="products-grid">`)
		for _, product := range products {
			html.WriteString(fmt.Sprintf(`
<div class="product-card">
<h3 class="product-name">%s</h3>
<div class="versions-list">`, product.Name))

			for _, version := range product.Versions {
				productParam := url.QueryEscape(product.Name)
				versionParam := url.QueryEscape(version.Version)
				baseURL := fmt.Sprintf("/sessions?product=%s&version=%s", productParam, versionParam)

				html.WriteString(fmt.Sprintf(`
<div class="version-item" onclick="window.location.href='%s'" style="cursor: pointer;">
<a href="%s" class="version-link" onclick="event.stopPropagation();">
<span class="version-number">v%s</span>
<span class="session-count">%d sessions</span>
</a>
<div class="version-stats" onclick="event.stopPropagation();">`,
					baseURL,
					baseURL,
					version.Version,
					version.SessionCount))

				if version.ActiveCount > 0 {
					activeURL := fmt.Sprintf("%s&status=active", baseURL)
					html.WriteString(fmt.Sprintf(`<a href="%s" class="stat-badge stat-active" style="text-decoration: none;">%d active</a>`, activeURL, version.ActiveCount))
				}
				if version.EndedCount > 0 {
					endedURL := fmt.Sprintf("%s&status=ended", baseURL)
					html.WriteString(fmt.Sprintf(`<a href="%s" class="stat-badge stat-ended" style="text-decoration: none;">%d ended</a>`, endedURL, version.EndedCount))
				}
				if version.CrashedCount > 0 {
					crashedURL := fmt.Sprintf("%s&status=crashed", baseURL)
					html.WriteString(fmt.Sprintf(`<a href="%s" class="stat-badge stat-crashed" style="text-decoration: none;">%d crashed</a>`, crashedURL, version.CrashedCount))
				}
				if version.HungCount > 0 {
					hungURL := fmt.Sprintf("%s&status=hung", baseURL)
					html.WriteString(fmt.Sprintf(`<a href="%s" class="stat-badge stat-hung" style="text-decoration: none;">%d hung</a>`, hungURL, version.HungCount))
				}

				html.WriteString(`</div>
</div>`)
			}

			html.WriteString(fmt.Sprintf(`
<div class="version-item">
<a href="/sessions?product=%s" class="version-link version-all">
<span class="version-number">All Versions</span>
<span class="session-count">View all →</span>
</a>
</div>
</div>
</div>`, url.QueryEscape(product.Name)))
		}
		html.WriteString(`</div>`)

		html.WriteString(`
<div style="margin-top: 30px; text-align: center;">
<a href="/sessions" class="btn">View All Sessions</a>
</div>`)
	}

	html.WriteString(`</div>`)
	w.Write([]byte(html.String()))
}

// handleSessionTimeline returns the timeline HTML for a session with live updates
func handleSessionTimeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session-timeline/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		w.Write([]byte(`<div id="timeline-data">
<p style="color: var(--error-color);">Error loading timeline</p>
</div>`))
		return
	}

	// Determine poll interval based on session status
	pollInterval := ""
	if session.Active {
		pollInterval = ` hx-get="/api/session-timeline/` + sessionID + `" hx-trigger="every 5s" hx-swap="outerHTML"`
	}

	// Build timeline items JSON data
	var timelineItems []map[string]interface{}

	// Add state changes
	for i, snapshot := range session.StateHistory {
		timelineItems = append(timelineItems, map[string]interface{}{
			"type":        "state",
			"timestamp":   snapshot.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
			"displayTime": snapshot.Timestamp.Format("15:04:05.000"),
			"index":       i,
		})
	}

	// Add events
	for _, event := range session.Events {
		eventType := "event"
		if event.Name == "exception" {
			eventType = "exception"
		}
		timelineItems = append(timelineItems, map[string]interface{}{
			"type":        eventType,
			"timestamp":   event.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
			"displayTime": event.Timestamp.Format("15:04:05.000"),
			"name":        event.Name,
		})
	}

	timelineJSON, _ := json.Marshal(timelineItems)

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="timeline-data"%s>`, pollInterval))
	html.WriteString(fmt.Sprintf(`<script>
// Update timeline with new data
if (typeof updateTimelineData === 'function') {
updateTimelineData(%s);
}
</script>`, string(timelineJSON)))
	html.WriteString(`</div>`)

	w.Write([]byte(html.String()))
}

// handleSessionMetricsList returns the metrics list with live updates
func handleSessionMetricsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session-metrics-list/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		w.Write([]byte(`<div id="metrics-data">
<p style="color: var(--error-color);">Error loading metrics</p>
</div>`))
		return
	}

	// Determine poll interval based on session status
	pollInterval := ""
	if session.Active {
		pollInterval = ` hx-get="/api/session-metrics-list/` + sessionID + `" hx-trigger="every 5s" hx-swap="outerHTML"`
	}

	// Build metrics JSON data
	metricsJSON, _ := json.Marshal(session.Metrics)

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="metrics-data"%s>`, pollInterval))
	html.WriteString(fmt.Sprintf(`<script>
// Update metrics with new data
if (typeof updateMetricsData === 'function') {
updateMetricsData(%s);
}
</script>`, string(metricsJSON)))
	html.WriteString(`</div>`)

	w.Write([]byte(html.String()))
}

// handleSessionEvents returns the events table with live updates
func handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session-events/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		w.Write([]byte(`<div id="events-container">
<p style="color: var(--error-color);">Error loading events</p>
</div>`))
		return
	}

	// Determine poll interval based on session status
	pollInterval := ""
	if session.Active {
		pollInterval = ` hx-get="/api/session-events/` + sessionID + `" hx-trigger="every 10s" hx-swap="outerHTML"`
	}

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="events-container"%s>`, pollInterval))

	if len(session.Events) == 0 {
		html.WriteString(`<p style="color: var(--text-secondary); padding: 20px;">No events recorded</p>`)
	} else {
		html.WriteString(`<table>
<thead>
<tr>
<th style="width: 120px">Timestamp</th>
<th style="width: 200px">Name</th>
<th>Data</th>
</tr>
</thead>
<tbody>`)

		for i, event := range session.Events {
			eventDataJSON, _ := json.Marshal(event.Data)
			html.WriteString(fmt.Sprintf(`
<tr>
<td style="font-family: monospace">%s</td>
<td>%s</td>
<td>
<div id="event-data-%d" class="json-viewer-container"></div>
<script>
(function() {
const eventData = %s;
new JSONViewer('event-data-%d', eventData, {
collapsed: true,
showCopyButton: true,
showRawButton: true
});
})();
</script>
</td>
</tr>`,
				event.Timestamp.Format("15:04:05"),
				event.Name,
				i,
				string(eventDataJSON),
				i))
		}

		html.WriteString(`</tbody>
</table>`)
	}

	html.WriteString(`</div>`)
	w.Write([]byte(html.String()))
}

// handleSessionMetricsTable returns the metrics data table with live updates
func handleSessionMetricsTable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session-metrics-table/")

	session, err := datacatClient.GetSession(sessionID)
	if err != nil {
		w.Write([]byte(`<div id="metrics-table-container">
<p style="color: var(--error-color);">Error loading metrics</p>
</div>`))
		return
	}

	// Determine poll interval based on session status
	pollInterval := ""
	if session.Active {
		pollInterval = ` hx-get="/api/session-metrics-table/` + sessionID + `" hx-trigger="every 10s" hx-swap="outerHTML"`
	}

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div id="metrics-table-container"%s>`, pollInterval))

	if len(session.Metrics) == 0 {
		html.WriteString(`<p style="color: var(--text-secondary); padding: 20px;">No metrics recorded</p>`)
	} else {
		html.WriteString(`<table>
<thead>
<tr>
<th>Timestamp</th>
<th>Name</th>
<th>Value</th>
<th>Tags</th>
</tr>
</thead>
<tbody>`)

		for _, metric := range session.Metrics {
			tags := ""
			for _, tag := range metric.Tags {
				tags += tag + " "
			}
			html.WriteString(fmt.Sprintf(`
<tr>
<td style="font-family: monospace">%s</td>
<td>%s</td>
<td>%f</td>
<td>%s</td>
</tr>`,
				metric.Timestamp.Format("15:04:05"),
				metric.Name,
				metric.Value,
				tags))
		}

		html.WriteString(`</tbody>
</table>`)
	}

	html.WriteString(`</div>`)
	w.Write([]byte(html.String()))
}
