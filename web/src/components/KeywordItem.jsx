import { useState } from "react";
import MessageItem from "./MessageItem";
import { formatTime } from "../utils/timeFormatter";
import { splitTextByJsonAssignments } from "../utils/jsonPrettify";

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

  const renderedArguments = hasArguments
    ? keyword.arguments.map((arg, i) => {
        const segments = splitTextByJsonAssignments(arg);
        return (
          <div key={i} className="argument-item">
            {segments.map((seg, j) => {
              if (seg.type === "json") {
                return (
                  <span key={`${i}-${j}`} className="argument-json-block">
                    <span className="argument-key">{seg.key}=</span>
                    <pre className="argument-json">{seg.pretty}</pre>
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
      </div>

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
