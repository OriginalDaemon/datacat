package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/OriginalDaemon/datacat/client"
)

func main() {
	// Create client
	c := client.NewClient("http://localhost:9090")

	// Create session
	sessionID, err := c.CreateSession()
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	log.Printf("Created session: %s", sessionID)

	// Set initial state
	err = c.UpdateState(sessionID, map[string]interface{}{
		"application": map[string]interface{}{
			"name":    "go-example",
			"version": "1.0.0",
			"status":  "starting",
		},
		"metrics": map[string]interface{}{
			"cpu":    0.0,
			"memory": 0.0,
		},
	})
	if err != nil {
		log.Fatalf("Failed to update state: %v", err)
	}

	// Log startup event
	err = c.LogEvent(sessionID, "application_started", map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"host":      "localhost",
	})
	if err != nil {
		log.Fatalf("Failed to log event: %v", err)
	}

	// Update to running status
	err = c.UpdateState(sessionID, map[string]interface{}{
		"application": map[string]interface{}{
			"status": "running",
		},
	})
	if err != nil {
		log.Fatalf("Failed to update state: %v", err)
	}

	// Simulate work with metrics
	rand.Seed(time.Now().UnixNano())
	numIterations := rand.Intn(6) + 5 // 5-10 iterations for variety
	for i := 0; i < numIterations; i++ {
		// Generate random metrics
		cpu := 20.0 + rand.Float64()*60.0
		memory := 100.0 + rand.Float64()*400.0

		// Log metrics
		err = c.LogMetric(sessionID, "cpu_usage", cpu, []string{"app:go-example", "unit:percent"})
		if err != nil {
			log.Printf("Warning: Failed to log CPU metric: %v", err)
		}

		err = c.LogMetric(sessionID, "memory_usage", memory, []string{"app:go-example", "unit:mb"})
		if err != nil {
			log.Printf("Warning: Failed to log memory metric: %v", err)
		}

		// Update state with current metrics
		err = c.UpdateState(sessionID, map[string]interface{}{
			"metrics": map[string]interface{}{
				"cpu":    cpu,
				"memory": memory,
			},
		})
		if err != nil {
			log.Printf("Warning: Failed to update state: %v", err)
		}

		// Simulate occasional errors (30% chance)
		if rand.Float64() < 0.3 {
			errorTypes := []string{"network_timeout", "disk_full", "connection_lost"}
			errorType := errorTypes[rand.Intn(len(errorTypes))]
			errorMsg := map[string]string{
				"network_timeout": "Network request timed out after 30s",
				"disk_full":       "Insufficient disk space for operation",
				"connection_lost": "Database connection lost unexpectedly",
			}

			log.Printf("Iteration %d: Simulated error - %s", i+1, errorType)

			// Log error as event (similar to Python's log_exception)
			err = c.LogEvent(sessionID, "exception", map[string]interface{}{
				"type":      errorType,
				"message":   errorMsg[errorType],
				"iteration": i + 1,
				"severity":  "warning",
			})
			if err != nil {
				log.Printf("Warning: Failed to log error: %v", err)
			}
		}

		log.Printf("Iteration %d/%d: CPU=%.2f%%, Memory=%.2fMB", i+1, numIterations, cpu, memory)
		time.Sleep(2 * time.Second)
	}

	// Log completion event
	err = c.LogEvent(sessionID, "work_completed", map[string]interface{}{
		"iterations": numIterations,
		"duration":   fmt.Sprintf("%ds", numIterations*2),
	})
	if err != nil {
		log.Printf("Warning: Failed to log event: %v", err)
	}

	// End session
	err = c.EndSession(sessionID)
	if err != nil {
		log.Fatalf("Failed to end session: %v", err)
	}

	log.Printf("Session ended successfully")

	// Retrieve session to verify
	session, err := c.GetSession(sessionID)
	if err != nil {
		log.Fatalf("Failed to retrieve session: %v", err)
	}

	// Count errors
	errorCount := 0
	for _, event := range session.Events {
		if event.Name == "exception" {
			errorCount++
		}
	}

	log.Printf("Final session state:")
	log.Printf("  Active: %v", session.Active)
	log.Printf("  Events: %d", len(session.Events))
	log.Printf("  Metrics: %d", len(session.Metrics))
	log.Printf("  Errors: %d", errorCount)
}
