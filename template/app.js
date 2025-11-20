// Status calculation utilities
function calculateTestStatus(results) {
  const hasPass = results.some((r) => r === "PASS");
  const hasFail = results.some((r) => r === "FAIL");
  const hasMissing = results.some((r) => r === "MISSING");
  const allPass = results.every((r) => r === "PASS");
  const allFail = results.every((r) => r === "FAIL");

  // Priority: diff (PASS+FAIL conflict) > missing > all_passed/all_failed
  if (hasPass && hasFail) {
    return "diff";
  } else if (hasMissing) {
    return "missing";
  } else if (allPass) {
    return "all_passed";
  } else {
    return "all_failed";
  }
}

function calculateSuiteStatus(suite) {
  let allPassed = true;
  let anyFailed = false;
  let anyDiff = false;

  suite.tests.forEach((test) => {
    const status = calculateTestStatus(test.results);
    if (status !== "all_passed") allPassed = false;
    if (status === "all_failed") anyFailed = true;
    if (status === "diff" || status === "missing") anyDiff = true;
  });

  return anyDiff ? "diff" : allPassed ? "all_passed" : "all_failed";
}

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// Render functions
function renderReport() {
  document.getElementById("title").textContent = DATA.title;
  renderTableHeader();
  renderTableBody();
  calculateStats();
}

function renderTableHeader() {
  const header = document.getElementById("tableHeader");
  let headerHTML = '<tr><th class="col_name">Test Name</th>';
  DATA.columns.forEach((col) => {
    headerHTML += '<th class="col_status">' + escapeHtml(col) + "</th>";
  });
  headerHTML += "</tr>";
  header.innerHTML = headerHTML;
}

function renderTableBody() {
  const tbody = document.getElementById("tableBody");
  let bodyHTML = "";

  DATA.suites.forEach((suite) => {
    const suiteName = suite.name;
    const suiteDisplayName = suiteName.split(".").pop();
    const suiteStatus = calculateSuiteStatus(suite);

    // Suite row
    bodyHTML +=
      '<tr class="suite-row" data-suite-name="' + escapeHtml(suiteName) + '">';
    bodyHTML +=
      '<td class="col_name ' +
      suiteStatus +
      '" title="' +
      escapeHtml(suiteName) +
      '">';
    bodyHTML +=
      '<span class="suite-name">' + escapeHtml(suiteDisplayName) + "</span>";
    bodyHTML += "</td>";

    // Suite status cells (aggregated)
    DATA.columns.forEach((_, i) => {
      const allPass = suite.tests.every((t) => t.results[i] === "PASS");
      const allFail = suite.tests.every((t) => t.results[i] === "FAIL");
      const hasMissing = suite.tests.some((t) => t.results[i] === "MISSING");
      const status = hasMissing
        ? "not_available"
        : allPass
        ? "pass"
        : allFail
        ? "fail"
        : "diff";
      const text = hasMissing
        ? "N/A"
        : allPass
        ? "PASS"
        : allFail
        ? "FAIL"
        : "MIXED";
      bodyHTML += '<td class="col_status ' + status + '">' + text + "</td>";
    });
    bodyHTML += "</tr>";

    // Test rows
    suite.tests.forEach((test) => {
      const testFullName = suiteName + "." + test.name;
      const testStatus = calculateTestStatus(test.results);

      bodyHTML +=
        '<tr class="test-row" data-full-name="' +
        escapeHtml(testFullName) +
        '" data-suite="' +
        escapeHtml(suiteName) +
        '">';
      bodyHTML +=
        '<td class="col_name ' +
        testStatus +
        '" title="' +
        escapeHtml(testFullName) +
        '">';
      bodyHTML +=
        '<span class="suite-name">' + escapeHtml(test.name) + "</span>";
      bodyHTML += "</td>";

      test.results.forEach((result) => {
        const status =
          result === "MISSING"
            ? "not_available"
            : result === "PASS"
            ? "pass"
            : "fail";
        const text = result === "MISSING" ? "N/A" : result;
        bodyHTML += '<td class="col_status ' + status + '">' + text + "</td>";
      });
      bodyHTML += "</tr>";
    });
  });

  tbody.innerHTML = bodyHTML;
}

function calculateStats() {
  const testRows = document.querySelectorAll(".test-row");
  let allPassed = 0,
    allFailed = 0,
    diff = 0,
    missing = 0,
    total = 0;

  testRows.forEach((row) => {
    total++;
    const nameCell = row.querySelector(".col_name");
    if (nameCell.classList.contains("all_passed")) allPassed++;
    else if (nameCell.classList.contains("all_failed")) allFailed++;
    else if (nameCell.classList.contains("diff")) diff++;
    else if (nameCell.classList.contains("missing")) missing++;
  });

  const statsHTML = [
    '<div class="stat">Total: ' + total + "</div>",
    allPassed > 0 ? '<div class="stat">✓ Passed: ' + allPassed + "</div>" : "",
    diff > 0 ? '<div class="stat">⚠ Differences: ' + diff + "</div>" : "",
    missing > 0 ? '<div class="stat">⊘ Missing: ' + missing + "</div>" : "",
    allFailed > 0 ? '<div class="stat">✗ Failed: ' + allFailed + "</div>" : "",
  ]
    .filter((s) => s)
    .join("");

  document.getElementById("stats").innerHTML = statsHTML;
}

// Collapsible suite functionality
function setupCollapsible() {
  document.querySelectorAll(".suite-row").forEach((suite) => {
    suite.addEventListener("click", () => {
      const suiteName = suite.dataset.suiteName;
      const isCollapsed = suite.classList.toggle("collapsed");

      document.querySelectorAll(".test-row").forEach((test) => {
        if (test.dataset.suite === suiteName) {
          test.classList.toggle("hidden", isCollapsed);
        }
      });
    });
  });
}

document.getElementById("collapseAll").addEventListener("click", () => {
  document
    .querySelectorAll(".suite-row")
    .forEach((s) => s.classList.add("collapsed"));
  document
    .querySelectorAll(".test-row")
    .forEach((t) => t.classList.add("hidden"));
});

document.getElementById("expandAll").addEventListener("click", () => {
  document
    .querySelectorAll(".suite-row")
    .forEach((s) => s.classList.remove("collapsed"));
  document
    .querySelectorAll(".test-row")
    .forEach((t) => t.classList.remove("hidden"));
});

// Filter functionality
let currentFilter = "all";

function applyFilter(filter) {
  currentFilter = filter;

  // Update button states
  document
    .querySelectorAll("#showAll, #showDiff, #showFailed")
    .forEach((btn) => {
      btn.classList.remove("active");
    });

  if (filter === "all")
    document.getElementById("showAll").classList.add("active");
  else if (filter === "diff")
    document.getElementById("showDiff").classList.add("active");
  else if (filter === "failed")
    document.getElementById("showFailed").classList.add("active");

  // Filter tests
  const testRows = document.querySelectorAll(".test-row");
  const visibleSuites = new Set();

  testRows.forEach((row) => {
    const nameCell = row.querySelector(".col_name");
    let visible = false;

    if (filter === "all") {
      visible = true;
    } else if (filter === "diff") {
      // Show tests with actual differences (PASS vs FAIL) or missing
      visible =
        nameCell.classList.contains("diff") ||
        nameCell.classList.contains("missing");
    } else if (filter === "failed") {
      // Show only tests where at least one result is FAIL (not just missing)
      const hasFail = Array.from(row.querySelectorAll(".col_status")).some(
        (cell) => cell.classList.contains("fail")
      );
      visible = hasFail;
    }

    row.style.display = visible ? "" : "none";
    if (visible) {
      visibleSuites.add(row.dataset.suite);
    }
  });

  // Show/hide suites based on visible tests
  document.querySelectorAll(".suite-row").forEach((suite) => {
    const suiteName = suite.dataset.suiteName;
    suite.style.display =
      filter === "all" || visibleSuites.has(suiteName) ? "" : "none";
  });
}

document
  .getElementById("showAll")
  .addEventListener("click", () => applyFilter("all"));
document
  .getElementById("showDiff")
  .addEventListener("click", () => applyFilter("diff"));
document
  .getElementById("showFailed")
  .addEventListener("click", () => applyFilter("failed"));

// Search functionality
const searchBox = document.getElementById("searchBox");
let searchTimeout;

searchBox.addEventListener("input", (e) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    const query = e.target.value.toLowerCase().trim();

    if (query === "") {
      applyFilter(currentFilter);
      return;
    }

    const testRows = document.querySelectorAll(".test-row");
    const visibleSuites = new Set();

    testRows.forEach((row) => {
      const fullName = row.dataset.fullName.toLowerCase();
      const displayName = row
        .querySelector(".col_name")
        .textContent.toLowerCase();
      const visible = fullName.includes(query) || displayName.includes(query);

      row.style.display = visible ? "" : "none";
      if (visible) {
        visibleSuites.add(row.dataset.suite);
      }
    });

    // Show/hide suites
    document.querySelectorAll(".suite-row").forEach((suite) => {
      const suiteName = suite.dataset.suiteName;
      suite.style.display = visibleSuites.has(suiteName) ? "" : "none";
    });
  }, 300);
});

// Initialize
renderReport();
setupCollapsible();
