package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

// Checker reports readiness of a dependency.
type Checker func(ctx context.Context) error

// Handler serves /healthz and /readyz.
type Handler struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

func New() *Handler {
	return &Handler{checkers: make(map[string]Checker)}
}

func (h *Handler) Register(name string, c Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = c
}

func (h *Handler) Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func (h *Handler) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		defer h.mu.RUnlock()
		status := http.StatusOK
		result := map[string]string{}
		for name, check := range h.checkers {
			if err := check(r.Context()); err != nil {
				status = http.StatusServiceUnavailable
				result[name] = err.Error()
			} else {
				result[name] = "ok"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": map[bool]string{true: "ready", false: "not_ready"}[status == http.StatusOK],
			"checks": result,
		})
	}
}
