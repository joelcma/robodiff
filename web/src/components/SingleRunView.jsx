import { useLayoutEffect, useRef, useState } from "react";
import Sidebar from "./Sidebar";
import TestDetailsPanel from "./TestDetailsPanel";

export default function SingleRunView({
  singleRun,
  onClose,
  diffFilter,
  onFilterChange,
  collapsedSuites,
  onToggleSuite,
}) {
  const [testDetails, setTestDetails] = useState(null);
  const [loadingTest, setLoadingTest] = useState(null);
  const [activeSuite, setActiveSuite] = useState(null);
  const [activeTestName, setActiveTestName] = useState(null);
  const mainContentRef = useRef(null);
  const mainScrollTopRef = useRef(0);

  useLayoutEffect(() => {
    if (!mainContentRef.current || !activeTestName) return;
    const target = mainContentRef.current.querySelector(
      `[data-test-row="${CSS.escape(activeTestName)}"]`
    );
    if (!target) return;
    target.scrollIntoView({ block: "center", inline: "nearest" });
  }, [testDetails, activeTestName]);

  const handleTestClick = async (testName, suiteName) => {
    setLoadingTest(testName);
    setActiveSuite(suiteName);
    setActiveTestName(testName);

    const payload = {
      runId: singleRun.runId,
      testName: testName,
    };
    console.log("Sending test details request:", payload);

    try {
      const res = await fetch("/api/test-details", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const errorData = await res.json();
        console.error("Server error:", errorData);
        throw new Error(errorData?.error || "Failed to fetch test details");
      }

      const data = await res.json();
      if (mainContentRef.current) {
        mainScrollTopRef.current = mainContentRef.current.scrollTop;
      }
      setTestDetails(data);
    } catch (err) {
      console.error("Failed to load test details:", err);
      alert(`Failed to load test details: ${err.message}`);
    } finally {
      setLoadingTest(null);
    }
  };

  return (
    <section
      className={`panel with-sidebar ${
        testDetails ? "with-details-panel" : ""
      }`}
    >
      <Sidebar suites={singleRun.suites} activeSuite={activeSuite} />

      <div
        className="main-content"
        ref={mainContentRef}
        onScroll={() => {
          if (mainContentRef.current) {
            mainScrollTopRef.current = mainContentRef.current.scrollTop;
          }
        }}
      >
        <div className="panel-header-outer">
          <div class="panel-header">
            <h2>{singleRun.title}</h2>
            <div className="diff-meta">
              <div className="filter-buttons">
                <button
                  className={diffFilter === "all" ? "active" : ""}
                  onClick={() => onFilterChange("all")}
                >
                  All Tests
                </button>
                <button
                  className={diffFilter === "pass" ? "active" : ""}
                  onClick={() => onFilterChange("pass")}
                >
                  Passed Only
                </button>
                <button
                  className={diffFilter === "failures" ? "active" : ""}
                  onClick={() => onFilterChange("failures")}
                >
                  Failed Only
                </button>
              </div>
            </div>
            <button className="close-btn" onClick={onClose} title="Close (Esc)">
              ✕
            </button>
          </div>
          <span className="path-code">{singleRun.file}</span>
        </div>

        {singleRun.suites?.map((suite) => {
          const isCollapsed = collapsedSuites.has(suite.name);
          const filteredTests = suite.tests.filter((test) => {
            if (diffFilter === "all") return true;
            if (diffFilter === "pass")
              return test.status.toUpperCase() === "PASS";
            if (diffFilter === "failures")
              return test.status.toUpperCase() === "FAIL";
            return true;
          });

          if (filteredTests.length === 0) return null;

          return (
            <div className="suite" key={suite.name} id={`suite-${suite.name}`}>
              <h3
                className="suite-header"
                onClick={() => onToggleSuite(suite.name)}
              >
                <span className="suite-toggle">{isCollapsed ? "▶" : "▼"}</span>
                {suite.name}
                <span className="suite-count muted">
                  ({filteredTests.length} tests)
                </span>
              </h3>
              {!isCollapsed && (
                <div className="suite-content">
                  <table className="diff">
                    <thead>
                      <tr>
                        <th>Test</th>
                        <th style={{ width: "120px" }}>Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredTests.map((test, idx) => (
                        <tr
                          key={idx}
                          className={`test-row ${test.status.toLowerCase()} ${
                            loadingTest === test.name ? "loading" : ""
                          } ${activeTestName === test.name ? "selected" : ""}`}
                          data-test-row={test.name}
                          onClick={() => handleTestClick(test.name, suite.name)}
                          style={{ cursor: "pointer" }}
                          title="Click to view test details"
                        >
                          <td className="name-cell">
                            {test.name}
                            {loadingTest === test.name && (
                              <span className="loading-spinner"> ⏳</span>
                            )}
                          </td>
                          <td>
                            <span
                              className={`status status-${test.status.toLowerCase()}`}
                            >
                              {test.status}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          );
        })}
      </div>

      <TestDetailsPanel
        testDetails={testDetails}
        onClose={() => {
          if (mainContentRef.current) {
            mainScrollTopRef.current = mainContentRef.current.scrollTop;
          }
          setTestDetails(null);
          setActiveSuite(null);
        }}
      />
    </section>
  );
}
