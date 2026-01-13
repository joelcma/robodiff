import { formatTime } from "../utils/timeFormatter";
import {
  splitTextByJsonAssignments,
  tryExtractJsonishComparison,
} from "../utils/jsonPrettify";
import { buildCurlFromText } from "../utils/httpCurl";
import { diffAlignLines } from "../utils/lineDiff";

async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
    return;
  } catch {
    // fall back
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "absolute";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  document.body.removeChild(textarea);
}

function isUrlKey(key) {
  if (!key) return false;
  const lower = String(key).toLowerCase();
  return lower === "url" || lower === "path_url" || lower.endsWith("url");
}

export default function MessageItem({ message }) {
  const comparison = tryExtractJsonishComparison(message.text);
  const isFailLevel =
    String(message.level || "").toLowerCase() === "fail" ||
    String(message.level || "").toLowerCase() === "error";
  const shouldShowDiff =
    Boolean(comparison) && isFailLevel && comparison.operator === "!=";

  const diff = shouldShowDiff
    ? diffAlignLines(comparison.left.pretty, comparison.right.pretty)
    : null;
  const segments = splitTextByJsonAssignments(message.text);
  const curlInfo = buildCurlFromText(message.text);
  const curl = curlInfo?.curl;
  const firstUrlCopyIndex = segments.findIndex(
    (s) => s?.type === "copy" && isUrlKey(s.key)
  );

  return (
    <div
      className={`message-item message-${
        message.level?.toLowerCase() || "info"
      }`}
    >
      <span className="message-level">{message.level}</span>
      {message.timestamp && (
        <span className="message-timestamp">
          {formatTime(message.timestamp)}
        </span>
      )}
      <span className="message-text">
        {comparison ? (
          <>
            {comparison.prefix ? <span>{comparison.prefix}</span> : null}
            <div className="message-compare keyword-compare">
              <div className="keyword-compare-side">
                <span className="argument-key-row">
                  <span className="argument-key">left</span>
                  <button
                    type="button"
                    className="json-copy-btn"
                    title="Copy JSON"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      copyToClipboard(comparison.left.pretty);
                    }}
                  >
                    Copy
                  </button>
                </span>
                <pre className="argument-json argument-json-diff">
                  {diff
                    ? diff.rows.map((row, i) => (
                        <div
                          key={i}
                          className={`diff-line diff-line-${row.left.type}`}
                        >
                          {Array.isArray(row.left.parts)
                            ? row.left.parts.map((p, j) => (
                                <span
                                  key={j}
                                  className={`diff-ch diff-ch-${p.type}`}
                                >
                                  {p.text}
                                </span>
                              ))
                            : row.left.text}
                        </div>
                      ))
                    : comparison.left.pretty}
                </pre>
              </div>

              <div className="keyword-compare-operator">
                {comparison.operator}
              </div>

              <div className="keyword-compare-side">
                <span className="argument-key-row">
                  <span className="argument-key">right</span>
                  <button
                    type="button"
                    className="json-copy-btn"
                    title="Copy JSON"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      copyToClipboard(comparison.right.pretty);
                    }}
                  >
                    Copy
                  </button>
                </span>
                <pre className="argument-json argument-json-diff">
                  {diff
                    ? diff.rows.map((row, i) => (
                        <div
                          key={i}
                          className={`diff-line diff-line-${row.right.type}`}
                        >
                          {Array.isArray(row.right.parts)
                            ? row.right.parts.map((p, j) => (
                                <span
                                  key={j}
                                  className={`diff-ch diff-ch-${p.type}`}
                                >
                                  {p.text}
                                </span>
                              ))
                            : row.right.text}
                        </div>
                      ))
                    : comparison.right.pretty}
                </pre>
              </div>
            </div>
            {comparison.suffix ? <span>{comparison.suffix}</span> : null}
          </>
        ) : (
          segments.map((seg, i) => {
            if (seg.type === "json") {
              return (
                <span key={i} className="argument-json-block">
                  <span className="argument-key-row">
                    <span className="argument-key">{seg.key}=</span>
                    <button
                      type="button"
                      className="json-copy-btn"
                      title="Copy JSON"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        copyToClipboard(seg.pretty);
                      }}
                    >
                      Copy
                    </button>
                  </span>
                  <pre className="argument-json">{seg.pretty}</pre>
                </span>
              );
            }
            if (seg.type === "copy") {
              const isUrl = isUrlKey(seg.key);
              const showCurl =
                Boolean(curl) && isUrl && i === firstUrlCopyIndex;
              return (
                <span key={i} className="argument-json-block">
                  <span className="argument-key-row">
                    <span className="argument-key">{seg.key}=</span>
                    <span className="argument-actions">
                      <button
                        type="button"
                        className="json-copy-btn"
                        title={seg.label || "Copy"}
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          copyToClipboard(seg.value);
                        }}
                      >
                        Copy
                      </button>
                      {showCurl ? (
                        <button
                          type="button"
                          className="json-copy-btn"
                          title="Copy as curl"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            copyToClipboard(curl);
                          }}
                        >
                          Curl
                        </button>
                      ) : null}
                    </span>
                  </span>
                  <div className="argument-copy-value">{seg.value}</div>
                </span>
              );
            }
            return <span key={i}>{seg.value}</span>;
          })
        )}
      </span>
    </div>
  );
}
