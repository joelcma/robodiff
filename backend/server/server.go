package backend

import (
	"net/http"
	"sync"
	"time"

	rdiff "robot_diff/backend/diff"
	"robot_diff/backend/store"
)

type Server struct {
	store      *store.RunStore
	addr       string
	agentMu    sync.Mutex
	agentRobot *rdiff.Robot
	agentData  *agentIndex
}

func NewServer(addr string, store *store.RunStore) *Server {
	return &Server{addr: addr, store: store}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	ui, err := newUIHandler()
	if err != nil {
		return err
	}
	mux.Handle("/", ui)

	h := withCORS(mux)
	server := &http.Server{
		Addr:              s.addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}
