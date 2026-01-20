export default function Header({
  dir,
  runCount,
  selectedCount,
  loadingRuns,
  onRefresh,
  onShowHelp,
  showRunList,
  onToggleRunList,
  theme,
  onToggleTheme,
}) {
  return (
    <header className="header">
      <div className="title-row">
        <div className="title-group">
          <h1>Robot Diff</h1>
          <button
            className="help-btn"
            onClick={onShowHelp}
            title="Keyboard shortcuts (?)"
          >
            ?
          </button>
        </div>
        <div className="meta">
          <div className="meta-info">
            <span className="label">Watching:</span>{" "}
            <code>{dir || "(unknown)"}</code>
            <span className="divider">|</span>
            <span className="label">Runs:</span> <strong>{runCount}</strong>
            <span className="divider">|</span>
            <span className="label">Selected:</span>{" "}
            <strong>{selectedCount}</strong>
          </div>
          <div className="header-actions">
            <button
              className="theme-toggle-btn"
              onClick={onToggleTheme}
              title={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
            >
              {theme === "dark" ? "☀️" : "🌙"}
            </button>
            <button
              className="toggle-list-btn"
              onClick={onToggleRunList}
              title={showRunList ? "Hide run list" : "Show run list"}
            >
              {showRunList ? "▼" : "▶"}
            </button>
            <button
              onClick={onRefresh}
              disabled={loadingRuns}
              title="Refresh (R)"
            >
              {loadingRuns ? "⟳ Refreshing…" : "⟳ Refresh"}
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
