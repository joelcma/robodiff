package robodiff

import (
	"context"
	"encoding/xml"
	"os"
	"strings"
)

func ParseRobotXMLBytes(data []byte) (*Robot, error) {
	return ParseRobotXMLBytesContext(context.Background(), data)
}

func ParseRobotXMLFile(path string) (*Robot, error) {
	return ParseRobotXMLFileContext(context.Background(), path)
}

func ParseRobotXMLBytesContext(ctx context.Context, data []byte) (*Robot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resultCh := make(chan *Robot, 1)
	errCh := make(chan error, 1)

	go func() {
		var robot Robot
		if err := xml.Unmarshal(data, &robot); err != nil {
			errCh <- err
			return
		}
		resultCh <- &robot
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case robot := <-resultCh:
		return robot, nil
	}
}

func ParseRobotXMLFileContext(ctx context.Context, path string) (*Robot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseRobotXMLBytesContext(ctx, data)
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
