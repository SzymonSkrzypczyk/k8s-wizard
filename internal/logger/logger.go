package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	logFile *os.File
)

// Init initializes the logger to write to a temporary file.
func Init() (string, error) {
	tempDir := os.TempDir()
	path := filepath.Join(tempDir, "k8s-wizard.log")
	
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	
	logFile = f
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	log.Println("--- Logger initialized ---")
	return path, nil
}

// Close closes the log file.
func Close() {
	if logFile != nil {
		log.Println("--- Logger closing ---")
		logFile.Close()
	}
}

// Info logs an informational message.
func Info(format string, v ...interface{}) {
	log.SetPrefix("INFO: ")
	log.Printf(format, v...)
}

// Error logs an error message.
func Error(format string, v ...interface{}) {
	log.SetPrefix("ERROR: ")
	log.Printf(format, v...)
}

// Debug logs a debug message.
func Debug(format string, v ...interface{}) {
	log.SetPrefix("DEBUG: ")
	log.Printf(format, v...)
}
