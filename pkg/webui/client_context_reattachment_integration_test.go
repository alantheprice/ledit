package webui

import (
	"sync"
	"testing"
	"time"

	"github.com/alantheprice/ledit/pkg/events"
)

func TestReactWebServer_QueryStateManagement(t *testing.T) {
	// Create a minimal ReactWebServer for testing
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID := "test-query-123"

	// Test startQuery
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// Verify query state was created
	state := ws.getOrCreateChatQueryState(clientID, chatID)
	if state == nil {
		t.Fatal("Expected query state to be created")
	}
	if state.QueryID != queryID {
		t.Errorf("Expected QueryID '%s', got '%s'", queryID, state.QueryID)
	}
	if !state.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if state.QueryText != "test query" {
		t.Errorf("Expected QueryText 'test query', got '%s'", state.QueryText)
	}

	// Test addMissedEvent
	testEvent := events.UIEvent{
		ID:    "event-1",
		Type:  "test_event",
		Data:  map[string]string{"test": "data"},
	}
	ws.addMissedEvent(clientID, chatID, queryID, testEvent)

	if len(state.MissedEvents) != 1 {
		t.Errorf("Expected 1 missed event, got %d", len(state.MissedEvents))
	}

	// Test getMissedEvents
	retrievedEvents := ws.getMissedEvents(clientID, chatID, queryID, chatQueryState{})
	if len(retrievedEvents) != 1 {
		t.Errorf("Expected 1 retrieved event, got %d", len(retrievedEvents))
	}
	if retrievedEvents[0].ID != "event-1" {
		t.Errorf("Expected event ID 'event-1', got '%s'", retrievedEvents[0].ID)
	}

	// Verify events were cleared
	if len(state.MissedEvents) != 0 {
		t.Errorf("Expected 0 events after retrieval, got %d", len(state.MissedEvents))
	}

	// Test endQuery
	ws.endQuery(clientID, chatID)

	// Verify query state was cleared
	ws.mutex.RLock()
	state, exists := ws.clientContexts[clientID].ChatQueryState[chatID]
	ws.mutex.RUnlock()
	if exists {
		t.Error("Expected query state to be deleted after endQuery")
	}
}

func TestReactWebServer_QueryState_MismatchedQueryID(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID1 := "query-1"
	queryID2 := "query-2"

	// Start query 1
	ws.startQueryUnlocked(clientID, chatID, queryID1, "query 1")

	// Try to add event for query 2 (should be ignored)
	testEvent := events.UIEvent{
		ID:    "event-1",
		Type:  "test_event",
		Data:  map[string]string{"test": "data"},
	}
	ws.addMissedEvent(clientID, chatID, queryID2, testEvent)

	// Verify no events were added (wrong query ID)
	ws.mutex.RLock()
	state := ws.clientContexts[clientID].ChatQueryState[chatID]
	ws.mutex.RUnlock()
	if len(state.MissedEvents) != 0 {
		t.Errorf("Expected 0 events for wrong query ID, got %d", len(state.MissedEvents))
	}

	// Verify events for correct query ID work
	ws.addMissedEvent(clientID, chatID, queryID1, testEvent)
	if len(state.MissedEvents) != 1 {
		t.Errorf("Expected 1 event for correct query ID, got %d", len(state.MissedEvents))
	}
}

func TestReactWebServer_QueryState_NonExistentChat(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "non-existent-chat"
	queryID := "test-query"

	// Try to start query for non-existent chat
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// startQuery now creates the client context, so we verify the state was created
	ws.mutex.RLock()
	ctx, exists := ws.clientContexts[clientID]
	ws.mutex.RUnlock()
	if !exists {
		t.Error("Expected client context to be created by startQuery")
	}
	if ctx == nil {
		t.Fatal("Expected client context to exist")
	}

	// Verify query state was created
	state, exists := ctx.ChatQueryState[chatID]
	if !exists {
		t.Error("Expected query state to be created")
	}
	if state == nil {
		t.Fatal("Expected query state to exist")
	}
	if state.QueryID != queryID {
		t.Errorf("Expected QueryID '%s', got '%s'", queryID, state.QueryID)
	}
}

func TestReactWebServer_QueryState_TimeoutCleanup(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"
	queryID := "test-query"

	// Start a query
	ws.startQueryUnlocked(clientID, chatID, queryID, "test query")

	// Verify cleanup doesn't remove active queries
	removed := ws.cleanupInactiveClientContexts(30 * time.Minute)

	// Should not remove the context because query is active
	if removed != 0 {
		t.Errorf("Expected 0 contexts removed, got %d", removed)
	}

	// End the query
	ws.endQuery(clientID, chatID)

	// Now cleanup should work (but won't remove because client is still connected)
	// We can't easily test full cleanup without a full server setup,
	// but we verified the active query check works
}

func TestReactWebServer_HasActiveQueryForChat(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"

	// Test with no query
	if ws.hasActiveQueryForChat(clientID, chatID) {
		t.Error("Expected no active query")
	}

	// Start a query
	ws.startQueryUnlocked(clientID, chatID, "query-1", "test")

	// Test with active query
	if !ws.hasActiveQueryForChat(clientID, chatID) {
		t.Error("Expected active query")
	}

	// End the query
	ws.endQuery(clientID, chatID)

	// Test after ending
	if ws.hasActiveQueryForChat(clientID, chatID) {
		t.Error("Expected no active query after end")
	}
}

func TestReactWebServer_GetActiveQueryID(t *testing.T) {
	ws := &ReactWebServer{
		clientContexts: make(map[string]*webClientContext),
		connections:    sync.Map{},
		mutex:          sync.RWMutex{},
	}

	clientID := "test-client"
	chatID := "test-chat"

	// Test with no query
	queryID := ws.getActiveQueryID(clientID, chatID)
	if queryID != "" {
		t.Errorf("Expected empty query ID, got '%s'", queryID)
	}

	// Start a query
	ws.startQueryUnlocked(clientID, chatID, "my-query-123", "test")

	// Test with active query
	queryID = ws.getActiveQueryID(clientID, chatID)
	if queryID != "my-query-123" {
		t.Errorf("Expected 'my-query-123', got '%s'", queryID)
	}

	// End the query
	ws.endQuery(clientID, chatID)

	// Test after ending
	queryID = ws.getActiveQueryID(clientID, chatID)
	if queryID != "" {
		t.Errorf("Expected empty query ID after end, got '%s'", queryID)
	}
}
