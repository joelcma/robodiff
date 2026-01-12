function formatTime(isoOrDate) {
  const d = new Date(isoOrDate);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString();
}

export default function RunList({
  runs,
  selected,
  onToggle,
  title,
  onTitleChange,
  searchQuery,
  onSearchChange,
  onSelectAll,
  onSelectFailed,
  onClearSelection,
  onGenerate,
  loadingDiff,
  sortBy,
  sortDir,
  onSort,
  showActions,
}) {
  const selectedIds = Array.from(selected);

  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Test Runs</h2>
        <div className="search-box">
          <input
            type="search"
            placeholder="Search runs... (Ctrl+F)"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
      </div>

      {showActions && (
        <div className="controls">
          <div className="control-group">
            <label>
              Title:
              <input
                value={title}
                onChange={(e) => onTitleChange(e.target.value)}
              />
            </label>
          </div>
          <div className="action-buttons">
            <button
              className="secondary"
              onClick={onSelectAll}
              title="Select all (Ctrl+A)"
            >
              ✓ All
            </button>
            <button
              className="secondary"
              onClick={onSelectFailed}
              title="Select failed (F)"
            >
              ✗ Failed
            </button>
            <button
              className="secondary"
              onClick={onClearSelection}
              title="Clear (C)"
            >
              ✕ Clear
            </button>
            <button
              className="primary"
              onClick={onGenerate}
              disabled={selectedIds.length < 1 || loadingDiff}
              title={
                selectedIds.length === 1
                  ? "View run (Ctrl+D)"
                  : "Generate diff (Ctrl+D)"
              }
            >
              {loadingDiff
                ? "⟳ Loading…"
                : selectedIds.length === 1
                ? `👁️ View Run`
                : `⚡ Compare (${selectedIds.length})`}
            </button>
          </div>
        </div>
      )}

      {runs.length === 0 ? (
        <div className="empty-state">
          {searchQuery ? "No runs match your search." : "No runs found yet."}
        </div>
      ) : (
        <div className="table-wrapper">
          <table className="runs">
            <thead>
              <tr>
                <th style={{ width: "40px" }}></th>
                <th
                  className={`sortable ${
                    sortBy === "name" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("name")}
                >
                  Name
                  {sortBy === "name" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                </th>
                <th
                  className={`sortable ${
                    sortBy === "modTime" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("modTime")}
                >
                  Modified
                  {sortBy === "modTime" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                </th>
                <th
                  className={`sortable ${
                    sortBy === "testCount" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("testCount")}
                  style={{ width: "80px" }}
                >
                  Tests
                  {sortBy === "testCount" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                </th>
                <th
                  className={`sortable ${
                    sortBy === "passCount" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("passCount")}
                  style={{ width: "80px" }}
                >
                  Pass
                  {sortBy === "passCount" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                </th>
                <th
                  className={`sortable ${
                    sortBy === "failCount" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("failCount")}
                  style={{ width: "80px" }}
                >
                  Fail
                  {sortBy === "failCount" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                </th>
                <th>Pass Rate</th>
                <th>Path</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => {
                const isSelected = selected.has(run.id);
                const passRate =
                  run.testCount > 0
                    ? Math.round((run.passCount / run.testCount) * 100)
                    : 0;
                const rateClass =
                  passRate === 100 ? "high" : passRate >= 70 ? "medium" : "low";

                return (
                  <tr
                    key={run.id}
                    className={isSelected ? "selected" : ""}
                    onClick={() => onToggle(run.id)}
                  >
                    <td>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => onToggle(run.id)}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </td>
                    <td className="name-cell">{run.name}</td>
                    <td className="time-cell">{formatTime(run.modTime)}</td>
                    <td className="num-cell">{run.testCount}</td>
                    <td className="num-cell pass-cell">{run.passCount}</td>
                    <td className="num-cell fail-cell">{run.failCount}</td>
                    <td>
                      <div className="progress-bar">
                        <div
                          className={`progress-fill ${rateClass}`}
                          style={{ width: `${passRate}%` }}
                        ></div>
                      </div>
                      <span style={{ fontSize: "0.85em", marginLeft: "8px" }}>
                        {passRate}%
                      </span>
                    </td>
                    <td>
                      <code className="path-code">{run.relPath}</code>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
