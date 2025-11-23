package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// initLogging initializes file logging based on config
// Returns the log file path and a cleanup function
// Note: This should only be called if config.LogFile is set
func initLogging(config *Config) (string, func(), error) {
	var logPath string
	var logFile *os.File
	var err error

	// Determine log file path
	if config.LogFile != "" {
		logPath = config.LogFile
	} else {
		// Generate unique log file name with timestamp and PID in current directory
		timestamp := time.Now().Format("20060102-150405")
		pid := os.Getpid()
		logPath = fmt.Sprintf("datacat-web-%s-%d.log", timestamp, pid)
	}

	// Create log file
	logFile, err = os.Create(logPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create log file %s: %v", logPath, err)
	}

	// Set up multi-writer to write to both stderr and file
	// Keep stderr so we still see output in console
	multiWriter := io.MultiWriter(os.Stderr, logFile)
	log.SetOutput(multiWriter)

	// Cleanup function to close log file
	cleanup := func() {
		if logFile != nil {
			logFile.Close()
		}
	}

	return logPath, cleanup, nil
}
