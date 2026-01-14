package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	robotdiff "robot_diff/backend/diff"
)

var errRunNotFound = errors.New("run not found")

type RunInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	RelPath   string    `json:"relPath"`
	ModTime   time.Time `json:"modTime"`
	Size      int64     `json:"size"`
	TestCount int       `json:"testCount"`
	PassCount int       `json:"passCount"`
	FailCount int       `json:"failCount"`
}

type runEntry struct {
	info  RunInfo
	abs   string
	robot *robotdiff.Robot
}

type RunStore struct {
	dir      string
	interval time.Duration

	mu   sync.RWMutex
	runs map[string]*runEntry
}

func NewRunStore(dir string, interval time.Duration) *RunStore {
	return &RunStore{
		dir:      dir,
		interval: interval,
		runs:     make(map[string]*runEntry, 128),
	}
}

func (s *RunStore) Dir() string             { return s.dir }
func (s *RunStore) Interval() time.Duration { return s.interval }

func (s *RunStore) Start() {
	s.scanOnce()
	go func() {
		t := time.NewTicker(s.interval)
		defer t.Stop()
		for range t.C {
			s.scanOnce()
		}
	}()
}

func (s *RunStore) ListRuns() []RunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]RunInfo, 0, len(s.runs))
	for _, e := range s.runs {
		infos = append(infos, e.info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ModTime.After(infos[j].ModTime) })
	return infos
}

func (s *RunStore) GetRuns(ids []string) (columns []string, inputFiles []string, robots []*robotdiff.Robot, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	columns = make([]string, 0, len(ids))
	inputFiles = make([]string, 0, len(ids))
	robots = make([]*robotdiff.Robot, 0, len(ids))

	for _, id := range ids {
		e, ok := s.runs[id]
		if !ok || e.robot == nil {
			return nil, nil, nil, fmt.Errorf("%w: %s", errRunNotFound, id)
		}
		columns = append(columns, e.info.Name)
		inputFiles = append(inputFiles, e.abs)
		robots = append(robots, e.robot)
	}
	return columns, inputFiles, robots, nil
}

func (s *RunStore) scanOnce() {
	// Build a fresh map each scan so deleted runs disappear.
	updated := make(map[string]*runEntry, 128)

	s.mu.RLock()
	prev := make(map[string]*runEntry, len(s.runs))
	for k, v := range s.runs {
		prev[k] = v
	}
	s.mu.RUnlock()

	const maxDepth = 3 // allow nested layouts like root/run/output.xml or root/env/run/output.xml

	var scanDir func(absDir string, depth int)
	scanDir = func(absDir string, depth int) {
		if depth > maxDepth {
			return
		}

		entries, err := os.ReadDir(absDir)
		if err != nil {
			return
		}

		for _, ent := range entries {
			name := ent.Name()
			absPath := filepath.Join(absDir, name)

			isDir := ent.IsDir()
			if !isDir && (ent.Type()&fs.ModeSymlink) != 0 {
				// Follow symlinked directories (common when results are linked in).
				if st, err := os.Stat(absPath); err == nil && st.IsDir() {
					isDir = true
				}
			}
			if isDir {
				scanDir(absPath, depth+1)
				continue
			}

			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".xml") {
				continue
			}

			// Heuristic: prefer Robot's canonical output file.
			// This avoids trying to parse every XML file (xunit.xml, junit.xml, etc.)
			// which is common in result folders.
			if lower != "output.xml" {
				continue
			}

			fi, err := os.Stat(absPath)
			if err != nil {
				continue
			}

			abs, err := filepath.Abs(absPath)
			if err != nil {
				continue
			}

			rel, err := filepath.Rel(s.dir, abs)
			if err != nil {
				rel = name
			}

			id := stableID(abs)

			// Use directory name for output.xml so multiple runs don't all show as "output".
			runName := filepath.Base(filepath.Dir(abs))
			if runName == "" || runName == string(filepath.Separator) {
				runName = strings.TrimSuffix(name, filepath.Ext(name))
			}

			if existing, ok := prev[id]; ok && existing != nil {
				if existing.info.ModTime.Equal(fi.ModTime()) && existing.info.Size == fi.Size() {
					updated[id] = existing
					continue
				}
			}

			robot, err := robotdiff.ParseRobotXMLFile(abs)
			if err != nil {
				continue
			}
			pass, fail, total := robotdiff.CountTests(&robot.Suite)
			if robot.Statistics != nil {
				if p, f, sk, ok := robot.Statistics.Total.AllTests(); ok {
					pass, fail, total = p, f, p+f+sk
				}
			}

			updated[id] = &runEntry{
				abs:   abs,
				robot: robot,
				info: RunInfo{
					ID:        id,
					Name:      runName,
					RelPath:   filepath.ToSlash(rel),
					ModTime:   fi.ModTime(),
					Size:      fi.Size(),
					TestCount: total,
					PassCount: pass,
					FailCount: fail,
				},
			}
		}
	}

	scanDir(s.dir, 0)

	s.mu.Lock()
	s.runs = updated
	s.mu.Unlock()
}

func stableID(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (s *RunStore) GetTestDetails(runID, testName string) (*robotdiff.Test, error) {
	s.mu.RLock()
	entry, ok := s.runs[runID]
	s.mu.RUnlock()

	if !ok {
		return nil, errRunNotFound
	}

	// Search for the test in the cached robot data
	test := findTestInSuite(&entry.robot.Suite, testName)
	if test != nil {
		return test, nil
	}

	return nil, fmt.Errorf("test %q not found in run", testName)
}

func findTestInSuite(suite *robotdiff.Suite, testName string) *robotdiff.Test {
	// Check tests in current suite
	for i := range suite.Tests {
		if suite.Tests[i].Name == testName {
			return &suite.Tests[i]
		}
	}

	// Recursively check sub-suites
	for i := range suite.Suites {
		if test := findTestInSuite(&suite.Suites[i], testName); test != nil {
			return test
		}
	}

	return nil
}
