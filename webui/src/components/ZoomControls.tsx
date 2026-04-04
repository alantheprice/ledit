import React from 'react';
import { ZoomIn, ZoomOut, RotateCcw } from 'lucide-react';
import './ZoomControls.css';

interface ZoomControlsProps {
  zoomLevel: number;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onReset: () => void;
  onZoomChange: (level: number) => void;
}

const MIN_ZOOM = 50;
const MAX_ZOOM = 200;
const ZOOM_STEP = 10;

const ZoomControls: React.FC<ZoomControlsProps> = ({
  zoomLevel,
  onZoomIn,
  onZoomOut,
  onReset,
  onZoomChange,
}) => {
  return (
    <div className="config-item">
      <label htmlFor="ui-zoom-slider">UI Zoom</label>
      <div className="zoom-controls">
        <button
          type="button"
          className="zoom-btn"
          onClick={onZoomOut}
          title="Zoom out (−10%)"
          aria-label={`Zoom out to ${Math.max(MIN_ZOOM, zoomLevel - ZOOM_STEP)}%`}
        >
          <ZoomOut size={14} />
        </button>
        <input
          id="ui-zoom-slider"
          type="range"
          className="zoom-slider"
          min={MIN_ZOOM}
          max={MAX_ZOOM}
          step={5}
          value={zoomLevel}
          aria-valuemin={MIN_ZOOM}
          aria-valuemax={MAX_ZOOM}
          aria-valuenow={zoomLevel}
          aria-label="UI Zoom level"
          onChange={(e) => onZoomChange(Number(e.target.value))}
        />
        <button
          type="button"
          className="zoom-btn"
          onClick={onZoomIn}
          title="Zoom in (+10%)"
          aria-label={`Zoom in to ${Math.min(MAX_ZOOM, zoomLevel + ZOOM_STEP)}%`}
        >
          <ZoomIn size={14} />
        </button>
        <button
          type="button"
          className="zoom-btn"
          onClick={onReset}
          title="Reset zoom (100%)"
          aria-label="Reset zoom to 100%"
        >
          <RotateCcw size={12} />
        </button>
        <span className="zoom-value">{zoomLevel}%</span>
      </div>
    </div>
  );
};

export default ZoomControls;
