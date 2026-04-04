import { useState, useCallback, useLayoutEffect } from 'react';

const STORAGE_KEY = 'ledit:ui-zoom';
const DEFAULT_ZOOM = 100;
const MIN_ZOOM = 50;
const MAX_ZOOM = 200;
const ZOOM_STEP = 10;

function clamp(value: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Math.round(value)));
}

function readStoredZoom(): number {
  if (typeof window === 'undefined') return DEFAULT_ZOOM;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === null) return DEFAULT_ZOOM;
    const parsed = parseInt(raw, 10);
    return Number.isNaN(parsed) ? DEFAULT_ZOOM : clamp(parsed);
  } catch {
    return DEFAULT_ZOOM;
  }
}

/** Read the currently-applied zoom from the DOM (authoritative source). */
function readAppliedZoom(): number {
  if (typeof window === 'undefined') return DEFAULT_ZOOM;
  try {
    const applied = parseFloat((document.documentElement.style as any).zoom || '1');
    return Number.isNaN(applied) ? DEFAULT_ZOOM : clamp(Math.round(applied * 100));
  } catch {
    return DEFAULT_ZOOM;
  }
}

function applyZoom(zoomLevel: number): void {
  if (typeof window === 'undefined') return;
  // CSS `zoom` is not yet in TS DOM typings despite broad browser support.
  // See: https://developer.mozilla.org/en-US/docs/Web/CSS/zoom
  (document.documentElement.style as any).zoom = `${zoomLevel / 100}`;
}

function storeZoom(zoomLevel: number): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, String(zoomLevel));
  } catch {
    // Storage unavailable — ignore
  }
}

export function useUiZoom() {
  const [zoomLevel, setZoomLevelState] = useState<number>(readStoredZoom);

  const setZoomLevel = useCallback((zoom: number) => {
    const clamped = clamp(zoom);
    storeZoom(clamped);
    applyZoom(clamped);
    setZoomLevelState(clamped);
  }, []);

  /** Read authoritative value from DOM so instances stay in sync. */
  const zoomIn = useCallback(() => {
    setZoomLevel(readAppliedZoom() + ZOOM_STEP);
  }, [setZoomLevel]);

  const zoomOut = useCallback(() => {
    setZoomLevel(readAppliedZoom() - ZOOM_STEP);
  }, [setZoomLevel]);

  const resetZoom = useCallback(() => {
    setZoomLevel(DEFAULT_ZOOM);
  }, [setZoomLevel]);

  // Apply persisted zoom before first paint to prevent FOUC
  useLayoutEffect(() => {
    applyZoom(readStoredZoom());
  }, []);

  return {
    zoomLevel,
    setZoomLevel,
    zoomIn,
    zoomOut,
    resetZoom,
  };
}
