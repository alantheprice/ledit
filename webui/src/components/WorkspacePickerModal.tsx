import React, { useState, useEffect, useCallback, useRef } from 'react';
import LeditLogo from './LeditLogo';
import { ApiService, WorkspaceHistoryEntry, WorkspaceBrowseResponse } from '../services/api';
import './WorkspacePickerModal.css';

interface WorkspacePickerModalProps {
  onWorkspaceSelected: (path: string) => void;
  onSSHSelected: (hostAlias: string, remotePath: string) => void;
  onDismissed: () => void;
}

type BrowseFileEntry = WorkspaceBrowseResponse['files'][number];

const MAX_LOCAL_ENTRIES = 8;
const MAX_SSH_ENTRIES = 4;

const formatRelativeTime = (isoDate: string): string => {
  try {
    const then = new Date(isoDate).getTime();
    const now = Date.now();
    const deltaMs = now - then;
    const mins = Math.floor(deltaMs / 60_000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    const days = Math.floor(hrs / 24);
    if (days < 30) return `${days}d ago`;
    return new Date(isoDate).toLocaleDateString();
  } catch {
    return '';
  }
};

/** Split a filesystem path into its components, excluding empty segments. */
const splitPath = (p: string): string[] => {
  // Normalise backslashes for Windows paths that might arrive this way.
  const normalised = p.replace(/\\/g, '/');
  return normalised.split('/').filter(Boolean);
};

const WorkspacePickerModal: React.FC<WorkspacePickerModalProps> = ({
  onWorkspaceSelected,
  onSSHSelected,
  onDismissed,
}) => {
  const api = ApiService.getInstance();

  // History state
  const [history, setHistory] = useState<WorkspaceHistoryEntry[]>([]);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [historyError, setHistoryError] = useState<string | null>(null);

  // Browse state
  const [browsing, setBrowsing] = useState(false);
  const [browsePath, setBrowsePath] = useState('');
  const [browseFiles, setBrowseFiles] = useState<BrowseFileEntry[]>([]);
  const [browseLoading, setBrowseLoading] = useState(false);
  const [browseError, setBrowseError] = useState<string | null>(null);

  // Custom path state
  const [customPath, setCustomPath] = useState('');
  const [customSubmitting, setCustomSubmitting] = useState(false);

  // Ref so we don't fire handlers after unmount
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => { mountedRef.current = false; };
  }, []);

  // Load workspace history on mount
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const resp = await api.getWorkspaceHistory();
        if (cancelled) return;
        setHistory(Array.isArray(resp.entries) ? resp.entries : []);
      } catch (err) {
        if (cancelled) return;
        setHistoryError(err instanceof Error ? err.message : 'Failed to load history');
      } finally {
        if (!cancelled) {
          setHistoryLoading(false);
        }
      }
    })();
    return () => { cancelled = true; };
  }, [api]);

  // Escape key handler
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (browsing) {
          setBrowsing(false);
        } else {
          onDismissed();
        }
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [browsing, onDismissed]);

  // Separate local and SSH entries
  const localEntries = history
    .filter((e) => e.type === 'local' && e.path)
    .slice(0, MAX_LOCAL_ENTRIES);

  const sshEntries = history
    .filter((e) => e.type === 'ssh' && e.host_alias)
    .slice(0, MAX_SSH_ENTRIES);

  // Browse handlers
  const startBrowse = useCallback(async () => {
    setBrowsing(true);
    setBrowseLoading(true);
    setBrowseError(null);
    try {
      const resp = await api.browseWorkspace(); // defaults to daemon root
      if (!mountedRef.current) return;
      setBrowsePath(resp.path);
      setBrowseFiles(Array.isArray(resp.files) ? resp.files : []);
    } catch (err) {
      if (!mountedRef.current) return;
      setBrowseError(err instanceof Error ? err.message : 'Failed to browse');
    } finally {
      if (mountedRef.current) setBrowseLoading(false);
    }
  }, [api]);

  const navigateBrowse = useCallback(async (dirPath: string) => {
    setBrowseLoading(true);
    setBrowseError(null);
    try {
      const resp = await api.browseWorkspace(dirPath);
      if (!mountedRef.current) return;
      setBrowsePath(resp.path);
      setBrowseFiles(Array.isArray(resp.files) ? resp.files : []);
    } catch (err) {
      if (!mountedRef.current) return;
      setBrowseError(err instanceof Error ? err.message : 'Failed to browse');
    } finally {
      if (mountedRef.current) setBrowseLoading(false);
    }
  }, [api]);

  const cancelBrowse = useCallback(() => {
    setBrowsing(false);
  }, []);

  // Custom path handler
  const handleCustomGo = useCallback(async () => {
    const trimmed = customPath.trim();
    if (!trimmed) return;
    setCustomSubmitting(true);
    try {
      onWorkspaceSelected(trimmed);
    } finally {
      if (mountedRef.current) setCustomSubmitting(false);
    }
  }, [customPath, onWorkspaceSelected]);

  const handleCustomKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleCustomGo();
    }
  }, [handleCustomGo]);

  // Breadcrumb navigation
  const breadcrumbs = splitPath(browsePath);

  const handleCrumbClick = useCallback((index: number) => {
    // Rebuild the path up to (and including) the segment at `index`.
    // index=0 → "/", index=1 → "/home", index=2 → "/home/user", etc.
    const rebuilt = '/' + breadcrumbs.slice(0, index + 1).join('/');
    navigateBrowse(rebuilt || '/');
  }, [breadcrumbs, navigateBrowse]);

  // Show only directories in browse list, sorted directories-first
  const dirEntries = browseFiles
    .filter((f) => f.type === 'directory')
    .sort((a, b) => a.name.localeCompare(b.name));
  const nonHiddenDirs = dirEntries.filter((f) => !f.name.startsWith('.'));
  const hiddenDirs = dirEntries.filter((f) => f.name.startsWith('.'));

  return (
    <div className="wp-overlay" role="dialog" aria-modal="true" aria-label="Open Workspace">
      <div className="wp-card">
        {/* Header */}
        <div className="wp-header">
          <div className="wp-logo-wrap">
            <LeditLogo compact />
          </div>
          <h1 className="wp-title">Open Workspace</h1>
          <p className="wp-subtitle">Select a workspace to get started</p>
        </div>

        {/* Loading state */}
        {historyLoading && (
          <div className="wp-loading">
            <div className="wp-loading-spinner" />
            Loading workspaces…
          </div>
        )}

        {/* Error state */}
        {historyError && !historyLoading && (
          <div className="wp-error">{historyError}</div>
        )}

        {/* Recent Local Workspaces */}
        {!historyLoading && localEntries.length > 0 && (
          <div className="wp-section">
            <h2 className="wp-section-title">Recent Local Workspaces</h2>
            <div className="wp-entries">
              {localEntries.map((entry) => (
                <button
                  key={`local-${entry.path}`}
                  type="button"
                  className="wp-entry"
                  onClick={() => onWorkspaceSelected(entry.path)}
                >
                  <span className="wp-entry-icon" aria-hidden="true">📁</span>
                  <div className="wp-entry-body">
                    <div className="wp-entry-path" title={entry.path}>
                      {entry.path}
                    </div>
                    {entry.last_used && (
                      <div className="wp-entry-meta">{formatRelativeTime(entry.last_used)}</div>
                    )}
                  </div>
                  {entry.use_count > 1 && (
                    <span className="wp-entry-badge">{entry.use_count}×</span>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Recent SSH Workspaces */}
        {!historyLoading && sshEntries.length > 0 && (
          <div className="wp-section">
            <h2 className="wp-section-title">Recent SSH Workspaces</h2>
            <div className="wp-entries">
              {sshEntries.map((entry) => {
                const label = entry.remote_path
                  ? `${entry.host_alias} · ${entry.remote_path}`
                  : entry.host_alias;
                return (
                  <button
                    key={`ssh-${entry.host_alias}-${entry.remote_path}`}
                    type="button"
                    className="wp-entry"
                    onClick={() => onSSHSelected(entry.host_alias!, entry.remote_path || '')}
                  >
                    <span className="wp-entry-icon wp-entry-icon--ssh" aria-hidden="true">🌐</span>
                    <div className="wp-entry-body">
                      <div className="wp-entry-path" title={label}>
                        {entry.host_alias}
                      </div>
                      {entry.remote_path && (
                        <div className="wp-entry-meta">{entry.remote_path}</div>
                      )}
                    </div>
                    {entry.last_used && (
                      <div className="wp-entry-meta">{formatRelativeTime(entry.last_used)}</div>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        {/* Browse button / inlined browse panel */}
        <div className="wp-browse-wrapper">
          {!browsing && (
            <button type="button" className="wp-browse-btn" onClick={startBrowse}>
              📂 Browse Files…
            </button>
          )}

          {browsing && (
            <div className="wp-browse-panel">
              {/* Breadcrumbs */}
              <div className="wp-browse-breadcrumbs">
                <button
                  type="button"
                  className="wp-browse-crumb"
                  onClick={() => navigateBrowse('/')}
                >
                  /
                </button>
                {breadcrumbs.map((segment, idx) => (
                  <React.Fragment key={idx}>
                    <span className="wp-browse-crumb-sep">/</span>
                    <button
                      type="button"
                      className={
                        idx === breadcrumbs.length - 1
                          ? 'wp-browse-crumb wp-browse-crumb--current'
                          : 'wp-browse-crumb'
                      }
                      disabled={idx === breadcrumbs.length - 1}
                      onClick={() => handleCrumbClick(idx)}
                    >
                      {segment}
                    </button>
                  </React.Fragment>
                ))}
              </div>

              {/* File listing */}
              <div className="wp-browse-list">
                {browseLoading && (
                  <div className="wp-loading" style={{ padding: '16px 0' }}>
                    <div className="wp-loading-spinner" />
                  </div>
                )}

                {browseError && <div className="wp-error" style={{ margin: '4px 8px' }}>{browseError}</div>}

                {!browseLoading && nonHiddenDirs.map((file) => (
                  <div
                    key={file.path}
                    className="wp-browse-item"
                    role="button"
                    tabIndex={0}
                    onClick={() => navigateBrowse(file.path)}
                    onKeyDown={(e) => { if (e.key === 'Enter') navigateBrowse(file.path); }}
                  >
                    <span className="wp-browse-item-icon wp-browse-item-icon--dir" aria-hidden="true">
                      📁
                    </span>
                    <span className="wp-browse-item-name">{file.name}</span>
                  </div>
                ))}

                {!browseLoading && hiddenDirs.length > 0 && nonHiddenDirs.length > 0 && (
                  <div style={{ height: 4 }} />
                )}

                {!browseLoading && hiddenDirs.map((file) => (
                  <div
                    key={file.path}
                    className="wp-browse-item"
                    style={{ opacity: 0.6 }}
                    role="button"
                    tabIndex={0}
                    onClick={() => navigateBrowse(file.path)}
                    onKeyDown={(e) => { if (e.key === 'Enter') navigateBrowse(file.path); }}
                  >
                    <span className="wp-browse-item-icon wp-browse-item-icon--dir" aria-hidden="true">
                      📁
                    </span>
                    <span className="wp-browse-item-name">{file.name}</span>
                  </div>
                ))}

                {!browseLoading && nonHiddenDirs.length === 0 && hiddenDirs.length === 0 && !browseError && (
                  <div className="wp-section-empty" style={{ padding: '12px 8px' }}>
                    Empty directory
                  </div>
                )}
              </div>

              {/* Footer: select / cancel */}
              <div className="wp-browse-footer">
                <button type="button" className="wp-browse-cancel-btn" onClick={cancelBrowse}>
                  Cancel
                </button>
                <button
                  type="button"
                  className="wp-browse-select-btn"
                  onClick={() => onWorkspaceSelected(browsePath)}
                >
                  Select This Folder
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Custom path input */}
        <div>
          <h2 className="wp-section-title" style={{ marginBottom: 6 }}>Enter Path Manually</h2>
          <div className="wp-custom-input-row">
            <input
              type="text"
              className="wp-custom-input"
              placeholder="/home/user/my-project"
              value={customPath}
              onChange={(e) => setCustomPath(e.target.value)}
              onKeyDown={handleCustomKeyDown}
              spellCheck={false}
              autoComplete="off"
            />
            <button
              type="button"
              className="wp-custom-go-btn"
              disabled={!customPath.trim() || customSubmitting}
              onClick={handleCustomGo}
            >
              Go
            </button>
          </div>
        </div>

        {/* Dismiss */}
        <div className="wp-dismiss">
          <button type="button" className="wp-dismiss-btn" onClick={onDismissed}>
            Continue with default workspace
          </button>
        </div>
      </div>
    </div>
  );
};

export default WorkspacePickerModal;
