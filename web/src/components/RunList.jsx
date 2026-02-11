function formatTime(isoOrDate) {
  const d = new Date(isoOrDate);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString();
}

function formatDuration(ms) {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  const totalSeconds = Math.round(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(
      seconds
    ).padStart(2, "0")}`;
  }
  if (minutes > 0) {
    return `${minutes}:${String(seconds).padStart(2, "0")}`;
  }
  return `${seconds}s`;
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const idx = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  const value = bytes / Math.pow(1024, idx);
  const precision = value >= 100 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(precision)} ${units[idx]}`;
}

export default function RunList({
  runs,
  dir,
  pinned,
  selected,
  onToggle,
  onTogglePin,
  searchQuery,
  onSearchChange,
  onSelectAll,
  onSelectFailed,
  onClearSelection,
  onGenerate,
  onDeleteSelected,
  onRenameRun,
  deletingRuns,
  renamingRunId,
  loadingDiff,
  loadingRuns,
  sortBy,
  sortDir,
  onSort,
}) {
  const selectedIds = Array.from(selected);
  const totalSize = runs.reduce((sum, run) => sum + (run.size || 0), 0);

  function confirmDelete() {
    if (!onDeleteSelected) return;
    if (selectedIds.length < 1) return;

    const chosen = runs.filter((r) => selected.has(r.id));
    const names = chosen
      .slice(0, 10)
      .map((r) => `- ${r.name}`)
      .join("\n");
    const suffix =
      chosen.length > 10 ? `\n(and ${chosen.length - 10} more)` : "";

    const ok = window.confirm(
      `Delete ${selectedIds.length} selected run(s)?\n\nThis deletes the entire run folder(s) from disk and cannot be undone.\n\n${names}${suffix}`,
    );
    if (!ok) return;
    onDeleteSelected(selectedIds);
  }

  function promptRename(run) {
    if (!onRenameRun || !run?.id) return;
    const current = String(run.name || "").trim();
    const suggested = current || String(run.relPath || "").trim();
    const input = window.prompt("Rename run", suggested);
    if (input === null) return;
    const next = input.trim();
    if (!next || next === current) return;
    onRenameRun(run.id, next);
  }

  return (
    <section className="panel p-1">
      <div className="panel-header">
        <h2>Test Runs</h2>
        <div className="controls">
          <div className="search-box">
            <input
              type="search"
              placeholder="Search runs... (Ctrl+F)"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
            />
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
          </div>
        </div>
        <div className="controls">
          <div className="action-buttons">
            <button
              className="secondary"
              onClick={confirmDelete}
              disabled={selectedIds.length < 1 || deletingRuns}
              title="Delete selected runs"
            >
              {deletingRuns
                ? "🗑️ Deleting…"
                : `🗑️ Delete (${selectedIds.length})`}
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
      </div>

      {runs.length === 0 ? (
        <div className="empty-state">
          {loadingRuns
            ? "Loading runs…"
            : searchQuery
              ? "No runs match your search."
              : "No runs found yet."}
          {!loadingRuns && !searchQuery ? (
            <div className="empty-state-detail">
              <div>Watching: {dir || "(unknown)"}</div>
              <div>
                Tip: point the server at a folder with Robot output.xml files.
              </div>
            </div>
          ) : null}
        </div>
      ) : (
        <div className="table-wrapper">
          <table className="runs">
            <thead>
              <tr>
                <th style={{ width: "36px" }} title="Pinned">
                  📌
                </th>
                <th style={{ width: "40px" }}>
                  <span
                    style={{
                      position: "absolute",
                      left: "-10000px",
                      top: "auto",
                      width: "1px",
                      height: "1px",
                      overflow: "hidden",
                    }}
                  >
                    Select
                  </span>
                </th>
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
                    sortBy === "durationMs" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("durationMs")}
                  style={{ width: "110px" }}
                >
                  Duration
                  {sortBy === "durationMs" && (
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
                <th
                  className={`sortable ${
                    sortBy === "size" ? "sort-active" : ""
                  }`}
                  onClick={() => onSort("size")}
                  style={{ width: "140px" }}
                  title={`Total size: ${formatBytes(totalSize)}`}
                >
                  Size
                  {sortBy === "size" && (
                    <span className="sort-arrow">
                      {sortDir === "asc" ? "↑" : "↓"}
                    </span>
                  )}
                  <span style={{ marginLeft: "8px", opacity: 0.7 }}>
                    ({formatBytes(totalSize)})
                  </span>
                </th>
                <th>Path</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => {
                const isSelected = selected.has(run.id);
                const isPinned = pinned?.has(run.id);
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
                      <button
                        type="button"
                        className={`pin-btn ${isPinned ? "pinned" : ""}`}
                        title={isPinned ? "Unpin run" : "Pin run"}
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          onTogglePin?.(run.id);
                        }}
                      >
                        {isPinned ? "📌" : "📍"}
                      </button>
                    </td>
                    <td>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => onToggle(run.id)}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </td>
                    <td className="name-cell">
                      <div className="name-cell-content">
                        <span>{run.name}</span>
                        <button
                          type="button"
                          className="rename-btn"
                          title="Rename run"
                          disabled={renamingRunId === run.id}
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            promptRename(run);
                          }}
                        >
                          {renamingRunId === run.id ? "…" : "✎"}
                        </button>
                      </div>
                    </td>
                    <td className="time-cell">{formatTime(run.modTime)}</td>
                    <td className="num-cell">
                      {formatDuration(run.durationMs) || "—"}
                    </td>
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
                    <td className="num-cell">{formatBytes(run.size || 0)}</td>
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
