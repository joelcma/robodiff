// Test utilities
function runTests() {
  console.log("Running tests...\n");

  testCalculateTestStatus();
  testCalculateSuiteStatus();
  testShouldShowInFailedFilter();

  console.log("\n✅ All tests passed!");
}

function assert(condition, message) {
  if (!condition) {
    throw new Error("Assertion failed: " + message);
  }
}

function assertEquals(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(
      `Assertion failed: ${message}\nExpected: ${expected}\nActual: ${actual}`
    );
  }
}

// Tests for calculateTestStatus
function testCalculateTestStatus() {
  console.log("Testing calculateTestStatus...");

  // All PASS
  assertEquals(
    calculateTestStatus(["PASS", "PASS"]),
    "all_passed",
    "All PASS should return all_passed"
  );

  // All FAIL
  assertEquals(
    calculateTestStatus(["FAIL", "FAIL"]),
    "all_failed",
    "All FAIL should return all_failed"
  );

  // PASS and FAIL (diff has priority)
  assertEquals(
    calculateTestStatus(["PASS", "FAIL"]),
    "diff",
    "PASS and FAIL should return diff"
  );

  assertEquals(
    calculateTestStatus(["FAIL", "PASS"]),
    "diff",
    "FAIL and PASS should return diff"
  );

  // MISSING only
  assertEquals(
    calculateTestStatus(["MISSING", "MISSING"]),
    "missing",
    "All MISSING should return missing"
  );

  // PASS and MISSING (missing, but no diff)
  assertEquals(
    calculateTestStatus(["PASS", "MISSING"]),
    "missing",
    "PASS and MISSING should return missing"
  );

  // FAIL and MISSING (missing, but no diff)
  assertEquals(
    calculateTestStatus(["FAIL", "MISSING"]),
    "missing",
    "FAIL and MISSING should return missing"
  );

  // PASS, FAIL, and MISSING (diff has priority)
  assertEquals(
    calculateTestStatus(["PASS", "FAIL", "MISSING"]),
    "diff",
    "PASS, FAIL, and MISSING should return diff (conflict has priority)"
  );

  // Three PASSes
  assertEquals(
    calculateTestStatus(["PASS", "PASS", "PASS"]),
    "all_passed",
    "Three PASS should return all_passed"
  );

  // Two PASS, one FAIL
  assertEquals(
    calculateTestStatus(["PASS", "PASS", "FAIL"]),
    "diff",
    "Two PASS and one FAIL should return diff"
  );

  console.log("  ✓ calculateTestStatus tests passed");
}

// Tests for calculateSuiteStatus
function testCalculateSuiteStatus() {
  console.log("Testing calculateSuiteStatus...");

  // All tests passed
  const allPassedSuite = {
    name: "Suite 1",
    tests: [
      { name: "Test 1", results: ["PASS", "PASS"] },
      { name: "Test 2", results: ["PASS", "PASS"] },
    ],
  };
  assertEquals(
    calculateSuiteStatus(allPassedSuite),
    "all_passed",
    "Suite with all PASS tests should return all_passed"
  );

  // All tests failed
  const allFailedSuite = {
    name: "Suite 2",
    tests: [
      { name: "Test 1", results: ["FAIL", "FAIL"] },
      { name: "Test 2", results: ["FAIL", "FAIL"] },
    ],
  };
  assertEquals(
    calculateSuiteStatus(allFailedSuite),
    "all_failed",
    "Suite with all FAIL tests should return all_failed"
  );

  // Suite with differences
  const diffSuite = {
    name: "Suite 3",
    tests: [
      { name: "Test 1", results: ["PASS", "FAIL"] },
      { name: "Test 2", results: ["PASS", "PASS"] },
    ],
  };
  assertEquals(
    calculateSuiteStatus(diffSuite),
    "diff",
    "Suite with diff test should return diff"
  );

  // Suite with missing
  const missingSuite = {
    name: "Suite 4",
    tests: [
      { name: "Test 1", results: ["PASS", "MISSING"] },
      { name: "Test 2", results: ["PASS", "PASS"] },
    ],
  };
  assertEquals(
    calculateSuiteStatus(missingSuite),
    "diff",
    "Suite with missing test should return diff"
  );

  // Mixed suite
  const mixedSuite = {
    name: "Suite 5",
    tests: [
      { name: "Test 1", results: ["PASS", "PASS"] },
      { name: "Test 2", results: ["FAIL", "FAIL"] },
    ],
  };
  assertEquals(
    calculateSuiteStatus(mixedSuite),
    "all_failed",
    "Suite with PASS and FAIL tests (no diff within tests) should return all_failed"
  );

  console.log("  ✓ calculateSuiteStatus tests passed");
}

// Tests for shouldShowInFailedFilter
function testShouldShowInFailedFilter() {
  console.log("Testing shouldShowInFailedFilter...");

  // All PASS - should NOT show
  assertEquals(
    shouldShowInFailedFilter(["PASS", "PASS"]),
    false,
    "All PASS should NOT show in Failed Only"
  );

  // All FAIL - should show
  assertEquals(
    shouldShowInFailedFilter(["FAIL", "FAIL"]),
    true,
    "All FAIL should show in Failed Only"
  );

  // PASS and FAIL (diff) - should show
  assertEquals(
    shouldShowInFailedFilter(["PASS", "FAIL"]),
    true,
    "PASS and FAIL (diff) should show in Failed Only"
  );

  // FAIL and PASS (diff) - should show
  assertEquals(
    shouldShowInFailedFilter(["FAIL", "PASS"]),
    true,
    "FAIL and PASS (diff) should show in Failed Only"
  );

  // PASS and MISSING - should NOT show
  assertEquals(
    shouldShowInFailedFilter(["PASS", "MISSING"]),
    false,
    "PASS and MISSING should NOT show in Failed Only (no failure)"
  );

  // FAIL and MISSING - should show
  assertEquals(
    shouldShowInFailedFilter(["FAIL", "MISSING"]),
    true,
    "FAIL and MISSING should show in Failed Only (has failure)"
  );

  // MISSING only - should NOT show
  assertEquals(
    shouldShowInFailedFilter(["MISSING", "MISSING"]),
    false,
    "All MISSING should NOT show in Failed Only (no failure)"
  );

  // PASS, FAIL, MISSING - should show (has failure)
  assertEquals(
    shouldShowInFailedFilter(["PASS", "FAIL", "MISSING"]),
    true,
    "PASS, FAIL, MISSING should show in Failed Only (has failure)"
  );

  // Three PASSes - should NOT show
  assertEquals(
    shouldShowInFailedFilter(["PASS", "PASS", "PASS"]),
    false,
    "Three PASS should NOT show in Failed Only"
  );

  // Two PASS, one MISSING - should NOT show
  assertEquals(
    shouldShowInFailedFilter(["PASS", "PASS", "MISSING"]),
    false,
    "Two PASS and one MISSING should NOT show in Failed Only (no failure)"
  );

  // Two FAIL, one MISSING - should show
  assertEquals(
    shouldShowInFailedFilter(["FAIL", "FAIL", "MISSING"]),
    true,
    "Two FAIL and one MISSING should show in Failed Only (has failure)"
  );

  console.log("  ✓ shouldShowInFailedFilter tests passed");
}

// Helper function to determine if test should show in "Failed Only" filter
function shouldShowInFailedFilter(results) {
  // Show if any result is FAIL (actual failure, not just missing)
  return results.some((r) => r === "FAIL");
}

// Run tests if in Node.js environment
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    runTests,
    calculateTestStatus,
    calculateSuiteStatus,
    shouldShowInFailedFilter,
  };
}
