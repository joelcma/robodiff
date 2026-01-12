function calculateTestStatus(results) {
  const hasPass = results.includes("PASS");
  const hasFail = results.includes("FAIL");
  const hasMissing = results.includes("MISSING");
  if (hasPass && hasFail) return "diff";
  if (hasMissing) return "missing";
  if (hasPass) return "all_passed";
  return "all_failed";
}

function statusLabel(status) {
  switch (status) {
    case "diff":
      return "DIFF";
    case "missing":
      return "MISSING";
    case "all_passed":
      return "PASS";
    case "all_failed":
      return "FAIL";
    default:
      return status;
  }
}

function statusClass(status) {
  switch (status) {
    case "diff":
      return "status-diff";
    case "missing":
      return "status-missing";
    case "all_passed":
      return "status-pass";
    case "all_failed":
      return "status-fail";
    default:
      return "";
  }
}

export default function DiffView({
  diff,
  onClose,
  diffFilter,
  onFilterChange,
  filteredDiffSuites,
  collapsedSuites,
  onToggleSuite,
}) {
  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Comparison Results</h2>
        <button className="close-btn" onClick={onClose} title="Close (Esc)">
          ✕
        </button>
      </div>
      <div className="diff-meta">
        <span>{diff.columns?.length || 0} runs compared</span>
        <div className="filter-buttons">
          <button
            className={diffFilter === "all" ? "active" : ""}
            onClick={() => onFilterChange("all")}
          >
            All Tests
          </button>
          <button
            className={diffFilter === "diffs" ? "active" : ""}
            onClick={() => onFilterChange("diffs")}
          >
            Differences Only
          </button>
          <button
            className={diffFilter === "failures" ? "active" : ""}
            onClick={() => onFilterChange("failures")}
          >
            Failures Only
          </button>
        </div>
      </div>

      {filteredDiffSuites.map((suite) => {
        const isCollapsed = collapsedSuites.has(suite.name);
        return (
          <div className="suite" key={suite.name}>
            <h3
              className="suite-header"
              onClick={() => onToggleSuite(suite.name)}
            >
              <span className="collapse-icon">{isCollapsed ? "▶" : "▼"}</span>
              {suite.name}
              <span className="suite-count">({suite.tests.length} tests)</span>
            </h3>
            {!isCollapsed && (
              <div className="table-wrapper">
                <table className="diff">
                  <thead>
                    <tr>
                      <th>Test</th>
                      {(diff.columns || []).map((c) => (
                        <th key={c}>{c}</th>
                      ))}
                      <th style={{ width: "100px" }}>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {suite.tests.map((t) => {
                      const st = calculateTestStatus(t.results || []);
                      return (
                        <tr key={t.name} className={`test-row ${st}`}>
                          <td className="test-name">{t.name}</td>
                          {(t.results || []).map((v, i) => (
                            <td
                              key={i}
                              className={`result-cell result-${v.toLowerCase()}`}
                            >
                              <span className="cell-badge">{v}</span>
                            </td>
                          ))}
                          <td>
                            <span className={`status ${statusClass(st)}`}>
                              {statusLabel(st)}
                            </span>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        );
      })}
    </section>
  );
}
