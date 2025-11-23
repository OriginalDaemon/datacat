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
		logPath = fmt.Sprintf("datacat-server-%s-%d.log", timestamp, pid)
	}

	// Create log file
	logFile, err = os.Create(logPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create log file: %v", err)
	}

	// Set up multi-writer to write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Cleanup function to close log file
	cleanup := func() {
		if logFile != nil {
			logFile.Close()
		}
	}

	return logPath, cleanup, nil
}
