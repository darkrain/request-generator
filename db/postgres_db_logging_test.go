package db

import "testing"

func TestPostgresLoggingAllowsMissingRequestLogger(t *testing.T) {
	database := &DB{Debug: true}

	database.debugLog(nil, "debug message")
	errorLog(nil, "error message")
}
