import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import KeywordItem from "./KeywordItem";
import { formatTime } from "../utils/timeFormatter";
import { buildApiUrl } from "../utils/apiBase";

export default function TestComparisonModal({
  suiteName,
  testName,
  fullName,
  results,
  runIds,
  runNames,
  onClose,
}) {
  const [details, setDetails] = useState([]);

  const title = useMemo(() => {
    if (testName) return testName;
    if (fullName) return fullName;
    if (suiteName) return `${suiteName} / ${testName}`;
    return testName;
  }, [fullName, suiteName, testName]);

  const lookupName = useMemo(() => {
    if (fullName && fullName.includes(".")) {
      return fullName.split(".").pop() || testName;
    }
    if (testName && testName.includes(".")) {
      return testName.split(".").pop() || testName;
    }
    return testName;
  }, [fullName, testName]);

  useEffect(() => {
    if (!lookupName || !Array.isArray(runIds) || runIds.length === 0) return;

    let cancelled = false;
    const initial = runIds.map((_, i) => ({
      loading: results?.[i] !== "MISSING",
      missing: results?.[i] === "MISSING",
      error: null,
      data: null,
    }));
    setDetails(initial);

    Promise.all(
      runIds.map(async (runId, i) => {
        if (results?.[i] === "MISSING") {
          return { missing: true };
        }
        try {
          const res = await fetch(buildApiUrl("/api/test-details"), {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ runId, testName: lookupName }),
          });
          const json = await res.json().catch(() => ({}));
          if (!res.ok) {
            return { error: json?.error || `Failed (${res.status})` };
          }
          return { data: json };
        } catch (err) {
          return { error: String(err) };
        }
      }),
    ).then((items) => {
      if (cancelled) return;
      const merged = items.map((item) => ({
        loading: false,
        missing: Boolean(item?.missing),
        error: item?.error || null,
        data: item?.data || null,
      }));
      setDetails(merged);
    });

    return () => {
      cancelled = true;
    };
  }, [runIds, lookupName, results]);

  useEffect(() => {
    const handleEsc = (e) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [onClose]);

  if (!testName) return null;

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal-content test-comparison-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2 title={fullName || title}>Compare Test: {title}</h2>
          <button className="close-btn" onClick={onClose} title="Close (Esc)">
            ×
          </button>
        </div>

        <div className="comparison-grid">
          {runIds.map((runId, index) => {
            const detail = details[index] || {};
            const runLabel = runNames?.[index] || `Run ${index + 1}`;
            return (
              <div className="comparison-column" key={`${runId}-${index}`}>
                <div className="test-details-header">
                  <h3>{runLabel}</h3>
                </div>
                <div className="test-details-content">
                  {detail.loading && (
                    <p className="no-data">Loading test details…</p>
                  )}
                  {detail.missing && (
                    <p className="no-data">Test missing in this run.</p>
                  )}
                  {detail.error && <p className="no-data">{detail.error}</p>}
                  {detail.data && (
                    <>
                      <div className="test-details-title">
                        <h4>{detail.data.name}</h4>
                        <span
                          className={`status-badge ${detail.data.status.toLowerCase()}`}
                        >
                          {detail.data.status}
                        </span>
                      </div>

                      {detail.data.start && (
                        <div className="test-details-time">
                          {formatTime(detail.data.start)} →
                          {formatTime(detail.data.end)}
                        </div>
                      )}

                      {detail.data.keywords &&
                      detail.data.keywords.length > 0 ? (
                        <div className="keywords-section">
                          <h5>Keywords</h5>
                          <div className="keywords-list">
                            {detail.data.keywords.map((kw, i) => (
                              <KeywordItem
                                key={i}
                                keyword={kw}
                                depth={0}
                                runId={runId}
                              />
                            ))}
                          </div>
                        </div>
                      ) : (
                        <p className="no-data">
                          No keyword data available for this test.
                        </p>
                      )}
                    </>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>,
    document.body,
  );
}
