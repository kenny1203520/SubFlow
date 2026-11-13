package upgrade

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

var (
	ErrPreviousFailure      = errors.New("previous upgrade failed")
	ErrUnsupportedDowngrade = errors.New("downgrades are not supported")
	ErrInvalidVersion       = errors.New("invalid upgrade version")
)

type Failure struct {
	Code  string `json:"code"`
	Phase string `json:"phase,omitempty"`
}

func canonicalVersion(value string) string {
	value = "v" + strings.TrimPrefix(strings.TrimSpace(value), "v")
	core, _, _ := strings.Cut(value, "-")
	core, _, _ = strings.Cut(core, "+")
	if strings.Count(core, ".") != 2 || !semver.IsValid(value) {
		return ""
	}
	return value
}

func compareVersions(left, right string) int {
	return semver.Compare(canonicalVersion(left), canonicalVersion(right))
}

func checkVersions(recorded, executable string) error {
	target, source := canonicalVersion(executable), canonicalVersion(recorded)
	if target == "" || (recorded != "" && source == "") {
		return ErrInvalidVersion
	}
	if source != "" && compareVersions(source, target) > 0 {
		return fmt.Errorf("%w (data is at %s, executable is %s)", ErrUnsupportedDowngrade, source, target)
	}
	return nil
}

type Preflight struct {
	DataDir           string   `json:"dataDir"`
	DataVersion       string   `json:"dataVersion,omitempty"`
	ExecutableVersion string   `json:"executableVersion"`
	State             string   `json:"state,omitempty"`
	Blockers          []string `json:"blockers"`
}

func (m *Manager) Preflight(executable string) Preflight {
	result := Preflight{DataDir: m.DataDir, ExecutableVersion: canonicalVersion(executable), Blockers: []string{}}
	state, err := m.readState()
	if err != nil {
		result.Blockers = append(result.Blockers, "upgrade_state_unavailable")
		return result
	}
	result.DataVersion, result.State = canonicalVersion(state.AppVersion), state.State
	switch err := checkVersions(state.AppVersion, executable); {
	case errors.Is(err, ErrInvalidVersion):
		result.Blockers = append(result.Blockers, "invalid_upgrade_version")
	case errors.Is(err, ErrUnsupportedDowngrade):
		result.Blockers = append(result.Blockers, "unsupported_downgrade")
	}
	if state.ProblemCode == "restore_requires_matching_binary" {
		result.Blockers = append(result.Blockers, "restore_requires_matching_binary")
	} else if state.State == StatusFailed {
		result.Blockers = append(result.Blockers, "manual_retry_required")
	}
	info, err := os.Stat(filepath.Join(m.DataDir, "data.db"))
	switch {
	case errors.Is(err, os.ErrNotExist) || (err == nil && info.Size() == 0):
		if state.AppVersion != "" {
			result.Blockers = append(result.Blockers, "database_missing")
		}
	case err != nil || !info.Mode().IsRegular():
		result.Blockers = append(result.Blockers, "database_unavailable")
	}
	return result
}

type Progress struct {
	mu     sync.Mutex
	write  func(string, ...any)
	stage  string
	detail string
	since  time.Time
	stop   chan struct{}
	done   chan struct{}
	reset  chan struct{}
}

func (m *Manager) StartProgress(write func(string, ...any)) func() {
	p := &Progress{write: write, stop: make(chan struct{}), done: make(chan struct{}), reset: make(chan struct{}, 1)}
	m.progress = p
	p.Report(StatusChecking, "")
	go func() {
		defer close(p.done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-p.reset:
				ticker.Reset(15 * time.Second)
			case now := <-ticker.C:
				p.mu.Lock()
				if p.stage != StatusReady && p.stage != StatusFailed && now.After(p.since) {
					p.write("SubFlow upgrade in progress stage=%s elapsed=%s %s", p.stage, now.Sub(p.since).Round(time.Second), p.detail)
				}
				p.mu.Unlock()
			}
		}
	}()
	return sync.OnceFunc(func() { close(p.stop); <-p.done })
}

func (p *Progress) Report(stage, detail string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stage, p.detail, p.since = stage, detail, time.Now()
	p.write("SubFlow upgrade stage=%s %s", stage, detail)
	select {
	case p.reset <- struct{}{}:
	default:
	}
}
