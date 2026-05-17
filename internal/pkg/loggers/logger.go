// Package loggers provides structured logging utilities used across the application,
package loggers

import "log"

// New returns the default logger used across the service.
func New() *log.Logger {
	return log.Default()
}
