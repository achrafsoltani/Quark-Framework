package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/AchrafSoltani/quark"
)

// RecoveryConfig defines the configuration for Recovery middleware.
type RecoveryConfig struct {
	// StackSize is the maximum size of the stack trace to capture.
	StackSize int

	// DisableStackAll disables capturing stack traces from all goroutines.
	DisableStackAll bool

	// DisablePrintStack disables printing stack traces.
	DisablePrintStack bool

	// Output is the writer where panic info is written.
	Output io.Writer

	// Handler is a custom handler called when a panic occurs.
	// If nil, a default JSON error response is sent.
	Handler func(*quark.Context, interface{}, []byte) error
}

// DefaultRecoveryConfig is the default recovery configuration.
var DefaultRecoveryConfig = RecoveryConfig{
	StackSize:         4 << 10, // 4 KB
	DisableStackAll:   false,
	DisablePrintStack: false,
	Output:            os.Stderr,
	Handler:           nil,
}

// Recovery returns a Recovery middleware with default configuration.
// It recovers from panics, logs the panic and stack trace, and returns a 500 error.
func Recovery() quark.MiddlewareFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig)
}

// RecoveryWithConfig returns a Recovery middleware with the given configuration.
func RecoveryWithConfig(config RecoveryConfig) quark.MiddlewareFunc {
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoveryConfig.StackSize
	}
	if config.Output == nil {
		config.Output = DefaultRecoveryConfig.Output
	}

	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// Capture stack trace
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, !config.DisableStackAll)
					stack = stack[:length]

					// Print stack trace if not disabled
					if !config.DisablePrintStack {
						fmt.Fprintf(config.Output, "[PANIC RECOVER] %v\n%s\n", r, stack)
					}

					// Call custom handler if set
					if config.Handler != nil {
						if err := config.Handler(c, r, stack); err != nil {
							// Handler failed, send default error response
							sendDefaultPanicResponse(c, r)
						}
						return
					}

					// Send default error response
					sendDefaultPanicResponse(c, r)
				}
			}()

			return next(c)
		}
	}
}

// sendDefaultPanicResponse sends a default 500 error response.
func sendDefaultPanicResponse(c *quark.Context, recovered interface{}) {
	if c.IsWritten() {
		return
	}

	c.JSON(http.StatusInternalServerError, quark.M{
		"error": quark.M{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		},
	})
}

// RecoveryWithHandler returns a Recovery middleware with a custom handler.
func RecoveryWithHandler(handler func(*quark.Context, interface{}, []byte) error) quark.MiddlewareFunc {
	config := DefaultRecoveryConfig
	config.Handler = handler
	return RecoveryWithConfig(config)
}

// RecoveryWithOutput returns a Recovery middleware with custom output.
func RecoveryWithOutput(w io.Writer) quark.MiddlewareFunc {
	config := DefaultRecoveryConfig
	config.Output = w
	return RecoveryWithConfig(config)
}

// DebugRecovery returns a Recovery middleware that includes panic details in the response.
// ONLY use this in development mode.
func DebugRecovery() quark.MiddlewareFunc {
	return RecoveryWithHandler(func(c *quark.Context, recovered interface{}, stack []byte) error {
		return c.JSON(http.StatusInternalServerError, quark.M{
			"error": quark.M{
				"code":    http.StatusInternalServerError,
				"message": "Internal Server Error",
				"panic":   fmt.Sprintf("%v", recovered),
				"stack":   string(stack),
			},
		})
	})
}
