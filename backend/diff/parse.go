package robotdiff

import (
	"encoding/xml"
	"os"
	"strings"
)

func ParseRobotXMLBytes(data []byte) (*Robot, error) {
	var robot Robot
	if err := xml.Unmarshal(data, &robot); err != nil {
		return nil, err
	}
	return &robot, nil
}

func ParseRobotXMLFile(path string) (*Robot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseRobotXMLBytes(data)
}

func CountTests(suite *Suite) (pass int, fail int, total int) {
	for i := range suite.Suites {
		p, f, t := CountTests(&suite.Suites[i])
		pass += p
		fail += f
		total += t
	}

	for _, test := range suite.Tests {
		total++
		s := strings.ToUpper(test.Status.Status)
		if s == "PASS" {
			pass++
		} else if s == "FAIL" {
			fail++
		}
	}

	return pass, fail, total
}
