package webui

import (
	"sync"
	"testing"

	"github.com/alantheprice/ledit/pkg/events"
)

func TestReactWebServer_ReattachmentFlow(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID := "reattach-test-query"

	// Simulate starting a query
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// Simulate events being published while "disconnected"
	eventsToBuffer := []events.UIEvent{
		{ID: "event-1", Type: "query_progress", Data: map[string]interface{}{"iteration": 1}},
		{ID: "event-2", Type: "tool_start", Data: map[string]interface{}{"tool_name": "test_tool"}},
		{ID: "event-3", Type: "tool_end", Data: map[string]interface{}{"status": "completed"}},
		{ID: "event-4", Type: "stream_chunk", Data: map[string]interface{}{"chunk": "test content"}},
	}

	for _, event := range eventsToBuffer {
		ws.addMissedEvent(clientID, chatID, queryID, event)
	}

	// Verify events were buffered
	ws.mutex.RLock()
	state := ws.clientContexts[clientID].ChatQueryState[chatID]
	ws.mutex.RUnlock()
	if len(state.MissedEvents) != 4 {
		t.Errorf("Expected 4 buffered events, got %d", len(state.MissedEvents))
	}

	// Simulate reconnection
	missedEvents := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})

	// Verify all events were retrieved
	if len(missedEvents) != 4 {
		t.Errorf("Expected 4 retrieved events, got %d", len(missedEvents))
	}

	// Verify event order and content
	expectedIDs := []string{"event-1", "event-2", "event-3", "event-4"}
	for i, event := range missedEvents {
		if event.ID != expectedIDs[i] {
			t.Errorf("Expected event %d ID '%s', got '%s'", i, expectedIDs[i], event.ID)
		}
	}

	// Verify events were cleared from buffer
	ws.mutex.RLock()
	state = ws.clientContexts[clientID].ChatQueryState[chatID]
	ws.mutex.RUnlock()
	if len(state.MissedEvents) != 0 {
		t.Errorf("Expected 0 events after retrieval, got %d", len(state.MissedEvents))
	}
}

func TestReactWebServer_Reattachment_WrongQueryID(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID1 := "query-1"
	queryID2 := "query-2"

	// Start query 1 and buffer events for it
	ws.startQueryUnlocked(clientID, chatID, queryID1, "query 1")
	ws.addMissedEvent(clientID, chatID, queryID1, events.UIEvent{
		ID: "event-1", Type: "test", Data: map[string]string{"test": "data"},
	})

	// Try to get events for query 2 (which doesn't exist)
	missedEvents := ws.getMissedEvents(clientID, chatID, queryID2, chatQueryState{})

	// Should return nil
	if missedEvents != nil {
		t.Errorf("Expected nil for non-existent query, got %d events", len(missedEvents))
	}

	// Query 1 should still have its events
	ws.mutex.RLock()
	state := ws.clientContexts[clientID].ChatQueryState[chatID]
	ws.mutex.RUnlock()
	if len(state.MissedEvents) != 1 {
		t.Errorf("Expected 1 event for query 1, got %d", len(state.MissedEvents))
	}
}

func TestReactWebServer_Reattachment_EndQueryBeforeReconnect(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID := "test-query"

	// Start query
	ws.startQueryUnlocked(clientID, chatID, queryID, "test")

	// Buffer some events
	ws.addMissedEvent(clientID, chatID, queryID, events.UIEvent{
		ID: "event-1", Type: "test", Data: map[string]string{"test": "data"},
	})

	// End query before reconnection
	ws.endQuery(clientID, chatID)

	// Try to get events - should return nil because query state was cleared
	missedEvents := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})

	// Should return nil
	if missedEvents != nil {
		t.Errorf("Expected nil after endQuery, got %d events", len(missedEvents))
	}
}

func TestReactWebServer_Reattachment_MultipleChats(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"

	// Start queries for two different chats
	ws.startQueryUnlocked(clientID, "chat-1", "query-1", "query 1")
	ws.startQueryUnlocked(clientID, "chat-2", "query-2", "query 2")

	// Buffer events for each chat
	ws.addMissedEvent(clientID, "chat-1", "query-1", events.UIEvent{
		ID: "chat1-event-1", Type: "test", Data: map[string]string{"chat": "1"},
	})
	ws.addMissedEvent(clientID, "chat-2", "query-2", events.UIEvent{
		ID: "chat2-event-1", Type: "test", Data: map[string]string{"chat": "2"},
	})

	// Get events for chat-1
	events1 := ws.getMissedEvents(clientID, "chat-1", "query-1", chatQueryState{})
	if len(events1) != 1 {
		t.Errorf("Expected 1 event for chat-1, got %d", len(events1))
	}
	if events1[0].ID != "chat1-event-1" {
		t.Errorf("Expected 'chat1-event-1', got '%s'", events1[0].ID)
	}

	// Chat-2 should still have its event
	ws.mutex.RLock()
	state := ws.clientContexts[clientID].ChatQueryState["chat-2"]
	ws.mutex.RUnlock()
	if len(state.MissedEvents) != 1 {
		t.Errorf("Expected 1 event for chat-2, got %d", len(state.MissedEvents))
	}

	// End query for chat-1
	ws.endQuery(clientID, "chat-1")

	// Chat-2 query should still be active
	if !ws.hasActiveQueryForChat(clientID, "chat-2") {
		t.Error("Expected chat-2 to still have active query")
	}
}
