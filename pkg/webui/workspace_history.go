package webui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	workspaceHistoryFilename = "workspace_history.json"
	maxWorkspaceHistory      = 50
)

// workspaceHistoryMu serializes read-modify-write cycles on the history file
// to prevent data loss from concurrent goroutine writes.
var workspaceHistoryMu sync.Mutex

// workspaceHistoryEntry records a single workspace path usage.
type workspaceHistoryEntry struct {
	Path       string    `json:"path"`
	Type       string    `json:"type"`                   // "local" or "ssh"
	HostAlias  string    `json:"host_alias,omitempty"`   // non-empty for SSH entries
	RemotePath string    `json:"remote_path,omitempty"`  // non-empty for SSH entries
	LastUsed   time.Time `json:"last_used"`
	UseCount   int       `json:"use_count"`
}

// readWorkspaceHistory reads workspace history entries from disk.
func readWorkspaceHistory() ([]workspaceHistoryEntry, error) {
	p := filepath.Join(getLeditConfigDir(), workspaceHistoryFilename)

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var entries []workspaceHistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// writeWorkspaceHistory writes workspace history entries to disk atomically
// using a temporary file and rename.
func writeWorkspaceHistory(entries []workspaceHistoryEntry) error {
	dir := getLeditConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	p := filepath.Join(dir, workspaceHistoryFilename)
	tmp := p + ".tmp"

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		os.Remove(tmp) // best-effort cleanup on rename failure
		return err
	}
	return nil
}

// recordWorkspacePath records usage of a local workspace path. If the path
// already exists in history it updates last_used and increments use_count;
// otherwise a new entry is appended.
func recordWorkspacePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	path = filepath.Clean(path)

	workspaceHistoryMu.Lock()
	defer workspaceHistoryMu.Unlock()

	entries, err := readWorkspaceHistory()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	found := false
	for i, e := range entries {
		if e.Type == "local" && filepath.Clean(e.Path) == path {
			entries[i].LastUsed = now
			entries[i].UseCount++
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, workspaceHistoryEntry{
			Path:     path,
			Type:     "local",
			LastUsed: now,
			UseCount: 1,
		})
	}

	entries = trimAndSortHistory(entries)
	return writeWorkspaceHistory(entries)
}

// recordSSHWorkspaceUsage records usage of an SSH workspace path. If the same
// host_alias + remote_path combination already exists it updates last_used and
// increments use_count; otherwise a new entry is appended.
func recordSSHWorkspaceUsage(hostAlias, remotePath string) error {
	hostAlias = strings.TrimSpace(hostAlias)
	remotePath = strings.TrimSpace(remotePath)
	if hostAlias == "" {
		return nil
	}

	workspaceHistoryMu.Lock()
	defer workspaceHistoryMu.Unlock()

	entries, err := readWorkspaceHistory()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	found := false
	for i, e := range entries {
		if e.Type == "ssh" && e.HostAlias == hostAlias && e.RemotePath == remotePath {
			entries[i].LastUsed = now
			entries[i].UseCount++
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, workspaceHistoryEntry{
			Type:       "ssh",
			HostAlias:  hostAlias,
			RemotePath: remotePath,
			LastUsed:   now,
			UseCount:   1,
		})
	}

	entries = trimAndSortHistory(entries)
	return writeWorkspaceHistory(entries)
}

// getWorkspaceHistory returns workspace history entries sorted by last_used
// descending (most recent first).
func getWorkspaceHistory() ([]workspaceHistoryEntry, error) {
	entries, err := readWorkspaceHistory()
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsed.After(entries[j].LastUsed)
	})
	return entries, nil
}

// trimAndSortHistory limits the slice to maxWorkspaceHistory entries and
// sorts by last_used descending.
func trimAndSortHistory(entries []workspaceHistoryEntry) []workspaceHistoryEntry {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsed.After(entries[j].LastUsed)
	})
	if len(entries) > maxWorkspaceHistory {
		entries = entries[:maxWorkspaceHistory]
	}
	return entries
}
