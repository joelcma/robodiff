import { useEffect } from "react";
import KeywordItem from "./KeywordItem";
import MessageItem from "./MessageItem";
import { formatTime } from "../utils/timeFormatter";

export default function TestDetailsModal({ testDetails, onClose }) {
  useEffect(() => {
    const handleEsc = (e) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [onClose]);

  if (!testDetails) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal-content test-details-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2>Test Details: {testDetails.name}</h2>
          <button className="close-btn" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="test-meta">
          <span className={`status-badge ${testDetails.status.toLowerCase()}`}>
            {testDetails.status}
          </span>
          {testDetails.start && (
            <span className="time-info">
              {formatTime(testDetails.start)} → {formatTime(testDetails.end)}
            </span>
          )}
        </div>

        <div className="modal-body">
          {testDetails.statusMessage && (
            <div className="test-status-message">
              <MessageItem
                message={{
                  level: testDetails.status === "FAIL" ? "FAIL" : "INFO",
                  text: testDetails.statusMessage,
                }}
                runId={testDetails.runId}
              />
            </div>
          )}

          {testDetails.keywords && testDetails.keywords.length > 0 ? (
            <div className="keywords-section">
              <h3>Keywords</h3>
              <div className="keywords-list">
                {testDetails.keywords.map((kw, i) => (
                  <KeywordItem
                    key={i}
                    keyword={kw}
                    depth={0}
                    runId={testDetails.runId}
                  />
                ))}
              </div>
            </div>
          ) : (
            !testDetails.statusMessage && (
              <p className="no-data">No keyword data available for this test.</p>
            )
          )}
        </div>
      </div>
    </div>
  );
}
