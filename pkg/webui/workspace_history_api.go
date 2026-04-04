package webui

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleAPIWorkspaceHistory returns a list of recently used workspace paths,
// covering both local and SSH workspaces.
func (ws *ReactWebServer) handleAPIWorkspaceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := getWorkspaceHistory()
	if err != nil {
		log.Printf("[web] read workspace history: %v", err)
		http.Error(w, "Failed to read workspace history", http.StatusInternalServerError)
		return
	}

	if entries == nil {
		entries = []workspaceHistoryEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "success",
		"entries": entries,
	})
}
