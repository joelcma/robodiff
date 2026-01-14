import { useEffect, useMemo, useState } from "react";
import "./App.css";
import Header from "./components/Header";
import HelpModal from "./components/HelpModal";
import RunList from "./components/RunList";
import SingleRunView from "./components/SingleRunView";
import DiffView from "./components/DiffView";

function calculateTestStatus(results = []) {
  const list = Array.isArray(results) ? results : [];
  const hasPass = list.includes("PASS");
  const hasFail = list.includes("FAIL");
  const hasMissing = list.includes("MISSING");
  if (hasPass && hasFail) return "diff";
  if (hasMissing) return "missing";
  if (hasPass) return "all_passed";
  return "all_failed";
}

function App() {
  const [runs, setRuns] = useState([]);
  const [dir, setDir] = useState("");
  const [selected, setSelected] = useState(() => new Set());
  const [title, setTitle] = useState("Robot Diff");
  const [diff, setDiff] = useState(null);
  const [singleRun, setSingleRun] = useState(null);
  const [loadingRuns, setLoadingRuns] = useState(false);
  const [loadingDiff, setLoadingDiff] = useState(false);
  const [deletingRuns, setDeletingRuns] = useState(false);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState("modTime");
  const [sortDir, setSortDir] = useState("desc");
  const [diffFilter, setDiffFilter] = useState("all");
  const [collapsedSuites, setCollapsedSuites] = useState(() => new Set());
  const [showHelp, setShowHelp] = useState(false);
  const [showRunList, setShowRunList] = useState(true);

  const selectedIds = useMemo(() => Array.from(selected), [selected]);

  // Filtered and sorted runs
  const filteredRuns = useMemo(() => {
    let filtered = runs.filter((r) =>
      searchQuery
        ? r.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          r.relPath.toLowerCase().includes(searchQuery.toLowerCase())
        : true
    );

    filtered.sort((a, b) => {
      let aVal = a[sortBy];
      let bVal = b[sortBy];
      if (sortBy === "modTime") {
        aVal = new Date(aVal).getTime();
        bVal = new Date(bVal).getTime();
      }
      const diff = sortDir === "asc" ? aVal - bVal : bVal - aVal;
      return typeof aVal === "string"
        ? sortDir === "asc"
          ? aVal.localeCompare(bVal)
          : bVal.localeCompare(aVal)
        : diff;
    });

    return filtered;
  }, [runs, searchQuery, sortBy, sortDir]);

  // Filtered diff suites
  const filteredDiffSuites = useMemo(() => {
    if (!diff?.suites) return [];
    return diff.suites
      .map((suite) => {
        let tests = suite.tests;
        if (diffFilter === "failures") {
          tests = tests.filter((t) => t.results.includes("FAIL"));
        } else if (diffFilter === "diffs") {
          tests = tests.filter((t) => {
            const st = calculateTestStatus(t.results || []);
            return st === "diff" || st === "missing";
          });
        }
        return { ...suite, tests };
      })
      .filter((s) => s.tests.length > 0);
  }, [diff, diffFilter]);

  async function refreshRuns() {
    setLoadingRuns(true);
    setError("");
    try {
      const res = await fetch("/api/runs");
      if (!res.ok) throw new Error(`Failed to load runs (${res.status})`);
      const data = await res.json();
      setRuns(Array.isArray(data.runs) ? data.runs : []);
      setDir(data.dir || "");
    } catch (e) {
      setError(e?.message || String(e));
    } finally {
      setLoadingRuns(false);
    }
  }

  async function generateDiff() {
    if (selectedIds.length < 1) return;
    setLoadingDiff(true);
    setError("");
    setDiff(null);
    setSingleRun(null);

    // If only 1 run selected, view that run
    if (selectedIds.length === 1) {
      try {
        const res = await fetch("/api/run", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ runId: selectedIds[0] }),
        });
        const data = await res.json();
        if (!res.ok)
          throw new Error(data?.error || `View run failed (${res.status})`);
        setSingleRun({ ...data, runId: selectedIds[0] });
        setShowRunList(false);
      } catch (e) {
        setError(e?.message || String(e));
      } finally {
        setLoadingDiff(false);
      }
      return;
    }

    // Otherwise generate diff
    try {
      const res = await fetch("/api/diff", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ runIds: selectedIds, title }),
      });
      const data = await res.json();
      if (!res.ok)
        throw new Error(data?.error || `Diff failed (${res.status})`);
      setDiff(data);
      setShowRunList(false);
    } catch (e) {
      setError(e?.message || String(e));
    } finally {
      setLoadingDiff(false);
    }
  }

  async function deleteSelectedRuns(ids) {
    const runIds = Array.isArray(ids) ? ids : selectedIds;
    if (runIds.length < 1) return;

    setDeletingRuns(true);
    setError("");
    try {
      const res = await fetch("/api/delete-runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ runIds }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok)
        throw new Error(data?.error || `Delete failed (${res.status})`);

      setSelected((prev) => {
        const next = new Set(prev);
        for (const id of runIds) next.delete(id);
        return next;
      });
      await refreshRuns();
    } catch (e) {
      setError(e?.message || String(e));
    } finally {
      setDeletingRuns(false);
    }
  }

  useEffect(() => {
    refreshRuns();
    const t = setInterval(refreshRuns, 2000);
    return () => clearInterval(t);
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    function handleKey(e) {
      // Ignore if typing in input
      if (e.target.tagName === "INPUT") return;

      if (e.key === "?" || (e.key === "/" && e.shiftKey)) {
        e.preventDefault();
        setShowHelp((v) => !v);
      } else if (e.key === "r" || e.key === "R") {
        e.preventDefault();
        refreshRuns();
      } else if ((e.ctrlKey || e.metaKey) && e.key === "a") {
        e.preventDefault();
        selectAll();
      } else if ((e.ctrlKey || e.metaKey) && e.key === "d") {
        e.preventDefault();
        if (selectedIds.length >= 1) generateDiff();
      } else if (e.key === "Escape") {
        if (showHelp) setShowHelp(false);
        else if (diff || singleRun) {
          setDiff(null);
          setSingleRun(null);
          setShowRunList(true);
        }
      } else if (e.key === "c" && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        clearSelection();
      } else if (e.key === "f" && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        selectFailed();
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedIds, showHelp, diff, singleRun, runs]);

  function toggle(id) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function selectAll() {
    setSelected(new Set(filteredRuns.map((r) => r.id)));
  }

  function clearSelection() {
    setSelected(new Set());
  }

  function selectFailed() {
    setSelected(
      new Set(filteredRuns.filter((r) => r.failCount > 0).map((r) => r.id))
    );
  }

  function sortColumn(col) {
    if (sortBy === col) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(col);
      setSortDir("desc");
    }
  }

  function toggleSuite(suiteName) {
    setCollapsedSuites((prev) => {
      const next = new Set(prev);
      if (next.has(suiteName)) next.delete(suiteName);
      else next.add(suiteName);
      return next;
    });
  }

  return (
    <div className="app">
      <Header
        dir={dir}
        runCount={runs.length}
        selectedCount={selectedIds.length}
        loadingRuns={loadingRuns}
        onRefresh={refreshRuns}
        onShowHelp={() => setShowHelp(true)}
        showRunList={showRunList}
        onToggleRunList={() => setShowRunList(!showRunList)}
      />

      {error ? <div className="error">⚠️ {error}</div> : null}

      {showRunList && (
        <RunList
          runs={filteredRuns}
          selected={selected}
          onToggle={toggle}
          title={title}
          onTitleChange={setTitle}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          onSelectAll={selectAll}
          onSelectFailed={selectFailed}
          onClearSelection={clearSelection}
          onGenerate={generateDiff}
          onDeleteSelected={deleteSelectedRuns}
          deletingRuns={deletingRuns}
          loadingDiff={loadingDiff}
          sortBy={sortBy}
          sortDir={sortDir}
          onSort={sortColumn}
          showActions={!singleRun && !diff}
        />
      )}

      {singleRun && (
        <SingleRunView
          singleRun={singleRun}
          onClose={() => {
            setSingleRun(null);
            setShowRunList(true);
          }}
          diffFilter={diffFilter}
          onFilterChange={setDiffFilter}
          collapsedSuites={collapsedSuites}
          onToggleSuite={toggleSuite}
        />
      )}

      {diff && (
        <DiffView
          diff={diff}
          onClose={() => {
            setDiff(null);
            setShowRunList(true);
          }}
          diffFilter={diffFilter}
          onFilterChange={setDiffFilter}
          filteredDiffSuites={filteredDiffSuites}
          collapsedSuites={collapsedSuites}
          onToggleSuite={toggleSuite}
        />
      )}

      {showHelp && <HelpModal onClose={() => setShowHelp(false)} />}
    </div>
  );
}

export default App;
