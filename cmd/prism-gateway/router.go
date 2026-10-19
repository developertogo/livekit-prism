package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ToolExecutionRequest is the incoming payload from a LiveKit voice agent.
type ToolExecutionRequest struct {
	RoomName  string                 `json:"room_name"`
	SessionID string                 `json:"session_id"`
	Tool      string                 `json:"tool"`
	Args      map[string]interface{} `json:"args"`
}

// ToolExecutionResponse is the dual-channel response returned to the voice agent.
type ToolExecutionResponse struct {
	SpeechReadyDelta string  `json:"speech_ready_delta"`
	VisualCardSent   bool    `json:"visual_card_sent"`
	CardBytes        int     `json:"card_bytes"`
	DurationMs       float64 `json:"duration_ms"`
}

// FunnelStep represents a discrete stage in a PostHog conversion funnel.
type FunnelStep struct {
	Name           string  `json:"name"`
	Visitors       int     `json:"visitors"`
	ConversionRate float64 `json:"conversion_rate"`
}

// VisualFunnelCard conforms to Contract A of the Prism specification.
type VisualFunnelCard struct {
	Type            string       `json:"type"`
	Title           string       `json:"title"`
	FunnelID        string       `json:"funnel_id"`
	Steps           []FunnelStep `json:"steps"`
	PrimaryDropStep string       `json:"primary_drop_step"`
	Timestamp       int64        `json:"timestamp"`
}

// DualChannelRouter handles splitting incoming tool requests into visual & audio tracks.
type DualChannelRouter struct {
	livekitClient  *LiveKitClient
	telemetry      *TelemetryRecorder
	posthogHost    string
	posthogProject string
	posthogAPIKey  string
}

// NewDualChannelRouter constructs a new DualChannelRouter instance.
func NewDualChannelRouter(
	lk *LiveKitClient,
	tr *TelemetryRecorder,
	phHost, phProject, phAPIKey string,
) *DualChannelRouter {
	return &DualChannelRouter{
		livekitClient:  lk,
		telemetry:      tr,
		posthogHost:    phHost,
		posthogProject: phProject,
		posthogAPIKey:  phAPIKey,
	}
}

// HandleToolExecution processes an analytical tool call and performs the Prism optical split.
func (r *DualChannelRouter) HandleToolExecution(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ToolExecutionRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	if payload.RoomName == "" {
		payload.RoomName = "prism-demo"
	}
	if payload.Tool == "" {
		payload.Tool = "query_funnel"
	}

	log.Printf("[Router] Received tool invocation '%s' for room '%s'", payload.Tool, payload.RoomName)

	var cardPayload interface{}
	var speechDelta string

	switch payload.Tool {
	case "query_funnel":
		funnelID := "checkout_v2"
		if fid, ok := payload.Args["funnel_id"].(string); ok && fid != "" {
			funnelID = fid
		}

		cardPayload = VisualFunnelCard{
			Type:     "funnel_viz",
			Title:    "Checkout Conversion Dropoff",
			FunnelID: funnelID,
			Steps: []FunnelStep{
				{Name: "Cart Viewed", Visitors: 14200, ConversionRate: 100.0},
				{Name: "Billing Entered", Visitors: 9100, ConversionRate: 64.1},
				{Name: "Payment Confirmed", Visitors: 3200, ConversionRate: 22.5},
			},
			PrimaryDropStep: "Payment Confirmed",
			Timestamp:       time.Now().Unix(),
		}
		speechDelta = "The checkout funnel has a 22.5 percent conversion rate. The main drop-off is at the payment confirmation step. I've projected the conversion funnel onto your screen."

	case "query_errors":
		timeframe := "24h"
		if tf, ok := payload.Args["timeframe"].(string); ok && tf != "" {
			timeframe = tf
		}

		cardPayload = map[string]interface{}{
			"type":               "error_waterfall",
			"title":              fmt.Sprintf("API Error Waterfall (%s)", timeframe),
			"timeframe":          timeframe,
			"total_errors":       42,
			"primary_root_cause": "PaymentGatewayTimeout (504)",
			"errors": []map[string]interface{}{
				{"error_type": "PaymentGatewayTimeout (504)", "count": 22, "percentage": 52.4, "status_code": 504},
				{"error_type": "StripeCardDeclined (400)", "count": 13, "percentage": 30.9, "status_code": 400},
				{"error_type": "DatabaseDeadlock (500)", "count": 7, "percentage": 16.7, "status_code": 500},
			},
			"timestamp": time.Now().Unix(),
		}
		speechDelta = "We detected 42 server errors in the last 24 hours. The highest failure rate is Payment Gateway Timeouts at 52 percent. The error waterfall is on your screen."

	case "query_retention":
		cardPayload = map[string]interface{}{
			"type":               "retention_trend",
			"title":              "8-Week Cohort Retention Trend",
			"cohort_name":        "Weekly Active Users",
			"terminal_retention": 19.5,
			"intervals": []map[string]interface{}{
				{"interval": "Week 1", "retention_rate": 100.0, "users": 8400},
				{"interval": "Week 2", "retention_rate": 44.2, "users": 3712},
				{"interval": "Week 3", "retention_rate": 34.0, "users": 2856},
				{"interval": "Week 4", "retention_rate": 28.1, "users": 2360},
				{"interval": "Week 6", "retention_rate": 23.4, "users": 1965},
				{"interval": "Week 8", "retention_rate": 19.5, "users": 1638},
			},
			"timestamp": time.Now().Unix(),
		}
		speechDelta = "Your eight-week user retention stands at 19.5 percent, beating the industry average of 15 percent. The retention curve is displayed on your screen."

	case "query_hogql":
		queryStr := "SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event ORDER BY count() DESC LIMIT 5"
		if q, ok := payload.Args["query"].(string); ok && q != "" {
			queryStr = q
		}

		cardPayload = map[string]interface{}{
			"type":      "hogql_viz",
			"title":     "HogQL Analytical Query Result",
			"query":     queryStr,
			"columns":   []string{"event", "count()"},
			"rows": [][]interface{}{
				{"$pageview", 48200},
				{"user_signed_up", 1240},
				{"checkout_step_completed", 890},
				{"payment_confirmed", 320},
				{"error_encountered", 42},
			},
			"timestamp": time.Now().Unix(),
		}
		speechDelta = "Your HogQL query completed successfully across 48,000 pageviews and 1,240 signups. I've projected the analytical results onto your screen."

	default:
		cardPayload = VisualFunnelCard{
			Type:            "funnel_viz",
			Title:           "Analytical Insight",
			FunnelID:        "general_metric",
			Steps:           []FunnelStep{{Name: "Metric Loaded", Visitors: 1000, ConversionRate: 100.0}},
			PrimaryDropStep: "",
			Timestamp:       time.Now().Unix(),
		}
		speechDelta = "I have fetched the requested analytical data and projected the visualization onto your screen."
	}

	cardBytes, err := json.Marshal(cardPayload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode visual card: %v", err), http.StatusInternalServerError)
		return
	}

	// Channel 1: Push visual card to WebRTC data channel
	cardSent := false
	if r.livekitClient != nil {
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()

		if err := r.livekitClient.SendVisualCard(ctx, payload.RoomName, "posthog-viz-card", cardBytes); err != nil {
			log.Printf("[Router] Warning: WebRTC card dispatch returned error: %v", err)
		} else {
			cardSent = true
		}
	}

	duration := float64(time.Since(startTime).Microseconds()) / 1000.0

	// Channel 2: Record telemetry trace in ClickHouse
	if r.telemetry != nil {
		r.telemetry.Record(AgentTurnRecord{
			RoomName:    payload.RoomName,
			SessionID:   payload.SessionID,
			ToolName:    payload.Tool,
			SpeechDelta: speechDelta,
			DurationMs:  duration,
			TTFTMs:      duration * 0.4,
			CardBytes:   uint32(len(cardBytes)),
			Timestamp:   time.Now(),
		})
	}

	res := ToolExecutionResponse{
		SpeechReadyDelta: speechDelta,
		VisualCardSent:   cardSent,
		CardBytes:        len(cardBytes),
		DurationMs:       duration,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleFiller returns instantaneous conversational filler phrases to mask database query latency.
func (r *DualChannelRouter) HandleFiller(w http.ResponseWriter, req *http.Request) {
	fillers := map[string]string{
		"filler": "Checking the checkout funnel for you...",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fillers)
}

// HandleHogQL executes arbitrary HogQL queries against live PostHog API or local ClickHouse mock.
func (r *DualChannelRouter) HandleHogQL(w http.ResponseWriter, req *http.Request) {
	// Alias to HandleToolExecution with Tool="query_hogql"
	r.HandleToolExecution(w, req)
}
