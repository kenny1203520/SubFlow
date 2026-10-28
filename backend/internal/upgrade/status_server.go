package upgrade

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"subflow/internal/web"
)

type PublicStatus struct {
	State       string `json:"state"`
	Phase       string `json:"phase"`
	Ready       bool   `json:"ready"`
	ProblemCode string `json:"problemCode,omitempty"`
}

type StatusServer struct {
	manager  *Manager
	server   *http.Server
	listener net.Listener
}

func StartStatusServer(address string, manager *Manager) (*StatusServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	service := &StatusServer{manager: manager, listener: listener}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/subflow/v1/runtime/status", service.status)
	mux.HandleFunc("/api/ready", service.ready)
	mux.HandleFunc("/api/health", service.ready)
	mux.HandleFunc("/api/live", service.live)
	mux.Handle("/", web.MaintenanceHandler())
	service.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = service.server.Serve(listener) }()
	return service, nil
}

func (s *StatusServer) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state, err := s.manager.ReadState()
	if err != nil || state.State == "" {
		state = State{State: StatusChecking, Phase: StatusChecking}
	}
	problem := ""
	if state.State == StatusFailed {
		problem = safeProblem(state.ProblemCode)
	}
	writeStatus(w, http.StatusOK, PublicStatus{State: state.State, Phase: state.Phase, Ready: state.Ready, ProblemCode: problem})
}

func (s *StatusServer) ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state, err := s.manager.ReadState()
	if err == nil && state.Ready {
		writeStatus(w, http.StatusOK, PublicStatus{State: StatusReady, Phase: StatusReady, Ready: true})
		return
	}
	writeStatus(w, http.StatusServiceUnavailable, PublicStatus{State: StatusChecking, Phase: StatusChecking, Ready: false})
}

func (s *StatusServer) live(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeStatus(w, http.StatusOK, map[string]bool{"live": true})
}

func writeStatus(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]any{"data": body}); err != nil {
		return
	}
}

func (s *StatusServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
