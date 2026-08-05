package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/redteam/orchestrator"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ═══════════════════════════════════════════════════════════
// DrogonClaw Local API
// Single-user API with static token authentication
// ═══════════════════════════════════════════════════════════

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var globalOrchestrator *orchestrator.Orchestrator

func init() {
	globalOrchestrator = orchestrator.New()
	sb, err := sandbox.New()
	if err != nil {
		log.Printf("[API] Sandbox unavailable, engagement execution will require manual sandbox setup: %v", err)
		return
	}
	globalOrchestrator.SetSandbox(sb)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			sendError(w, http.StatusUnauthorized, "Missing or invalid token")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		expectedToken := strings.TrimSpace(os.Getenv("DROGONCLAW_API_KEY"))
		if expectedToken == "" {
			sendError(w, http.StatusServiceUnavailable, "API authentication is not configured")
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			sendError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	}
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: data})
}

func sendError(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err})
}

// --- Handlers ---

func handleHealth(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, map[string]string{"status": "healthy", "version": "2.0.0"})
}

func handleGetScans(w http.ResponseWriter, r *http.Request) {
	scans := []map[string]interface{}{}
	for id, eng := range globalOrchestrator.ActiveEngagements {
		scans = append(scans, map[string]interface{}{
			"id":        id,
			"target":    eng.Target,
			"status":    eng.Status,
			"timestamp": eng.StartTime,
		})
	}
	sendJSON(w, http.StatusOK, scans)
}

func handleStartScan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req struct {
		Target string `json:"target"`
		Type   string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Target == "" || len(req.Target) > 2048 {
		sendError(w, http.StatusBadRequest, "Target must be non-empty and <= 2048 characters")
		return
	}

	eng := globalOrchestrator.StartEngagement(req.Target)
	go globalOrchestrator.Execute(context.Background(), eng.ID)

	sendJSON(w, http.StatusAccepted, map[string]string{
		"message": fmt.Sprintf("Started %s scan on %s", req.Type, req.Target),
		"scanId":  eng.ID,
	})
}

func StartServer(port string) error {
	return StartServerAt("127.0.0.1", port)
}

// StartServerAt starts the HTTP API on an explicit address. StartServer binds
// to loopback by default so an unauthenticated deployment is never exposed by
// accident; use StartTLSServerAt for a remote listener.
func StartServerAt(host, port string) error {
	mux := setupRoutes()
	server := &http.Server{
		Addr:         host + ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting Local API (HTTP) on %s", server.Addr)
	return server.ListenAndServe()
}

func StartTLSServer(port, certFile, keyFile string) error {
	return StartTLSServerAt("127.0.0.1", port, certFile, keyFile)
}

// StartTLSServerAt starts the TLS API on an explicit address.
func StartTLSServerAt(host, port, certFile, keyFile string) error {
	mux := setupRoutes()
	server := &http.Server{
		Addr:         host + ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting Local API (HTTPS) on %s", server.Addr)
	return server.ListenAndServeTLS(certFile, keyFile)
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	securityHeaders := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			next.ServeHTTP(w, r)
		})
	}

	mux.Handle("/health", securityHeaders(http.HandlerFunc(handleHealth)))
	mux.Handle("/api/v1/scans", securityHeaders(http.HandlerFunc(AuthMiddleware(handleGetScans))))
	mux.Handle("/api/v1/scans/start", securityHeaders(http.HandlerFunc(AuthMiddleware(handleStartScan))))
	mux.Handle("/api/v1/engagements/start", securityHeaders(http.HandlerFunc(AuthMiddleware(handleStartEngagement))))
	mux.Handle("/api/v1/engagements/status", securityHeaders(http.HandlerFunc(AuthMiddleware(handleGetEngagementStatus))))
	mux.Handle("/api/v1/engagements/approve", securityHeaders(http.HandlerFunc(AuthMiddleware(handleApproveEngagement))))

	return mux
}

// --- Red Team Handlers ---

func handleStartEngagement(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Target string `json:"target"`
		Mode   string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Target == "" || len(req.Target) > 2048 {
		sendError(w, http.StatusBadRequest, "Target must be non-empty and <= 2048 characters")
		return
	}

	eng := globalOrchestrator.StartEngagement(req.Target)
	go globalOrchestrator.Execute(context.Background(), eng.ID)

	sendJSON(w, http.StatusAccepted, map[string]string{
		"message":      fmt.Sprintf("Started Red Team engagement on %s", req.Target),
		"engagementId": eng.ID,
		"mode":         req.Mode,
	})
}

func handleGetEngagementStatus(w http.ResponseWriter, r *http.Request) {
	engID := r.URL.Query().Get("id")
	if engID == "" {
		sendError(w, http.StatusBadRequest, "Missing engagement ID")
		return
	}

	eng, ok := globalOrchestrator.ActiveEngagements[engID]
	if !ok {
		sendError(w, http.StatusNotFound, "Engagement not found")
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"engagementId": eng.ID,
		"status":       string(eng.Status),
		"target":       eng.Target,
	})
}

func handleApproveEngagement(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EngagementID string `json:"engagementId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := globalOrchestrator.ApproveEngagement(req.EngagementID); err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Engagement %s approved and resuming exploitation", req.EngagementID),
		"status":  "EXPLOITATION",
	})
}
