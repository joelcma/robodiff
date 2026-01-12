import { createPortal } from "react-dom";
import { useState } from "react";

function tryPrettifyJson(text) {
  if (typeof text !== "string") return null;
  const trimmed = text.trim();
  if (!trimmed) return null;
  try {
    const parsed = JSON.parse(trimmed);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return null;
  }
}

export default function HttpResponseModal({
  data,
  onClose,
  onResend,
  isResending,
}) {
  const request = data?.request;
  const response = data?.response;
  const [showHeaders, setShowHeaders] = useState(false);

  const responseHeadersPretty = response?.headers
    ? JSON.stringify(response.headers, null, 2)
    : null;

  const responseBodyPretty = tryPrettifyJson(response?.body);

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal http-response-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="http-response-title-row">
          <h2 className="http-response-title">HTTP Response</h2>
          <div className="http-response-title-actions">
            <button
              type="button"
              className="json-copy-btn"
              title="Re-send the same request"
              disabled={
                !request?.method || !request?.url || !onResend || isResending
              }
              onClick={() => onResend?.()}
            >
              {isResending ? "Sending…" : "Re-send"}
            </button>
          </div>
        </div>

        {data?.error ? (
          <div className="http-response-error">{String(data.error)}</div>
        ) : null}

        {request?.method && request?.url ? (
          <div className="http-response-meta">
            <div className="http-response-line">
              <span className="http-response-label">Request</span>
              <span className="http-response-value">
                {request.method} {request.url}
              </span>
            </div>
          </div>
        ) : null}

        {response ? (
          <div className="http-response-meta">
            <div className="http-response-line">
              <span className="http-response-label">Status</span>
              <span className="http-response-value">
                {response.status} {response.statusText}
              </span>
            </div>
            {typeof response.durationMs === "number" ? (
              <div className="http-response-line">
                <span className="http-response-label">Duration</span>
                <span className="http-response-value">
                  {response.durationMs}ms
                </span>
              </div>
            ) : null}
          </div>
        ) : null}

        <button
          type="button"
          className="http-response-toggle"
          onClick={() => setShowHeaders((v) => !v)}
        >
          {showHeaders ? "Hide headers" : "Show headers"}
        </button>

        {showHeaders ? (
          <>
            <h3 className="http-response-section-title">Headers</h3>
            {responseHeadersPretty ? (
              <pre className="argument-json">{responseHeadersPretty}</pre>
            ) : (
              <div className="http-response-empty">(none)</div>
            )}
          </>
        ) : null}

        <h3 className="http-response-section-title">Body</h3>
        {typeof response?.body === "string" && response.body !== "" ? (
          <pre className="http-response-body">
            {responseBodyPretty != null ? responseBodyPretty : response.body}
          </pre>
        ) : (
          <div className="http-response-empty">(empty)</div>
        )}
      </div>
    </div>,
    document.body
  );
}
