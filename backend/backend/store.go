package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"robot_diff/backend/robotdiff"
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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	_ = filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".xml") {
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return nil
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(s.dir, abs)
		if err != nil {
			rel = d.Name()
		}

		id := stableID(abs)
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

		s.mu.RLock()
		existing, ok := s.runs[id]
		s.mu.RUnlock()
		if ok && existing.info.ModTime.Equal(fi.ModTime()) && existing.info.Size == fi.Size() {
			return nil
		}

		robot, err := robotdiff.ParseRobotXMLFile(abs)
		if err != nil {
			return nil
		}
		pass, fail, total := robotdiff.CountTests(&robot.Suite)

		entry := &runEntry{
			abs:   abs,
			robot: robot,
			info: RunInfo{
				ID:        id,
				Name:      name,
				RelPath:   filepath.ToSlash(rel),
				ModTime:   fi.ModTime(),
				Size:      fi.Size(),
				TestCount: total,
				PassCount: pass,
				FailCount: fail,
			},
		}

		s.mu.Lock()
		s.runs[id] = entry
		s.mu.Unlock()
		return nil
	})
}

func stableID(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
