// Package middleware provides HTTP middleware integration for bifrost-extensions.
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// ApplyDefaultStack applies the middleware stack to a chi router.
// This includes CORS and basic request handling.
//
// Parameters:
//   - router: The chi router to apply middleware to
//
// Returns:
//   - error: An error if middleware setup fails
func ApplyDefaultStack(router *chi.Mux) error {
	// Apply CORS middleware
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:           300,
	}))
	return nil
}

// CORSOptions configures CORS with bifrost-extensions-specific settings.
type CORSOptions struct {
	AllowedOrigins []string
	AllowedHosts   []string
}

// ApplyCustomCORS applies custom CORS middleware with bifrost-extensions-specific configuration.
//
// Parameters:
//   - router: The chi router to apply middleware to
//   - options: CORS configuration options
func ApplyCustomCORS(router *chi.Mux, options CORSOptions) {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   options.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:           300,
	}))
}

// HealthCheckRoute registers a health check endpoint.
//
// Parameters:
//   - router: The chi router to register the route on
func HealthCheckRoute(router *chi.Mux) {
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
}

// ReadinessCheckRoute registers a readiness check endpoint.
//
// Parameters:
//   - router: The chi router to register the route on
func ReadinessCheckRoute(router *chi.Mux) {
	router.Get("/readiness", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ready":true}`))
	})
}

// RequestIDHandler is a helper that allows callers to use request timeouts.
type RequestIDHandler struct {
	timeout time.Duration
}

// NewRequestIDHandler creates a new RequestIDHandler.
//
// Parameters:
//   - timeout: The timeout for handling requests
//
// Returns:
//   - *RequestIDHandler: A new RequestIDHandler instance
func NewRequestIDHandler(timeout time.Duration) *RequestIDHandler {
	return &RequestIDHandler{
		timeout: timeout,
	}
}

// WrapHandler wraps a handler with timeout middleware.
//
// Parameters:
//   - h: The handler to wrap
//
// Returns:
//   - http.Handler: The wrapped handler
func (h *RequestIDHandler) WrapHandler(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, h.timeout, "Request timeout")
}
