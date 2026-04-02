package webui

import (
	"sync"
	"testing"
	"time"

	"github.com/alantheprice/ledit/pkg/events"
)

// TestReactWebServer_ConcurrentQueryOperations tests that query state
// operations are thread-safe under concurrent access
func TestReactWebServer_ConcurrentQueryOperations(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		mutex:          sync.RWMutex{},
		eventBus:       events.NewEventBus(),
	}

	clientID := "test-client"
	chatID := "chat-1"
	queryID := "query-123"

	// Start a query (uses startQueryUnlocked which acquires its own lock)
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// Run concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Multiple goroutines adding missed events
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := events.UIEvent{
				ID:   "event-" + string(rune(idx)),
				Type: "test_event",
				Data: map[string]string{"index": string(rune(idx))},
			}
			ws.addMissedEvent(clientID, chatID, queryID, event)
		}(i)
	}

	// Multiple goroutines getting missed events
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			events := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})
			if events == nil && len(events) == 0 {
				// This is OK - events may have been consumed by another goroutine
				return
			}
		}()
	}

	// Multiple goroutines checking active query status
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasActive := ws.hasActiveQueryForChat(clientID, chatID)
			if !hasActive {
				errors <- nil // OK
			}
		}()
	}

	// Multiple goroutines ending the query
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws.endQuery(clientID, chatID)
		}()
	}

	wg.Wait()
	close(errors)

	// Verify no panics occurred
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}

	// Verify query is ended
	if ws.hasActiveQueryForChat(clientID, chatID) {
		t.Error("Expected query to be ended after concurrent endQuery calls")
	}
}

// TestReactWebServer_QueryTimeoutCleanup tests that old events are cleaned up
func TestReactWebServer_QueryTimeoutCleanup(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		mutex:          sync.RWMutex{},
		eventBus:       events.NewEventBus(),
	}

	clientID := "test-client"
	chatID := "chat-1"
	queryID := "query-456"

	// Start a query (uses startQueryUnlocked which acquires its own lock)
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// Add some events
	event := events.UIEvent{
		ID:   "event-1",
		Type: "test_event",
		Data: map[string]string{"test": "data"},
	}
	ws.addMissedEvent(clientID, chatID, queryID, event)

	// Manually set LastSentAt to 11 minutes ago (older than 10 minute timeout)
	ws.mutex.Lock()
	if ctx := ws.clientContexts[clientID]; ctx != nil {
		if state, exists := ctx.ChatQueryState[chatID]; exists {
			state.LastSentAt = time.Now().Add(-11 * time.Minute)
		}
	}
	ws.mutex.Unlock()

	// Try to get missed events - should return nil due to timeout
	missedEvents := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})
	if missedEvents != nil && len(missedEvents) > 0 {
		t.Errorf("Expected nil events due to timeout, got %d events", len(missedEvents))
	}
}

// TestReactWebServer_MultipleChatsIsolation tests that different chats
// have isolated query states
func TestReactWebServer_MultipleChatsIsolation(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		mutex:          sync.RWMutex{},
		eventBus:       events.NewEventBus(),
	}

	clientID := "test-client"

	// Start queries for two different chats (uses startQueryUnlocked)
	ws.startQueryUnlocked(clientID, "chat-1", "query-1", "query 1")
	ws.startQueryUnlocked(clientID, "chat-2", "query-2", "query 2")

	// Add events to chat-1
	event1 := events.UIEvent{
		ID:   "event-1",
		Type: "test_event",
		Data: map[string]string{"chat": "1"},
	}
	ws.addMissedEvent(clientID, "chat-1", "query-1", event1)

	// Add events to chat-2
	event2 := events.UIEvent{
		ID:   "event-2",
		Type: "test_event",
		Data: map[string]string{"chat": "2"},
	}
	ws.addMissedEvent(clientID, "chat-2", "query-2", event2)

	// Get events for chat-1
	events1 := ws.getMissedEvents(clientID, "chat-1", "query-1", chatQueryState{})
	if len(events1) != 1 {
		t.Errorf("Expected 1 event for chat-1, got %d", len(events1))
	}

	// Get events for chat-2
	events2 := ws.getMissedEvents(clientID, "chat-2", "query-2", chatQueryState{})
	if len(events2) != 1 {
		t.Errorf("Expected 1 event for chat-2, got %d", len(events2))
	}

	// Verify chat-1 still has no events after retrieving chat-2's events
	events1Again := ws.getMissedEvents(clientID, "chat-1", "query-1", chatQueryState{})
	if events1Again != nil && len(events1Again) > 0 {
		t.Errorf("Expected 0 events for chat-1 after retrieval, got %d", len(events1Again))
	}
}

// TestReactWebServer_QueryIDMismatch tests that events are not buffered
// for wrong query IDs
func TestReactWebServer_QueryIDMismatch(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		mutex:          sync.RWMutex{},
		eventBus:       events.NewEventBus(),
	}

	clientID := "test-client"
	chatID := "chat-1"

	// Start a query (uses startQueryUnlocked)
	ws.startQueryUnlocked(clientID, chatID, "query-1", "test query")

	// Try to add event with wrong query ID
	event := events.UIEvent{
		ID:   "event-1",
		Type: "test_event",
		Data: map[string]string{"test": "data"},
	}
	ws.addMissedEvent(clientID, chatID, "wrong-query-id", event)

	// Get missed events - should be empty
	missedEvents := ws.getMissedEvents(clientID, chatID, "query-1", chatQueryState{})
	if missedEvents != nil && len(missedEvents) > 0 {
		t.Errorf("Expected 0 events for wrong query ID, got %d", len(missedEvents))
	}
}

// TestReactWebServer_CompletedQueryBuffering tests that events are not buffered
// for completed queries
func TestReactWebServer_CompletedQueryBuffering(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		mutex:          sync.RWMutex{},
		eventBus:       events.NewEventBus(),
	}

	clientID := "test-client"
	chatID := "chat-1"
	queryID := "query-789"

	// Start and complete a query (uses startQueryUnlocked)
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")
	ws.endQuery(clientID, chatID)

	// Try to add event after query completion
	event := events.UIEvent{
		ID:   "event-1",
		Type: "test_event",
		Data: map[string]string{"test": "data"},
	}
	ws.addMissedEvent(clientID, chatID, queryID, event)

	// Get missed events - should be empty
	missedEvents := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})
	if missedEvents != nil && len(missedEvents) > 0 {
		t.Errorf("Expected 0 events for completed query, got %d", len(missedEvents))
	}
}
