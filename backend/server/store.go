package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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
	go s.scanLoop()
}

func (s *RunStore) scanLoop() {
	s.scanOnce()
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for range t.C {
		s.scanOnce()
	}
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
		if !ok {
			return nil, nil, nil, fmt.Errorf("%w: %s", errRunNotFound, id)
		}
		if e.robot == nil {
			robot, err := robotdiff.ParseRobotXMLFile(e.abs)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("parse run %s: %w", e.abs, err)
			}
			e.robot = robot
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
				runSize := runFolderSize(filepath.Dir(abs))
				if existing.info.ModTime.Equal(fi.ModTime()) && existing.info.Size == runSize {
					updated[id] = existing
					continue
				}
			}

			runSize := runFolderSize(filepath.Dir(abs))

			pass, fail, total, okStats, err := readRobotStatistics(abs)
			if err != nil {
				continue
			}
			if !okStats {
				robot, err := robotdiff.ParseRobotXMLFile(abs)
				if err != nil {
					continue
				}
				pass, fail, total = robotdiff.CountTests(&robot.Suite)
			}

			updated[id] = &runEntry{
				abs: abs,
				info: RunInfo{
					ID:        id,
					Name:      runName,
					RelPath:   filepath.ToSlash(rel),
					ModTime:   fi.ModTime(),
					Size:      runSize,
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

func runFolderSize(dir string) int64 {
	files := []string{"output.xml", "log.html", "report.html"}
	var total int64
	for _, name := range files {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.IsDir() {
			continue
		}
		total += st.Size()
	}
	return total
}

type robotStat struct {
	Pass int    `xml:"pass,attr"`
	Fail int    `xml:"fail,attr"`
	Skip int    `xml:"skip,attr"`
	Name string `xml:",chardata"`
}

func readRobotStatistics(path string) (pass, fail, total int, ok bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, 0, false, err
	}

	// Fast path: read only the tail where <statistics> usually lives.
	if info.Size() > 0 {
		const maxTailBytes = 4 * 1024 * 1024
		readSize := int64(maxTailBytes)
		if info.Size() < readSize {
			readSize = info.Size()
		}
		f, err := os.Open(path)
		if err != nil {
			return 0, 0, 0, false, err
		}
		buf := make([]byte, readSize)
		_, _ = f.ReadAt(buf, info.Size()-readSize)
		_ = f.Close()

		if idx := bytes.LastIndex(buf, []byte("<statistics")); idx != -1 {
			pass, fail, total, ok, err = scanStatisticsBytes(buf[idx:])
			if err == nil && ok {
				return pass, fail, total, ok, nil
			}
		}
	}

	// Fallback: stream entire file if tail scan couldn't find statistics.
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, false, err
	}
	defer f.Close()
	return scanStatisticsStream(xml.NewDecoder(f))
}

func scanStatisticsBytes(b []byte) (pass, fail, total int, ok bool, err error) {
	return scanStatisticsStream(xml.NewDecoder(bytes.NewReader(b)))
}

func scanStatisticsStream(dec *xml.Decoder) (pass, fail, total int, ok bool, err error) {
	insideStats := false
	var fallback *robotStat

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, 0, 0, false, err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "statistics":
				insideStats = true
			case "stat":
				if !insideStats {
					continue
				}
				var st robotStat
				if err := dec.DecodeElement(&st, &se); err != nil {
					return 0, 0, 0, false, err
				}
				name := strings.TrimSpace(st.Name)
				if strings.EqualFold(name, "All Tests") {
					return st.Pass, st.Fail, st.Pass + st.Fail + st.Skip, true, nil
				}
				if fallback == nil {
					fallback = &st
				}
			}
		case xml.EndElement:
			if insideStats && se.Name.Local == "statistics" {
				if fallback != nil {
					return fallback.Pass, fallback.Fail, fallback.Pass + fallback.Fail + fallback.Skip, true, nil
				}
				return 0, 0, 0, false, nil
			}
		}
	}
	if fallback != nil {
		return fallback.Pass, fallback.Fail, fallback.Pass + fallback.Fail + fallback.Skip, true, nil
	}
	return 0, 0, 0, false, nil
}

func (s *RunStore) GetTestDetails(runID, testName string) (*robotdiff.Test, error) {
	s.mu.Lock()
	entry, ok := s.runs[runID]
	if !ok {
		s.mu.Unlock()
		return nil, errRunNotFound
	}
	if entry.robot == nil {
		robot, err := robotdiff.ParseRobotXMLFile(entry.abs)
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		entry.robot = robot
	}
	robot := entry.robot
	s.mu.Unlock()

	// Search for the test in the cached robot data
	test := findTestInSuite(&robot.Suite, testName)
	if test != nil {
		return test, nil
	}

	return nil, fmt.Errorf("test %q not found in run", testName)
}

func (s *RunStore) DeleteRuns(ids []string) (deleted int, err error) {
	ids = uniqueNonEmptyStrings(ids)
	if len(ids) == 0 {
		return 0, nil
	}

	rootAbs, err := filepath.Abs(s.dir)
	if err != nil {
		return 0, fmt.Errorf("resolve root dir: %w", err)
	}
	rootReal := rootAbs
	if r, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootReal = r
	}

	// Copy the run directories while holding the lock; delete outside the lock.
	runDirs := make([]string, 0, len(ids))
	s.mu.RLock()
	for _, id := range ids {
		e := s.runs[id]
		if e == nil {
			continue
		}
		runDirs = append(runDirs, filepath.Dir(e.abs))
	}
	s.mu.RUnlock()

	for _, dir := range runDirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return deleted, fmt.Errorf("resolve run dir: %w", err)
		}
		real := abs
		if r, err := filepath.EvalSymlinks(abs); err == nil {
			real = r
		}

		if !isSubpath(rootReal, real) {
			return deleted, fmt.Errorf("refusing to delete outside runs root: %s", real)
		}
		if samePath(rootReal, real) {
			return deleted, fmt.Errorf("refusing to delete runs root: %s", real)
		}

		st, err := os.Stat(real)
		if err != nil {
			// Already gone; treat as no-op.
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return deleted, fmt.Errorf("stat run dir: %w", err)
		}
		if !st.IsDir() {
			return deleted, fmt.Errorf("refusing to delete non-directory: %s", real)
		}

		if err := os.RemoveAll(real); err != nil {
			return deleted, fmt.Errorf("delete run dir: %w", err)
		}
		deleted++
	}

	return deleted, nil
}

func uniqueNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isSubpath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	sep := string(filepath.Separator)
	return !strings.HasPrefix(rel, ".."+sep) && rel != ".."
}

func samePath(a, b string) bool {
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if ca == cb {
		return true
	}
	// Best-effort case-insensitive comparison for macOS default FS behavior.
	return strings.EqualFold(ca, cb)
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
