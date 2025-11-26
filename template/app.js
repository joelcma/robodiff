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
setupHistoryFeature();

// History functionality
function setupHistoryFeature() {
  if (!HISTORY_ENABLED || !HISTORY_DATA) {
    // Hide history-related controls
    document.getElementById("showHistory").style.display = "none";
    document.getElementById("saveToHistoryGroup").style.display = "none";
    return;
  }

  // Show save to history button
  document.getElementById("saveToHistoryGroup").style.display = "flex";

  // Populate column selector dropdown
  const columnSelect = document.getElementById("columnSelect");
  DATA.columns.forEach((col, idx) => {
    const option = document.createElement("option");
    option.value = idx;
    option.textContent = col;
    columnSelect.appendChild(option);
  });

  // Populate tag dropdown
  const tags = HISTORY_DATA.entries
    .map((e) => e.tag)
    .filter((v, i, a) => a.indexOf(v) === i) // unique
    .sort();

  const tagSelect = document.getElementById("historyTagSelect");
  tags.forEach((tag) => {
    const option = document.createElement("option");
    option.value = tag;
    option.textContent = tag;
    tagSelect.appendChild(option);
  });

  // View switcher
  document.getElementById("showComparison").addEventListener("click", () => {
    document.getElementById("comparisonView").style.display = "block";
    document.getElementById("historyView").style.display = "none";
    document.getElementById("comparisonControls").style.display = "flex";
    document.getElementById("comparisonControls2").style.display = "flex";
    document.getElementById("searchBox").parentElement.style.display = "flex";
    document.getElementById("historyControls").style.display = "none";
    document.getElementById("saveToHistoryGroup").style.display = "flex";

    document.getElementById("showComparison").classList.add("active");
    document.getElementById("showHistory").classList.remove("active");
  });

  document.getElementById("showHistory").addEventListener("click", () => {
    document.getElementById("comparisonView").style.display = "none";
    document.getElementById("historyView").style.display = "block";
    document.getElementById("comparisonControls").style.display = "none";
    document.getElementById("comparisonControls2").style.display = "none";
    document.getElementById("searchBox").parentElement.style.display = "none";
    document.getElementById("historyControls").style.display = "flex";
    document.getElementById("saveToHistoryGroup").style.display = "none";

    document.getElementById("showHistory").classList.add("active");
    document.getElementById("showComparison").classList.remove("active");
  });

  // Tag selection
  tagSelect.addEventListener("change", (e) => {
    if (e.target.value) {
      renderHistoryView(e.target.value);
    } else {
      document.getElementById("historyContent").innerHTML =
        '<p class="history-placeholder">Select a tag to view historical trends</p>';
    }
  });

  // Save to history button
  document.getElementById("saveToHistory").addEventListener("click", () => {
    const columnIdx = document.getElementById("columnSelect").value;
    if (columnIdx === "") {
      alert("Please select which test run to save");
      return;
    }

    const tag = document.getElementById("tagInput").value.trim();
    if (!tag) {
      alert("Please enter a tag name");
      return;
    }

    const idx = parseInt(columnIdx);
    const selectedColumn = DATA.columns[idx];

    // Extract only the selected column's results
    const filteredSuites = DATA.suites.map((suite) => ({
      name: suite.name,
      tests: suite.tests.map((test) => ({
        name: test.name,
        results: [test.results[idx]], // Only the selected column
      })),
    }));

    const entry = {
      timestamp: new Date().toISOString(),
      tag: tag,
      title: selectedColumn,
      columns: [selectedColumn],
      suites: filteredSuites,
    };

    // Add to history
    HISTORY_DATA.entries.unshift(entry);

    // Show success message
    alert(
      `Saved "${selectedColumn}" to history with tag: ${tag}\n\nNote: History is saved in the browser's local storage. To persist across sessions, use the --enable-history flag and save the ${HISTORY_FILE} file.`
    );

    // Update dropdown if needed
    if (!tags.includes(tag)) {
      const option = document.createElement("option");
      option.value = tag;
      option.textContent = tag;
      tagSelect.appendChild(option);
    }

    // Clear inputs
    document.getElementById("tagInput").value = "";
    document.getElementById("columnSelect").value = "";

    // Try to save to localStorage
    try {
      localStorage.setItem("robotdiff_history", JSON.stringify(HISTORY_DATA));
    } catch (e) {
      console.warn("Could not save to localStorage:", e);
    }
  });

  // Load from localStorage if available
  try {
    const stored = localStorage.getItem("robotdiff_history");
    if (stored) {
      const storedData = JSON.parse(stored);
      HISTORY_DATA.entries = storedData.entries || [];

      // Refresh tag dropdown
      tagSelect.innerHTML = '<option value="">Select a tag...</option>';
      const updatedTags = HISTORY_DATA.entries
        .map((e) => e.tag)
        .filter((v, i, a) => a.indexOf(v) === i)
        .sort();
      updatedTags.forEach((tag) => {
        const option = document.createElement("option");
        option.value = tag;
        option.textContent = tag;
        tagSelect.appendChild(option);
      });
    }
  } catch (e) {
    console.warn("Could not load from localStorage:", e);
  }
}

function renderHistoryView(tag) {
  const entries = HISTORY_DATA.entries
    .filter((e) => e.tag === tag)
    .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));

  if (entries.length === 0) {
    document.getElementById("historyContent").innerHTML =
      '<p class="history-placeholder">No history found for tag: ' +
      escapeHtml(tag) +
      "</p>";
    return;
  }

  // Calculate statistics for each entry
  const stats = entries.map((entry) => {
    let totalTests = 0;
    let passedTests = 0;
    let failedTests = 0;
    let diffTests = 0;

    entry.suites.forEach((suite) => {
      suite.tests.forEach((test) => {
        totalTests++;
        const status = calculateTestStatus(test.results);
        if (status === "all_passed") passedTests++;
        else if (status === "all_failed") failedTests++;
        else if (status === "diff") diffTests++;
      });
    });

    return {
      timestamp: new Date(entry.timestamp),
      total: totalTests,
      passed: passedTests,
      failed: failedTests,
      diff: diffTests,
      passRate:
        totalTests > 0 ? ((passedTests / totalTests) * 100).toFixed(1) : 0,
    };
  });

  let html = '<div class="history-chart">';
  html += "<h2>Historical Trends for Tag: " + escapeHtml(tag) + "</h2>";

  // Simple text-based chart
  html += '<div class="chart-container">';
  html += '<canvas id="trendChart" width="800" height="300"></canvas>';
  html += "</div>";

  html += '<div class="timeline">';

  stats.forEach((stat, i) => {
    const entry = entries[i];
    html +=
      '<div class="timeline-item" data-timestamp="' + entry.timestamp + '">';
    html +=
      '<div class="timeline-date">' +
      stat.timestamp.toLocaleString() +
      "</div>";
    html += '<div class="timeline-stats">';
    html += '<div class="timeline-stat">Total: ' + stat.total + "</div>";
    html +=
      '<div class="timeline-stat pass">✓ Passed: ' +
      stat.passed +
      " (" +
      stat.passRate +
      "%)</div>";
    html +=
      '<div class="timeline-stat fail">✗ Failed: ' + stat.failed + "</div>";
    if (stat.diff > 0) {
      html +=
        '<div class="timeline-stat" style="color: #f59e0b;">⚠ Differences: ' +
        stat.diff +
        "</div>";
    }
    html += "</div>";
    html += '<div class="timeline-actions">';
    html +=
      '<button class="btn-icon btn-edit" title="Edit tag" data-timestamp="' +
      entry.timestamp +
      '">✏️</button>';
    html +=
      '<button class="btn-icon btn-delete" title="Delete entry" data-timestamp="' +
      entry.timestamp +
      '">🗑️</button>';
    html += "</div>";
    html += "</div>";
  });

  html += "</div>";
  html += "</div>";

  document.getElementById("historyContent").innerHTML = html;

  // Attach event listeners for edit and delete buttons
  document.querySelectorAll(".btn-edit").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      const timestamp = btn.dataset.timestamp;
      showEditModal(timestamp);
    });
  });

  document.querySelectorAll(".btn-delete").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      const timestamp = btn.dataset.timestamp;
      deleteHistoryEntry(timestamp);
    });
  });

  // Draw simple canvas chart
  drawTrendChart(stats.reverse()); // Reverse to show oldest first in chart
}

function drawTrendChart(stats) {
  const canvas = document.getElementById("trendChart");
  if (!canvas) return;

  const ctx = canvas.getContext("2d");
  const width = canvas.width;
  const height = canvas.height;
  const padding = 40;
  const chartWidth = width - padding * 2;
  const chartHeight = height - padding * 2;

  // Clear canvas
  ctx.clearRect(0, 0, width, height);

  if (stats.length === 0) return;

  // Find max value for scaling
  const maxValue = Math.max(...stats.map((s) => s.total));
  if (maxValue === 0) return;

  // Draw axes
  ctx.strokeStyle = "#e2e8f0";
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(padding, padding);
  ctx.lineTo(padding, height - padding);
  ctx.lineTo(width - padding, height - padding);
  ctx.stroke();

  // Draw grid lines
  ctx.strokeStyle = "#f7fafc";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 5; i++) {
    const y = padding + (chartHeight / 5) * i;
    ctx.beginPath();
    ctx.moveTo(padding, y);
    ctx.lineTo(width - padding, y);
    ctx.stroke();
  }

  // Draw pass rate line
  ctx.strokeStyle = "#22c55e";
  ctx.lineWidth = 3;
  ctx.beginPath();

  stats.forEach((stat, i) => {
    const x = padding + (chartWidth / (stats.length - 1 || 1)) * i;
    const y = height - padding - chartHeight * (stat.passed / maxValue);

    if (i === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  });
  ctx.stroke();

  // Draw fail line
  ctx.strokeStyle = "#ef4444";
  ctx.lineWidth = 3;
  ctx.beginPath();

  stats.forEach((stat, i) => {
    const x = padding + (chartWidth / (stats.length - 1 || 1)) * i;
    const y = height - padding - chartHeight * (stat.failed / maxValue);

    if (i === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  });
  ctx.stroke();

  // Draw points
  stats.forEach((stat, i) => {
    const x = padding + (chartWidth / (stats.length - 1 || 1)) * i;

    // Passed point
    const yPass = height - padding - chartHeight * (stat.passed / maxValue);
    ctx.fillStyle = "#22c55e";
    ctx.beginPath();
    ctx.arc(x, yPass, 5, 0, Math.PI * 2);
    ctx.fill();

    // Failed point
    const yFail = height - padding - chartHeight * (stat.failed / maxValue);
    ctx.fillStyle = "#ef4444";
    ctx.beginPath();
    ctx.arc(x, yFail, 5, 0, Math.PI * 2);
    ctx.fill();
  });

  // Draw labels
  ctx.fillStyle = "#64748b";
  ctx.font = '12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto';
  ctx.textAlign = "right";
  ctx.textBaseline = "middle";

  for (let i = 0; i <= 5; i++) {
    const value = Math.round(maxValue - (maxValue / 5) * i);
    const y = padding + (chartHeight / 5) * i;
    ctx.fillText(value.toString(), padding - 10, y);
  }

  // Legend
  ctx.textAlign = "left";
  ctx.fillStyle = "#22c55e";
  ctx.fillRect(width - 150, 20, 20, 3);
  ctx.fillStyle = "#2d3748";
  ctx.fillText("Passed", width - 125, 21);

  ctx.fillStyle = "#ef4444";
  ctx.fillRect(width - 150, 35, 20, 3);
  ctx.fillStyle = "#2d3748";
  ctx.fillText("Failed", width - 125, 36);
}

function showEditModal(timestamp) {
  const entry = HISTORY_DATA.entries.find((e) => e.timestamp === timestamp);
  if (!entry) return;

  const newTag = prompt(
    `Edit tag for entry from ${new Date(timestamp).toLocaleString()}:`,
    entry.tag
  );
  if (newTag && newTag.trim() && newTag !== entry.tag) {
    const oldTag = entry.tag;
    entry.tag = newTag.trim();

    // Save to localStorage
    try {
      localStorage.setItem("robotdiff_history", JSON.stringify(HISTORY_DATA));
    } catch (e) {
      console.warn("Could not save to localStorage:", e);
    }

    // Refresh the tag dropdown
    const tagSelect = document.getElementById("historyTagSelect");
    const tags = HISTORY_DATA.entries
      .map((e) => e.tag)
      .filter((v, i, a) => a.indexOf(v) === i)
      .sort();

    tagSelect.innerHTML = '<option value="">Select a tag...</option>';
    tags.forEach((tag) => {
      const option = document.createElement("option");
      option.value = tag;
      option.textContent = tag;
      tagSelect.appendChild(option);
    });

    // If we were viewing the old tag, switch to the new one
    if (tagSelect.value === oldTag) {
      tagSelect.value = newTag;
      renderHistoryView(newTag);
    } else {
      // Refresh current view
      const currentTag = tagSelect.value;
      if (currentTag) {
        renderHistoryView(currentTag);
      }
    }

    alert(`Tag updated from "${oldTag}" to "${newTag}"`);
  }
}

function deleteHistoryEntry(timestamp) {
  const entry = HISTORY_DATA.entries.find((e) => e.timestamp === timestamp);
  if (!entry) return;

  if (
    !confirm(
      `Delete entry from ${new Date(timestamp).toLocaleString()} with tag "${
        entry.tag
      }"?`
    )
  ) {
    return;
  }

  // Remove entry
  const index = HISTORY_DATA.entries.findIndex(
    (e) => e.timestamp === timestamp
  );
  if (index !== -1) {
    HISTORY_DATA.entries.splice(index, 1);
  }

  // Save to localStorage
  try {
    localStorage.setItem("robotdiff_history", JSON.stringify(HISTORY_DATA));
  } catch (e) {
    console.warn("Could not save to localStorage:", e);
  }

  // Refresh the tag dropdown
  const tagSelect = document.getElementById("historyTagSelect");
  const tags = HISTORY_DATA.entries
    .map((e) => e.tag)
    .filter((v, i, a) => a.indexOf(v) === i)
    .sort();

  tagSelect.innerHTML = '<option value="">Select a tag...</option>';
  tags.forEach((tag) => {
    const option = document.createElement("option");
    option.value = tag;
    option.textContent = tag;
    tagSelect.appendChild(option);
  });

  // Refresh current view
  const currentTag = tagSelect.value;
  if (currentTag) {
    renderHistoryView(currentTag);
  } else {
    document.getElementById("historyContent").innerHTML =
      '<p class="history-placeholder">Select a tag to view historical trends</p>';
  }

  alert("Entry deleted successfully");
}
