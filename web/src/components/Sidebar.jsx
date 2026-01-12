export default function Sidebar({ suites, activeSuite }) {
  // Sort suites: ones with failures first, then the rest in original order
  const sortedSuites = suites
    ? [...suites].sort((a, b) => {
        const aHasFails = a.tests.some(
          (t) => t.status.toUpperCase() === "FAIL"
        );
        const bHasFails = b.tests.some(
          (t) => t.status.toUpperCase() === "FAIL"
        );

        if (aHasFails && !bHasFails) return -1;
        if (!aHasFails && bHasFails) return 1;
        return 0; // Keep original order within each group
      })
    : [];

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h3>Suites</h3>
      </div>
      <nav className="sidebar-nav">
        {sortedSuites.map((suite) => {
          const failCount = suite.tests.filter(
            (t) => t.status.toUpperCase() === "FAIL"
          ).length;
          const passCount = suite.tests.filter(
            (t) => t.status.toUpperCase() === "PASS"
          ).length;
          const isActive = activeSuite === suite.name;

          return (
            <button
              key={suite.name}
              className={`sidebar-item ${isActive ? "active" : ""}`}
              onClick={() => {
                document
                  .getElementById(`suite-${suite.name}`)
                  ?.scrollIntoView({ behavior: "smooth", block: "start" });
              }}
            >
              <div className="sidebar-item-name">{suite.name}</div>
              <div className="sidebar-item-stats">
                {failCount > 0 && (
                  <span className="badge fail-badge">{failCount} ✗</span>
                )}
                {passCount > 0 && (
                  <span className="badge pass-badge">{passCount} ✓</span>
                )}
              </div>
            </button>
          );
        })}
      </nav>
    </div>
  );
}
