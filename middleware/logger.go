package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/AchrafSoltani/quark"
)

// LoggerConfig defines the configuration for Logger middleware.
type LoggerConfig struct {
	// Output is the writer where logs are written.
	Output io.Writer

	// Format is the log format template.
	// Available fields: ${time}, ${method}, ${path}, ${status}, ${latency}, ${ip}, ${user_agent}
	Format string

	// TimeFormat is the time format (time.Layout).
	TimeFormat string

	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string

	// CustomTimeFormat allows custom time formatting.
	CustomTimeFormat func(time.Time) string
}

// DefaultLoggerConfig is the default logger configuration.
var DefaultLoggerConfig = LoggerConfig{
	Output:     os.Stdout,
	Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}",
	TimeFormat: "2006/01/02 15:04:05",
	SkipPaths:  []string{},
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	quark.Context
	status int
}

// Logger returns a Logger middleware with default configuration.
func Logger() quark.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with the given configuration.
func LoggerWithConfig(config LoggerConfig) quark.MiddlewareFunc {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.Format == "" {
		config.Format = DefaultLoggerConfig.Format
	}
	if config.TimeFormat == "" {
		config.TimeFormat = DefaultLoggerConfig.TimeFormat
	}

	// Build skip paths map
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			// Check if path should be skipped
			if skipPaths[c.Path()] {
				return next(c)
			}

			start := time.Now()

			// Create a status capturing writer
			sw := &statusWriter{
				ResponseWriter: c.Writer,
				status:         200,
			}
			c.Writer = sw

			// Process request
			err := next(c)

			// Calculate latency
			latency := time.Since(start)

			// Get status code
			status := sw.status

			// If there was an error, try to get status from HTTPError
			if err != nil {
				if httpErr, ok := err.(*quark.HTTPError); ok {
					status = httpErr.Code
				} else {
					status = 500
				}
			}

			// Format time
			var timeStr string
			if config.CustomTimeFormat != nil {
				timeStr = config.CustomTimeFormat(start)
			} else {
				timeStr = start.Format(config.TimeFormat)
			}

			// Format latency
			latencyStr := formatLatency(latency)

			// Build log line
			log := config.Format
			log = replaceTag(log, "${time}", timeStr)
			log = replaceTag(log, "${method}", c.Method())
			log = replaceTag(log, "${path}", c.Path())
			log = replaceTag(log, "${status}", fmt.Sprintf("%d", status))
			log = replaceTag(log, "${latency}", latencyStr)
			log = replaceTag(log, "${ip}", c.RealIP())
			log = replaceTag(log, "${user_agent}", c.Header("User-Agent"))

			// Add status color codes for terminal output
			log = colorizeStatus(log, status)

			fmt.Fprintln(config.Output, log)

			return err
		}
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// formatLatency formats the latency duration.
func formatLatency(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// replaceTag replaces a tag in the format string.
func replaceTag(format, tag, value string) string {
	for i := 0; i < len(format)-len(tag)+1; i++ {
		if format[i:i+len(tag)] == tag {
			return format[:i] + value + format[i+len(tag):]
		}
	}
	return format
}

// ANSI color codes
const (
	reset  = "\033[0m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
)

// colorizeStatus adds color to the log line based on status code.
func colorizeStatus(log string, status int) string {
	var color string
	switch {
	case status >= 500:
		color = red
	case status >= 400:
		color = yellow
	case status >= 300:
		color = cyan
	case status >= 200:
		color = green
	default:
		color = blue
	}

	statusStr := fmt.Sprintf("%d", status)
	coloredStatus := color + statusStr + reset

	// Replace the status in the log with the colored version
	for i := 0; i < len(log)-len(statusStr)+1; i++ {
		if log[i:i+len(statusStr)] == statusStr {
			return log[:i] + coloredStatus + log[i+len(statusStr):]
		}
	}
	return log
}

// LoggerWithSkipPaths returns a logger that skips certain paths.
func LoggerWithSkipPaths(paths ...string) quark.MiddlewareFunc {
	config := DefaultLoggerConfig
	config.SkipPaths = paths
	return LoggerWithConfig(config)
}

// LoggerWithOutput returns a logger with custom output.
func LoggerWithOutput(w io.Writer) quark.MiddlewareFunc {
	config := DefaultLoggerConfig
	config.Output = w
	return LoggerWithConfig(config)
}
