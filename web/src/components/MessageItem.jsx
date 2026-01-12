import { formatTime } from "../utils/timeFormatter";
import { splitTextByJsonAssignments } from "../utils/jsonPrettify";
import { buildCurlFromText } from "../utils/httpCurl";

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
  const segments = splitTextByJsonAssignments(message.text);
  const curlInfo = buildCurlFromText(message.text);
  const curl = curlInfo?.curl;
  let curlButtonShown = false;

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
        {segments.map((seg, i) => {
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
            const showCurl = Boolean(curl) && isUrl && !curlButtonShown;
            if (showCurl) curlButtonShown = true;
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
        })}
      </span>
    </div>
  );
}
