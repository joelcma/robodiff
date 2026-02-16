import { useEffect, useRef } from "react";

export default function FolderModal({
  currentDir,
  value,
  onChange,
  onClose,
  onSelectDir,
  onSetDir,
  selectingDir,
  canSelectDir,
  canSetDir,
}) {
  const overlayClickRef = useRef(false);

  useEffect(() => {
    const handleEsc = (e) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [onClose]);

  async function handleUsePath() {
    if (!onSetDir) return;
    const result = await onSetDir(value);
    if (result?.dir) {
      onClose();
    }
  }

  async function handleChooseFolder() {
    if (!onSelectDir) return;
    const result = await onSelectDir();
    if (result?.dir) {
      onClose();
    }
  }

  return (
    <div
      className="modal-overlay"
      onMouseDown={(e) => {
        overlayClickRef.current = e.target === e.currentTarget;
      }}
      onClick={(e) => {
        if (overlayClickRef.current && e.target === e.currentTarget) {
          onClose();
        }
        overlayClickRef.current = false;
      }}
    >
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Edit watched folder</h2>
          <button className="close-btn" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="modal-body">
          <div className="muted">Current: {currentDir || "(unknown)"}</div>
          <div
            className="search-box"
            style={{ marginTop: "12px", display: "flex", gap: "10px" }}
          >
            <input
              type="text"
              placeholder="Enter folder path..."
              value={value}
              onChange={(e) => onChange(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleUsePath();
                }
              }}
              autoFocus
            />
            {canSetDir && (
              <button
                type="button"
                className="secondary"
                onClick={handleUsePath}
                disabled={selectingDir || value.trim() === ""}
              >
                Use Path
              </button>
            )}
          </div>
          <div
            style={{
              marginTop: "18px",
              marginBottom: "12px",
              textAlign: "center",
              fontSize: "1.2em",
              fontWeight: 700,
              letterSpacing: "0.08em",
            }}
          >
            OR
          </div>
          <div className="action-buttons" style={{ justifyContent: "center" }}>
            {canSelectDir && (
              <button
                type="button"
                className="primary"
                onClick={handleChooseFolder}
                disabled={selectingDir}
              >
                {selectingDir ? "Selecting…" : "Choose Folder"}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
