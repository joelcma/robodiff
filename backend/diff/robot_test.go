package robodiff

import (
	"strings"
	"testing"
)

func TestParseRobotXMLPreservesStatusMessage(t *testing.T) {
	data := []byte(`<robot>
  <suite name="Frontend">
    <test name="Skipped after setup failure">
      <status status="FAIL" start="2026-09-02T13:24:36.367668" elapsed="0.000360">Parent suite setup failed:
WebDriverException: failed to change window state</status>
    </test>
    <status status="FAIL" start="2026-09-02T13:24:36.000000" elapsed="0.500000">Suite setup failed</status>
  </suite>
</robot>`)

	robot, err := ParseRobotXMLBytes(data)
	if err != nil {
		t.Fatalf("ParseRobotXMLBytes() error = %v", err)
	}

	if got := robot.Suite.Tests[0].Status.Message; !strings.Contains(got, "Parent suite setup failed") {
		t.Fatalf("test status message = %q, want parent setup failure", got)
	}
	if got := robot.Suite.Status.Message; got != "Suite setup failed" {
		t.Fatalf("suite status message = %q, want %q", got, "Suite setup failed")
	}
}

func TestParseRobotXMLPreservesKeywordOwner(t *testing.T) {
	data := []byte(`<robot>
  <suite name="Suite">
    <test name="Test">
      <kw name="Log" owner="BuiltIn">
        <status status="PASS" elapsed="0.001" />
      </kw>
      <status status="PASS" elapsed="0.001" />
    </test>
    <status status="PASS" elapsed="0.001" />
  </suite>
</robot>`)

	robot, err := ParseRobotXMLBytes(data)
	if err != nil {
		t.Fatalf("ParseRobotXMLBytes() error = %v", err)
	}
	if got := robot.Suite.Tests[0].Keywords[0].Owner; got != "BuiltIn" {
		t.Fatalf("keyword owner = %q, want %q", got, "BuiltIn")
	}
}
