package main

import (
	"testing"
	"time"
)

func TestTelemetryRecorder_OfflineBuffer(t *testing.T) {
	tr := NewTelemetryRecorder("", 50)
	defer tr.Close()

	if tr.enabled {
		t.Errorf("expected recorder to be disabled in offline test mode")
	}

	// Should not panic or block
	tr.Record(AgentTurnRecord{
		RoomName:    "test-room",
		SessionID:   "session-1",
		ToolName:    "query_funnel",
		SpeechDelta: "delta",
		DurationMs:  12.5,
		TTFTMs:      4.2,
		CardBytes:   256,
		Timestamp:   time.Now(),
	})

	// Give buffer worker a moment to process
	time.Sleep(50 * time.Millisecond)
}
