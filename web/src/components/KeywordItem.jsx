import { useState } from "react";
import MessageItem from "./MessageItem";
import { formatTime } from "../utils/timeFormatter";

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

  return (
    <div className="keyword-item" style={{ marginLeft: `${indent}px` }}>
      <div
        className={`keyword-header ${hasContent ? "clickable" : ""}`}
        onClick={() => hasContent && setIsCollapsed(!isCollapsed)}
      >
        {hasContent && (
          <span className="keyword-toggle">{isCollapsed ? "▶" : "▼"}</span>
        )}
        <span
          className={`keyword-status ${
            keyword.status?.toLowerCase() || "pass"
          }`}
        >
          {keyword.status || "PASS"}
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
            <div className="keyword-arguments">
              {keyword.arguments.map((arg, i) => (
                <div key={i} className="argument-item">
                  {arg}
                </div>
              ))}
            </div>
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
