import { Fragment, useMemo, useState } from "react";

function formatDuration(ms) {
  if (!Number.isFinite(ms) || ms <= 0) return "0s";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 10000) return `${(ms / 1000).toFixed(1).replace(/\.0$/, "")}s`;
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
  if (value >= 99.5 && value <= 100) return "100%";
  if (value >= 10) return `${value.toFixed(1)}%`;
  return `${value.toFixed(2)}%`;
}

const keywordSortOptions = {
  total: { field: "totalDurationMs", label: "Total elapsed" },
  calls: { field: "callCount", label: "Calls" },
  average: { field: "averageDurationMs", label: "Average" },
  maximum: { field: "maxDurationMs", label: "Maximum" },
};

function KeywordBreakdown({ keywords, totalDurationMs }) {
  const [sortBy, setSortBy] = useState("total");
  const [expandedKeyword, setExpandedKeyword] = useState("");

  const sortedKeywords = useMemo(() => {
    const field = keywordSortOptions[sortBy].field;
    return [...(keywords || [])].sort((a, b) => {
      const difference = (b[field] || 0) - (a[field] || 0);
      if (difference !== 0) return difference;
      return a.displayName.localeCompare(b.displayName);
    });
  }, [keywords, sortBy]);

  if (sortedKeywords.length === 0) {
    return (
      <div className="time-keyword-empty">
        No timed keyword executions were found in this run.
      </div>
    );
  }

  return (
    <div className="time-keyword-panel">
      <div className="time-keyword-toolbar">
        <p>
          Durations are inclusive of nested keyword calls, so rows do not add up
          to the total runtime.
        </p>
        <label>
          Sort by
          <select value={sortBy} onChange={(event) => setSortBy(event.target.value)}>
            {Object.entries(keywordSortOptions).map(([value, option]) => (
              <option key={value} value={value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="time-keyword-table-wrap">
        <table className="time-keyword-table">
          <thead>
            <tr>
              <th>Keyword</th>
              <th>Total elapsed</th>
              <th>Calls</th>
              <th>Average</th>
              <th>Maximum</th>
              <th>Run</th>
            </tr>
          </thead>
          <tbody>
            {sortedKeywords.map((keyword) => {
              const key = `${keyword.owner || ""}\u0000${keyword.name}`;
              const isExpanded = expandedKeyword === key;
              const percentage =
                totalDurationMs > 0
                  ? (keyword.totalDurationMs / totalDurationMs) * 100
                  : 0;
              return (
                <Fragment key={key}>
                  <tr className={isExpanded ? "expanded" : ""}>
                    <td>
                      <button
                        type="button"
                        className="time-keyword-name"
                        onClick={() => setExpandedKeyword(isExpanded ? "" : key)}
                        aria-expanded={isExpanded}
                      >
                        <span>{isExpanded ? "▼" : "▶"}</span>
                        {keyword.displayName}
                      </button>
                    </td>
                    <td>{formatDuration(keyword.totalDurationMs)}</td>
                    <td>{keyword.callCount}</td>
                    <td>{formatDuration(keyword.averageDurationMs)}</td>
                    <td>{formatDuration(keyword.maxDurationMs)}</td>
                    <td>
                      <div className="time-keyword-run-share">
                        <div className="time-tree-bar-track">
                          <div
                            className="time-tree-bar keyword"
                            style={{ width: `${Math.min(100, percentage)}%` }}
                          />
                        </div>
                        <span>{formatPercent(percentage)}</span>
                      </div>
                    </td>
                  </tr>
                  {isExpanded ? (
                    <tr key={`${key}-occurrences`} className="time-keyword-occurrences-row">
                      <td colSpan="6">
                        <div className="time-keyword-occurrences">
                          <strong>Slowest occurrences</strong>
                          <ol>
                            {(keyword.occurrences || []).map((occurrence, index) => (
                              <li key={`${occurrence.testName}-${index}`}>
                                <span className={statusClass(occurrence.status)}>
                                  {occurrence.testName}
                                </span>
                                <span className="time-pill">
                                  {formatDuration(occurrence.durationMs)}
                                </span>
                              </li>
                            ))}
                          </ol>
                          {keyword.callCount > keyword.occurrences.length ? (
                            <span className="time-subtle">
                              Showing the {keyword.occurrences.length} slowest of{" "}
                              {keyword.callCount} calls.
                            </span>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
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

export default function TimeBreakdown({ breakdown, summary, keywords }) {
  const [expanded, setExpanded] = useState(() => buildDefaultExpanded(breakdown));
  const [view, setView] = useState("tests");

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
            See which tests and keywords consume the most time.
          </p>
        </div>
        <div className="filter-buttons time-breakdown-switch" aria-label="Time breakdown view">
          <button
            type="button"
            className={view === "tests" ? "active" : ""}
            onClick={() => setView("tests")}
          >
            Tests
          </button>
          <button
            type="button"
            className={view === "keywords" ? "active" : ""}
            onClick={() => setView("keywords")}
          >
            Keywords
          </button>
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

      {view === "tests" ? (
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
      ) : (
        <KeywordBreakdown
          keywords={keywords}
          totalDurationMs={summary?.totalDurationMs || breakdown.durationMs}
        />
      )}
    </section>
  );
}
