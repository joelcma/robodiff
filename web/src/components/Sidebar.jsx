export default function Sidebar({ suites, activeSuite }) {
  function normStatus(v) {
    return String(v || "")
      .trim()
      .toUpperCase();
  }

  function countSuite(suite) {
    let pass = 0;
    let fail = 0;
    for (const t of suite?.tests || []) {
      const s = normStatus(t?.status);
      if (s === "PASS") pass += 1;
      else if (s === "FAIL") fail += 1;
    }
    return { pass, fail };
  }

  // Sort suites: ones with failures first, then the rest in original order
  const sortedSuites = suites
    ? [...suites].sort((a, b) => {
        const aHasFails = (a?.tests || []).some(
          (t) => normStatus(t?.status) === "FAIL"
        );
        const bHasFails = (b?.tests || []).some(
          (t) => normStatus(t?.status) === "FAIL"
        );

        if (aHasFails && !bHasFails) return -1;
        if (!aHasFails && bHasFails) return 1;
        return 0; // Keep original order within each group
      })
    : [];

  const totals = (suites || []).reduce(
    (acc, suite) => {
      const c = countSuite(suite);
      acc.pass += c.pass;
      acc.fail += c.fail;
      return acc;
    },
    { pass: 0, fail: 0 }
  );

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h3>Suites</h3>
        <div className="sidebar-header-stats">
          <span className="badge fail-badge">{totals.fail} ✗</span>
          <span className="badge pass-badge">{totals.pass} ✓</span>
        </div>
      </div>
      <nav className="sidebar-nav">
        {sortedSuites.map((suite) => {
          const { fail: failCount, pass: passCount } = countSuite(suite);
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
