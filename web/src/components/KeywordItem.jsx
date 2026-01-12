import { useState } from "react";
import MessageItem from "./MessageItem";
import { formatTime } from "../utils/timeFormatter";
import { splitTextByJsonAssignments } from "../utils/jsonPrettify";
import {
  buildCurlFromText,
  extractHttpRequestFromText,
} from "../utils/httpCurl";
import HttpResponseModal from "./HttpResponseModal";

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

export default function KeywordItem({ keyword, depth }) {
  const indent = depth * 20;
  const hasChildren = keyword.keywords && keyword.keywords.length > 0;
  const hasMessages = keyword.messages && keyword.messages.length > 0;
  const hasArguments = keyword.arguments && keyword.arguments.length > 0;
  const hasFail = keyword.status?.toLowerCase() === "fail";

  // Collapse by default unless this keyword or any child has a failure
  const hasFailInBranch =
    hasFail || (hasChildren && hasFailureInChildren(keyword.keywords));
  const [isCollapsed, setIsCollapsed] = useState(!hasFailInBranch);

  const hasContent = hasArguments || hasMessages || hasChildren;

  // Determine effective status: if any child failed, show as failed
  const effectiveStatus = hasFailInBranch
    ? "fail"
    : keyword.status?.toLowerCase() || "pass";
  const effectiveStatusLabel = hasFailInBranch
    ? "FAIL"
    : keyword.status || "PASS";

  const requestMessageText = hasMessages
    ? keyword.messages.find((m) => buildCurlFromText(m.text))?.text
    : null;
  const curlInfo = requestMessageText
    ? buildCurlFromText(requestMessageText)
    : null;

  const requestInfo = requestMessageText
    ? extractHttpRequestFromText(requestMessageText)
    : null;

  const [httpModalData, setHttpModalData] = useState(null);
  const [isSending, setIsSending] = useState(false);

  async function tryHttpRequest(payload) {
    try {
      setIsSending(true);
      const res = await fetch("/api/http-try", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const json = await res.json().catch(() => null);
      if (!res.ok) {
        setHttpModalData({
          error: json?.error || `Request failed (${res.status})`,
          request: payload,
        });
        return;
      }
      setHttpModalData(json);
    } catch (err) {
      setHttpModalData({ error: String(err), request: payload });
    } finally {
      setIsSending(false);
    }
  }

  const renderedArguments = hasArguments
    ? keyword.arguments.map((arg, i) => {
        const segments = splitTextByJsonAssignments(arg);
        return (
          <div key={i} className="argument-item">
            {segments.map((seg, j) => {
              if (seg.type === "json") {
                return (
                  <span key={`${i}-${j}`} className="argument-json-block">
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
                return (
                  <span key={`${i}-${j}`} className="argument-json-block">
                    <span className="argument-key-row">
                      <span className="argument-key">{seg.key}=</span>
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
                    </span>
                    <div className="argument-copy-value">{seg.value}</div>
                  </span>
                );
              }
              return <span key={`${i}-${j}`}>{seg.value}</span>;
            })}
          </div>
        );
      })
    : null;

  return (
    <div className="keyword-item" style={{ marginLeft: `${indent}px` }}>
      <div
        className={`keyword-header ${hasContent ? "clickable" : ""}`}
        onClick={() => hasContent && setIsCollapsed(!isCollapsed)}
      >
        {hasContent && (
          <span className="keyword-toggle">{isCollapsed ? "▶" : "▼"}</span>
        )}
        <span className={`keyword-status ${effectiveStatus}`}>
          {effectiveStatusLabel}
        </span>
        <span className="keyword-type">{keyword.type}</span>
        <span className="keyword-name">{keyword.name}</span>
        {keyword.start && (
          <span className="keyword-time">
            {formatTime(keyword.start)} → {formatTime(keyword.end)}
          </span>
        )}
        {curlInfo?.curl ? (
          <span className="keyword-header-actions">
            <button
              type="button"
              className="json-copy-btn"
              title="Copy as curl"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                copyToClipboard(curlInfo.curl);
              }}
            >
              Curl
            </button>
            {requestInfo ? (
              <button
                type="button"
                className="json-copy-btn"
                title="Send HTTP request"
                disabled={isSending}
                onClick={async (e) => {
                  e.preventDefault();
                  e.stopPropagation();

                  if (requestInfo) {
                    await tryHttpRequest(requestInfo);
                  }
                }}
              >
                Send
              </button>
            ) : null}
          </span>
        ) : null}
      </div>

      {httpModalData ? (
        <HttpResponseModal
          data={httpModalData}
          onClose={() => setHttpModalData(null)}
          isResending={isSending}
          onResend={async () => {
            const payload = httpModalData?.request || requestInfo;
            if (payload) {
              await tryHttpRequest(payload);
            }
          }}
        />
      ) : null}

      {!isCollapsed && (
        <>
          {hasArguments && (
            <div className="keyword-arguments">{renderedArguments}</div>
          )}

          {hasMessages && (
            <div className="keyword-messages">
              {keyword.messages.map((msg, i) => (
                <MessageItem key={i} message={msg} />
              ))}
            </div>
          )}

          {hasChildren && (
            <div className="keyword-children">
              {keyword.keywords.map((child, i) => (
                <KeywordItem key={i} keyword={child} depth={depth + 1} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

function hasFailureInChildren(keywords) {
  if (!keywords || keywords.length === 0) return false;

  for (const kw of keywords) {
    if (kw.status?.toLowerCase() === "fail") return true;
    if (hasFailureInChildren(kw.keywords)) return true;
  }

  return false;
}
