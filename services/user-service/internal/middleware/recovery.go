package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
	"user-service/internal/models"
)

// RecoveryMiddleware creates a panic recovery middleware
func RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					log.Error().
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Interface("panic", err).
						Bytes("stack", debug.Stack()).
						Msg("Panic recovered")

					// Return 500 error
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					response := models.NewErrorResponse(
						"Internal server error",
						fmt.Sprintf("panic: %v", err),
					)

					// Simple JSON encoding
					w.Write([]byte(`{"success":false,"error":"Internal server error","timestamp":"` + response.Timestamp.Format("2006-01-02T15:04:05Z07:00") + `"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
