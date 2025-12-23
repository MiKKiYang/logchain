package http

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	core "tlng/ingestion/service/core"
)

// LogHandler encapsulates the logic for handling HTTP log requests
type LogHandler struct {
	svc    *core.Service
	logger *log.Logger
}

// NewLogHandler creates a new LogHandler
func NewLogHandler(s *core.Service, l *log.Logger) *LogHandler {
	return &LogHandler{svc: s, logger: l}
}

// SubmitLog handles POST /v1/logs requests
func (h *LogHandler) SubmitLog(w http.ResponseWriter, r *http.Request) {
	// start := time.Now()

	if r.Method != http.MethodPost {
		h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content-Type validation
	if r.Header.Get("Content-Type") != "application/json" {
		h.respondError(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Request size limit
	if r.ContentLength > 10*1024*1024 { // 10MB limit
		h.respondError(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// 1. Parse request body JSON
	var reqPayload struct {
		LogContent        string `json:"log_content"`
		ClientLogHash     string `json:"client_log_hash,omitempty"`
		ClientSourceOrgID string `json:"client_source_org_id,omitempty"`
		ClientTimestamp   string `json:"client_timestamp,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		h.logger.Printf("HTTP Handler: Failed to parse JSON request: %v", err)
		h.respondError(w, "Bad Request: Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 2. Validate required fields
	if reqPayload.LogContent == "" {
		h.respondError(w, "log_content is required", http.StatusBadRequest)
		return
	}

	// 2.5. Get source_org_id from header (set by API Gateway) or from payload
	sourceOrgID := r.Header.Get("X-Client-Org-ID")
	if sourceOrgID == "" {
		sourceOrgID = reqPayload.ClientSourceOrgID
	}

	// 3. Construct Service layer input
	input := &core.LogInput{
		LogContent:        reqPayload.LogContent,
		ClientLogHash:     reqPayload.ClientLogHash,
		ClientSourceOrgID: sourceOrgID,
	}

	// Parse optional timestamp
	if reqPayload.ClientTimestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, reqPayload.ClientTimestamp); err == nil {
			input.ClientTimestamp = &ts
		} else {
			h.logger.Printf("HTTP Handler: Invalid client_timestamp format: %v", err)
			// Continue processing - invalid timestamp is not fatal
		}
	}

	// 4. Call Service layer processing logic
	result, err := h.svc.SubmitLog(r.Context(), input)
	if err != nil {
		h.logger.Printf("HTTP Handler: Service layer processing failed: %v", err)

		// Map service errors to appropriate HTTP status codes
		statusCode := http.StatusInternalServerError
		if err.Error() == "log_content cannot be empty" {
			statusCode = http.StatusBadRequest
		} else if matched, _ := regexp.MatchString(`client provided hash .* does not match`, err.Error()); matched {
			statusCode = http.StatusBadRequest
		}

		h.respondError(w, err.Error(), statusCode)
		return
	}

	// 5. Log processing metrics
	// duration := time.Since(start)
	// h.logger.Printf("HTTP Handler: Processed log submission in %v, request_id: %s", duration, result.RequestID)

	// 6. Construct and return success response (HTTP 202 Accepted)
	respPayload := map[string]interface{}{
		"request_id":                result.RequestID,
		"server_log_hash":           result.ServerLogHash,
		"server_received_timestamp": result.ServerReceivedTimestamp.Format(time.RFC3339Nano),
		"status":                    "ACCEPTED",
	}

	h.respondJSON(w, respPayload, http.StatusAccepted)
}

// HealthCheck handles GET /health requests
func (h *LogHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"service":   "api-gateway",
	}

	h.respondJSON(w, resp, http.StatusOK)
}

// Metrics handles GET /metrics requests (basic metrics)
func (h *LogHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic metrics - in production, use proper metrics library
	resp := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"service":   "api-gateway",
		"version":   "1.0.0",
	}

	h.respondJSON(w, resp, http.StatusOK)
}

// respondJSON sends JSON response
func (h *LogHandler) respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("HTTP Handler: Failed to encode JSON response: %v", err)
		// Cannot send error to client at this point
	}
}

// respondError sends error response
func (h *LogHandler) respondError(w http.ResponseWriter, message string, statusCode int) {
	errorResp := map[string]interface{}{
		"error":   message,
		"status":  statusCode,
		"message": http.StatusText(statusCode),
	}

	h.respondJSON(w, errorResp, statusCode)
}
