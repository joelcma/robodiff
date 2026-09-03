package backend

import (
	"testing"

	rdiff "robot_diff/backend/diff"
)

func TestBuildKeywordTimingDataAggregatesCallsAndTraversesControlFlow(t *testing.T) {
	suite := rdiff.Suite{
		Name: "Root",
		Tests: []rdiff.Test{
			{
				Name: "First",
				Keywords: []rdiff.Keyword{
					{
						Name:   "Do_Thing",
						Owner:  "Library",
						Status: elapsedStatus("2.0", "PASS"),
						Keywords: []rdiff.Keyword{
							{Name: "Nested", Status: elapsedStatus("0.5", "PASS")},
						},
					},
				},
			},
			{
				Name: "Second",
				Keywords: []rdiff.Keyword{
					{Name: "do thing", Owner: "library", Status: elapsedStatus("3.0", "FAIL")},
				},
				Ifs: []rdiff.If{
					{
						Branches: []rdiff.Branch{
							{
								Type: "IF",
								Keywords: []rdiff.Keyword{
									{Name: "Inside branch", Status: elapsedStatus("1.0", "PASS")},
								},
							},
						},
					},
				},
			},
		},
	}

	got := buildKeywordTimingData(&suite)
	if len(got) != 3 {
		t.Fatalf("buildKeywordTimingData() returned %d rows, want 3: %#v", len(got), got)
	}

	aggregated := got[0]
	if aggregated.DisplayName != "Library.Do_Thing" {
		t.Fatalf("first keyword = %q, want %q", aggregated.DisplayName, "Library.Do_Thing")
	}
	if aggregated.CallCount != 2 || aggregated.TotalDurationMs != 5000 {
		t.Fatalf("aggregate calls/time = %d/%d, want 2/5000", aggregated.CallCount, aggregated.TotalDurationMs)
	}
	if aggregated.AverageDurationMs != 2500 || aggregated.MaxDurationMs != 3000 {
		t.Fatalf("aggregate average/max = %d/%d, want 2500/3000", aggregated.AverageDurationMs, aggregated.MaxDurationMs)
	}
	if aggregated.Occurrences[0].TestName != "Root.Second" {
		t.Fatalf("slowest occurrence test = %q, want %q", aggregated.Occurrences[0].TestName, "Root.Second")
	}

	for _, timing := range got {
		if timing.Name == "IF" {
			t.Fatal("synthetic IF node should not be included in keyword timing data")
		}
	}
}

func elapsedStatus(elapsed, status string) rdiff.Status {
	return rdiff.Status{Elapsed: elapsed, Status: status}
}
