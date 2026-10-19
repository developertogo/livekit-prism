package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleToolExecution_QueryFunnel(t *testing.T) {
	// Initialize router without LiveKit and ClickHouse (offline test mode)
	telemetry := NewTelemetryRecorder("", 100)
	defer telemetry.Close()
	router := NewDualChannelRouter(nil, telemetry, "", "", "")

	reqPayload := ToolExecutionRequest{
		RoomName:  "test-room",
		SessionID: "test-session-123",
		Tool:      "query_funnel",
		Args: map[string]interface{}{
			"funnel_id": "checkout_v2",
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/tools/funnel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.HandleToolExecution(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	var resp ToolExecutionResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify Contract B: Speech-ready delta
	if !strings.Contains(resp.SpeechReadyDelta, "checkout funnel") {
		t.Errorf("expected speech delta to mention checkout funnel, got: %s", resp.SpeechReadyDelta)
	}
	words := strings.Fields(resp.SpeechReadyDelta)
	if len(words) > 35 {
		t.Errorf("expected speech delta under 35 words (< 30 tokens), got %d words: %s", len(words), resp.SpeechReadyDelta)
	}

	if resp.CardBytes <= 0 {
		t.Errorf("expected non-zero card bytes, got %d", resp.CardBytes)
	}
}

func TestHandleToolExecution_QueryErrors(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	reqPayload := ToolExecutionRequest{
		RoomName:  "test-room",
		SessionID: "test-session-errors",
		Tool:      "query_errors",
		Args: map[string]interface{}{
			"timeframe": "24h",
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/tools/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.HandleToolExecution(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	var resp ToolExecutionResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.SpeechReadyDelta, "server errors") {
		t.Errorf("expected speech delta to mention server errors, got: %s", resp.SpeechReadyDelta)
	}
	if resp.CardBytes <= 0 {
		t.Errorf("expected visual card payload to be generated")
	}
}

func TestHandleToolExecution_QueryRetention(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	reqPayload := ToolExecutionRequest{
		RoomName:  "test-room",
		SessionID: "test-session-retention",
		Tool:      "query_retention",
		Args: map[string]interface{}{
			"cohort_id": "weekly_active",
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/tools/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.HandleToolExecution(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	var resp ToolExecutionResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.SpeechReadyDelta, "retention") {
		t.Errorf("expected speech delta to mention retention, got: %s", resp.SpeechReadyDelta)
	}
}

func TestHandleToolExecution_QueryHogQL(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	reqPayload := ToolExecutionRequest{
		RoomName:  "test-room",
		SessionID: "test-session-hogql",
		Tool:      "query_hogql",
		Args: map[string]interface{}{
			"query": "SELECT count() FROM events",
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/tools/hogql", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.HandleHogQL(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	var resp ToolExecutionResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.SpeechReadyDelta, "HogQL") {
		t.Errorf("expected speech delta to mention HogQL, got: %s", resp.SpeechReadyDelta)
	}
}

func TestHandleToolExecution_InvalidJSON(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/tools/execute", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	router.HandleToolExecution(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", w.Result().StatusCode)
	}
}

func TestHandleToolExecution_MethodNotAllowed(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/tools/execute", nil)
	w := httptest.NewRecorder()

	router.HandleToolExecution(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", w.Result().StatusCode)
	}
}

func TestHandleFiller(t *testing.T) {
	router := NewDualChannelRouter(nil, nil, "", "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/tools/filler", nil)
	w := httptest.NewRecorder()

	router.HandleFiller(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Result().StatusCode)
	}

	var fillers map[string]string
	if err := json.NewDecoder(w.Result().Body).Decode(&fillers); err != nil {
		t.Fatalf("failed to decode filler response: %v", err)
	}

	if fillers["filler"] != "Checking the checkout funnel for you..." {
		t.Errorf("unexpected filler phrase: %s", fillers["filler"])
	}
}
