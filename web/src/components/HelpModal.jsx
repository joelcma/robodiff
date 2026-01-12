export default function HelpModal({ onClose }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Keyboard Shortcuts</h2>
        <div className="shortcuts">
          <div className="shortcut-item">
            <kbd>?</kbd>
            <span className="shortcut-desc">Show this help</span>
          </div>
          <div className="shortcut-item">
            <kbd>R</kbd>
            <span className="shortcut-desc">Refresh runs list</span>
          </div>
          <div className="shortcut-item">
            <kbd>Ctrl/Cmd</kbd> + <kbd>A</kbd>
            <span className="shortcut-desc">Select all runs</span>
          </div>
          <div className="shortcut-item">
            <kbd>C</kbd>
            <span className="shortcut-desc">Clear selection</span>
          </div>
          <div className="shortcut-item">
            <kbd>F</kbd>
            <span className="shortcut-desc">Select failed runs only</span>
          </div>
          <div className="shortcut-item">
            <kbd>Ctrl/Cmd</kbd> + <kbd>D</kbd>
            <span className="shortcut-desc">
              View run (1 selected) or Generate diff (≥2 selected)
            </span>
          </div>
          <div className="shortcut-item">
            <kbd>Esc</kbd>
            <span className="shortcut-desc">Close diff or this help</span>
          </div>
        </div>
      </div>
    </div>
  );
}
