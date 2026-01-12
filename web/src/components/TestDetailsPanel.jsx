import { useEffect } from "react";
import KeywordItem from "./KeywordItem";
import { formatTime } from "../utils/timeFormatter";

export default function TestDetailsPanel({ testDetails, onClose }) {
  useEffect(() => {
    if (!testDetails) return;

    const handleEsc = (e) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [testDetails, onClose]);

  if (!testDetails) return null;

  return (
    <aside className="test-details-panel">
      <div className="test-details-header">
        <h3>Test Details</h3>
        <button className="close-btn" onClick={onClose} title="Close (Esc)">
          ✕
        </button>
      </div>

      <div className="test-details-content">
        <div className="test-details-title">
          <h4>{testDetails.name}</h4>
          <span className={`status-badge ${testDetails.status.toLowerCase()}`}>
            {testDetails.status}
          </span>
        </div>

        {testDetails.start && (
          <div className="test-details-time">
            {formatTime(testDetails.start)} → {formatTime(testDetails.end)}
          </div>
        )}

        {testDetails.keywords && testDetails.keywords.length > 0 ? (
          <div className="keywords-section">
            <h5>Keywords</h5>
            <div className="keywords-list">
              {testDetails.keywords.map((kw, i) => (
                <KeywordItem key={i} keyword={kw} depth={0} />
              ))}
            </div>
          </div>
        ) : (
          <p className="no-data">No keyword data available for this test.</p>
        )}
      </div>
    </aside>
  );
}
