package webui

import (
	"fmt"
	"testing"
	"time"

	"github.com/alantheprice/ledit/pkg/events"
)

func TestChatQueryState_Basic(t *testing.T) {
	state := &chatQueryState{
		QueryID:    "test-query-123",
		QueryText:  "test query",
		StartedAt:  time.Now(),
		IsActive:   true,
		MissedEvents: make([]events.UIEvent, 0),
		LastSentAt: time.Now(),
	}

	if state.QueryID != "test-query-123" {
		t.Errorf("Expected QueryID 'test-query-123', got '%s'", state.QueryID)
	}
	if !state.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if len(state.MissedEvents) != 0 {
		t.Errorf("Expected 0 missed events, got %d", len(state.MissedEvents))
	}
}

func TestChatQueryState_EventBuffering(t *testing.T) {
	state := &chatQueryState{
		QueryID:    "test-query-123",
		IsActive:   true,
		MissedEvents: make([]events.UIEvent, 0),
		LastSentAt: time.Now(),
	}

	// Add events
	for i := 0; i < 10; i++ {
		event := events.UIEvent{
			ID:    fmt.Sprintf("event-%d", i),
			Type:  "test_event",
			Data:  map[string]string{"index": fmt.Sprintf("%d", i)},
		}
		state.MissedEvents = append(state.MissedEvents, event)
	}

	if len(state.MissedEvents) != 10 {
		t.Errorf("Expected 10 events, got %d", len(state.MissedEvents))
	}
}

func TestChatQueryState_BufferLimit(t *testing.T) {
	state := &chatQueryState{
		QueryID:    "test-query-123",
		IsActive:   true,
		MissedEvents: make([]events.UIEvent, 0),
		LastSentAt: time.Now(),
	}

	// Add 60 events (exceeds 50 limit)
	for i := 0; i < 60; i++ {
		event := events.UIEvent{
			ID:    fmt.Sprintf("event-%d", i),
			Type:  "test_event",
		}
		if len(state.MissedEvents) >= 50 {
			state.MissedEvents = state.MissedEvents[1:] // Remove oldest
		}
		state.MissedEvents = append(state.MissedEvents, event)
	}

	if len(state.MissedEvents) > 50 {
		t.Errorf("Expected max 50 events, got %d", len(state.MissedEvents))
	}
}
