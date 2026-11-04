package store

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	robodiff "robot_diff/backend/diff"
)

// maxOutcomeMessageRunes caps cached status messages. History only needs a
// preview plus enough text to tell whether two failures look alike.
const maxOutcomeMessageRunes = 1000

// TestOutcome is a test's final status in one run. ID uses the agent API's
// positional scheme (s0-s1-t2), so it can be followed into that run.
type TestOutcome struct {
	ID      string
	Name    string
	Status  string
	Message string
	Start   string
}

type outcomeLoad struct {
	modTime time.Time
	size    int64
	done    chan struct{}
	tests   []TestOutcome
	err     error
}

// TestOutcomes returns the final status of every test in a run. It keeps only
// suite/test names and test statuses, so reading many earlier runs does not
// retain their keyword trees (as GetRuns would) or hold the store lock while
// parsing. Results are cached until the file changes. If ctx ends first the
// read keeps going in the background so a retry can use it.
func (s *RunStore) TestOutcomes(ctx context.Context, id string) ([]TestOutcome, error) {
	s.mu.RLock()
	entry := s.runs[id]
	var abs string
	if entry != nil {
		abs = entry.abs
	}
	s.mu.RUnlock()
	if entry == nil {
		return nil, fmt.Errorf("%w: %s", errRunNotFound, id)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat run %s: %w", abs, err)
	}

	s.outcomeMu.Lock()
	if s.outcomes == nil {
		s.outcomes = make(map[string]*outcomeLoad)
		workers := runtime.GOMAXPROCS(0) / 2
		if workers < 1 {
			workers = 1
		}
		s.outcomeSlots = make(chan struct{}, workers)
	}
	load := s.outcomes[id]
	if load == nil || !load.modTime.Equal(fi.ModTime()) || load.size != fi.Size() || load.failed() {
		s.pruneOutcomesLocked()
		load = &outcomeLoad{modTime: fi.ModTime(), size: fi.Size(), done: make(chan struct{})}
		s.outcomes[id] = load
		slots := s.outcomeSlots
		go func() {
			defer close(load.done)
			slots <- struct{}{}
			defer func() { <-slots }()
			load.tests, load.err = readTestOutcomes(abs)
		}()
	}
	s.outcomeMu.Unlock()

	select {
	case <-load.done:
		return load.tests, load.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *outcomeLoad) failed() bool {
	select {
	case <-l.done:
		return l.err != nil
	default:
		return false
	}
}

// pruneOutcomesLocked drops cached outcomes for runs that are gone.
func (s *RunStore) pruneOutcomesLocked() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id := range s.outcomes {
		if _, ok := s.runs[id]; !ok {
			delete(s.outcomes, id)
		}
	}
}

func readTestOutcomes(path string) ([]TestOutcome, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tests, err := scanTestOutcomes(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("parse run %s: %w", path, err)
	}
	return tests, nil
}

// scanTestOutcomes walks only <suite>, <test> and test <status> elements and
// skips keyword bodies. IDs and dotted names are built exactly like the agent
// index builds them, so both sides of a comparison agree.
func scanTestOutcomes(r io.Reader) ([]TestOutcome, error) {
	type frame struct {
		element, id, name string
		suites, tests     int
		test              int // index into tests, for <test> frames
	}
	dec := xml.NewDecoder(r)
	var stack []frame
	tests := []TestOutcome{}
	rootSeen := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return tests, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			parent := ""
			if len(stack) > 0 {
				parent = stack[len(stack)-1].element
			}
			switch {
			case t.Name.Local == "robot" && len(stack) == 0:
				stack = append(stack, frame{element: "robot"})
			case t.Name.Local == "suite" && (parent == "robot" || parent == "suite"):
				f := frame{element: "suite", name: xmlAttr(t, "name")}
				if parent == "robot" {
					if rootSeen {
						if err := dec.Skip(); err != nil {
							return nil, err
						}
						continue
					}
					rootSeen = true
					f.id = "s0"
				} else {
					p := &stack[len(stack)-1]
					f.id = fmt.Sprintf("%s-s%d", p.id, p.suites)
					f.name = p.name + "." + f.name
					p.suites++
				}
				stack = append(stack, f)
			case t.Name.Local == "test" && parent == "suite":
				p := &stack[len(stack)-1]
				tests = append(tests, TestOutcome{
					ID:   fmt.Sprintf("%s-t%d", p.id, p.tests),
					Name: p.name + "." + xmlAttr(t, "name"),
				})
				p.tests++
				stack = append(stack, frame{element: "test", test: len(tests) - 1})
			case t.Name.Local == "status" && parent == "test":
				var st robodiff.Status
				if err := dec.DecodeElement(&st, &t); err != nil {
					return nil, err
				}
				out := &tests[stack[len(stack)-1].test]
				out.Status = st.Status
				out.Message = truncateRunes(st.Message, maxOutcomeMessageRunes)
				out.Start = st.StartTime
			default:
				if err := dec.Skip(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

// MessageSignature is a comparable form of a status message: the cached
// preview with whitespace normalized. Apply it to both sides of a comparison.
func MessageSignature(msg string) string {
	return strings.Join(strings.Fields(truncateRunes(msg, maxOutcomeMessageRunes)), " ")
}

func xmlAttr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
