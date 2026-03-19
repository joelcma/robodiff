import { useMemo, useState } from "react";

function formatDuration(ms) {
  if (!Number.isFinite(ms) || ms <= 0) return "0s";
  const totalSeconds = Math.round(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(
      seconds,
    ).padStart(2, "0")}`;
  }
  if (minutes > 0) {
    return `${minutes}:${String(seconds).padStart(2, "0")}`;
  }
  return `${seconds}s`;
}

function formatPercent(value) {
  if (!Number.isFinite(value) || value <= 0) return "0%";
  if (value >= 99.5) return "100%";
  if (value >= 10) return `${value.toFixed(1)}%`;
  return `${value.toFixed(2)}%`;
}

function statusClass(status) {
  const value = String(status || "").toLowerCase();
  if (value === "fail") return "fail";
  if (value === "pass") return "pass";
  return "unknown";
}

function buildDefaultExpanded(node) {
  const expanded = new Set();
  if (!node) return expanded;
  expanded.add(node.fullName);
  for (const child of node.children || []) {
    if (child.type === "suite") {
      expanded.add(child.fullName);
    }
  }
  return expanded;
}

function TimeBreakdownNode({
  node,
  rootDurationMs,
  depth,
  expanded,
  onToggle,
  parentDurationMs,
}) {
  const hasChildren = Array.isArray(node.children) && node.children.length > 0;
  const isExpanded = expanded.has(node.fullName);
  const relativeToRoot =
    rootDurationMs > 0 ? (node.durationMs / rootDurationMs) * 100 : 0;
  const relativeToParent =
    parentDurationMs > 0 ? (node.durationMs / parentDurationMs) * 100 : 0;
  const displayName =
    depth === 0 ? node.fullName : node.name.split(".").pop() || node.name;

  return (
    <div className={`time-tree-node depth-${depth}`}>
      <div
        className={`time-tree-row ${node.type} ${statusClass(node.status)}`}
        style={{ "--depth": depth }}
      >
        <button
          type="button"
          className={`time-tree-label ${hasChildren ? "expandable" : "leaf"}`}
          onClick={() => hasChildren && onToggle(node.fullName)}
        >
          <span className="time-tree-toggle">
            {hasChildren ? (isExpanded ? "▼" : "▶") : "•"}
          </span>
          <span className="time-tree-name">{displayName}</span>
        </button>

        <div className="time-tree-metrics">
          <div className="time-tree-bar-track">
            <div
              className={`time-tree-bar ${node.type}`}
              style={{ width: `${Math.max(relativeToRoot, node.durationMs > 0 ? 2 : 0)}%` }}
            />
          </div>
          <div className="time-tree-values">
            <span className="time-pill">{formatDuration(node.durationMs)}</span>
            <span className="time-subtle">
              {formatPercent(relativeToRoot)} of run
            </span>
            {depth > 0 ? (
              <span className="time-subtle">
                {formatPercent(relativeToParent)} of parent
              </span>
            ) : null}
          </div>
        </div>
      </div>

      {hasChildren && isExpanded ? (
        <div className="time-tree-children">
          {node.children.map((child) => (
            <TimeBreakdownNode
              key={child.fullName}
              node={child}
              rootDurationMs={rootDurationMs}
              depth={depth + 1}
              expanded={expanded}
              onToggle={onToggle}
              parentDurationMs={node.durationMs}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

export default function TimeBreakdown({ breakdown, summary }) {
  const [expanded, setExpanded] = useState(() => buildDefaultExpanded(breakdown));

  const topTests = useMemo(() => {
    const tests = [];

    function visit(node) {
      if (!node) return;
      if (node.type === "test") {
        tests.push(node);
        return;
      }
      for (const child of node.children || []) {
        visit(child);
      }
    }

    visit(breakdown);
    return tests
      .sort((a, b) => b.durationMs - a.durationMs)
      .slice(0, 5);
  }, [breakdown]);

  if (!breakdown) return null;

  function toggleNode(fullName) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(fullName)) next.delete(fullName);
      else next.add(fullName);
      return next;
    });
  }

  return (
    <section className="time-breakdown-section">
      <div className="time-breakdown-header">
        <div>
          <h3>Time Breakdown</h3>
          <p>
            Expand suites to see which parts of the run consume the most time.
          </p>
        </div>
      </div>

      <div className="time-summary-grid">
        <div className="time-summary-card">
          <span className="time-summary-label">Total runtime</span>
          <strong>{formatDuration(summary?.totalDurationMs)}</strong>
        </div>
        <div className="time-summary-card">
          <span className="time-summary-label">Suites / tests</span>
          <strong>
            {summary?.suiteCount || 0} / {summary?.testCount || 0}
          </strong>
        </div>
        <div className="time-summary-card">
          <span className="time-summary-label">Longest suite</span>
          <strong>{summary?.longestSuiteName || "N/A"}</strong>
          <span className="time-summary-detail">
            {formatDuration(summary?.longestSuiteMs)}
          </span>
        </div>
        <div className="time-summary-card">
          <span className="time-summary-label">Longest test</span>
          <strong>{summary?.longestTestName || "N/A"}</strong>
          <span className="time-summary-detail">
            {formatDuration(summary?.longestTestMs)}
          </span>
        </div>
      </div>

      <div className="time-breakdown-layout">
        <div className="time-breakdown-tree">
          <TimeBreakdownNode
            node={breakdown}
            rootDurationMs={summary?.totalDurationMs || breakdown.durationMs}
            depth={0}
            expanded={expanded}
            onToggle={toggleNode}
            parentDurationMs={0}
          />
        </div>

        <aside className="time-breakdown-aside">
          <div className="time-side-card">
            <h4>Accounted time</h4>
            <strong>{formatDuration(summary?.accountedTestMs)}</strong>
            <span className="time-summary-detail">
              {formatPercent(summary?.accountedPct || 0)} of total runtime is
              attached directly to test cases.
            </span>
          </div>

          <div className="time-side-card">
            <h4>Slowest tests</h4>
            <ol className="time-top-list">
              {topTests.map((test) => (
                <li key={test.fullName}>
                  <span className="time-top-name">{test.fullName}</span>
                  <span className="time-pill">{formatDuration(test.durationMs)}</span>
                </li>
              ))}
            </ol>
          </div>
        </aside>
      </div>
    </section>
  );
}
