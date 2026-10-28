package httpapi

import (
	"net/http"
	"strconv"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/upgrade"
)

type upgradeHistoryResponse struct {
	AppVersion        string                   `json:"appVersion"`
	PocketBaseVersion string                   `json:"pocketBaseVersion"`
	SchemaVersion     int                      `json:"schemaVersion"`
	State             string                   `json:"state"`
	Ready             bool                     `json:"ready"`
	Backups           []backupSummary          `json:"backups"`
	Migrations        []upgrade.MigrationEntry `json:"migrations"`
}

type backupSummary struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	AppVersion        string `json:"appVersion"`
	PocketBaseVersion string `json:"pocketBaseVersion"`
	CreatedAt         string `json:"createdAt"`
	FileCount         int    `json:"fileCount"`
}

func (a *API) listSystemUpgrades(e *core.RequestEvent) error {
	if err := a.Service.RequireSystemPermission(e.Request.Context(), authID(e), "system.settings.manage"); err != nil {
		return fail(e, err)
	}
	if a.Upgrades == nil {
		return fail(e, domain.ErrConfiguration)
	}
	state, backups, err := a.Upgrades.Snapshot()
	if err != nil {
		return fail(e, err)
	}
	page, perPage := 1, 20
	query := e.Request.URL.Query()
	if value := query.Get("page"); value != "" {
		page, err = strconv.Atoi(value)
		if err != nil || page < 1 {
			return fail(e, domain.ErrInvalid)
		}
	}
	if value := query.Get("perPage"); value != "" {
		perPage, err = strconv.Atoi(value)
		if err != nil || perPage < 1 || perPage > 100 {
			return fail(e, domain.ErrInvalid)
		}
	}
	start := (page - 1) * perPage
	if start > len(state.History) {
		start = len(state.History)
	}
	end := min(start+perPage, len(state.History))
	history := append([]upgrade.MigrationEntry(nil), state.History...)
	for left, right := 0, len(history)-1; left < right; left, right = left+1, right-1 {
		history[left], history[right] = history[right], history[left]
	}
	items := history[start:end]
	backupItems := make([]backupSummary, 0, len(backups))
	for _, backup := range backups {
		backupItems = append(backupItems, backupSummary{ID: backup.ID, Status: backup.Status, AppVersion: backup.AppVersion, PocketBaseVersion: backup.PocketBaseVersion, CreatedAt: backup.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), FileCount: len(backup.Files)})
	}
	data := upgradeHistoryResponse{AppVersion: state.AppVersion, PocketBaseVersion: state.PocketBaseVersion, SchemaVersion: state.SchemaVersion, State: state.State, Ready: state.Ready, Backups: backupItems, Migrations: items}
	return ok(e, http.StatusOK, data, map[string]int{"page": page, "perPage": perPage, "totalItems": len(history), "totalPages": (len(history) + perPage - 1) / perPage})
}

func (a *API) runtimeStatus(e *core.RequestEvent) error {
	if a.Upgrades == nil {
		return ok(e, http.StatusOK, upgrade.PublicStatus{State: upgrade.StatusChecking, Phase: upgrade.StatusChecking}, nil)
	}
	state, err := a.Upgrades.ReadState()
	if err != nil || state.State == "" {
		return ok(e, http.StatusOK, upgrade.PublicStatus{State: upgrade.StatusChecking, Phase: upgrade.StatusChecking}, nil)
	}
	problem := ""
	if state.State == upgrade.StatusFailed {
		problem = state.ProblemCode
	}
	return ok(e, http.StatusOK, upgrade.PublicStatus{State: state.State, Phase: state.Phase, Ready: state.Ready, ProblemCode: problem}, nil)
}

func (a *API) runtimeReady(e *core.RequestEvent) error {
	if a.Upgrades == nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]any{"data": upgrade.PublicStatus{State: upgrade.StatusChecking, Phase: upgrade.StatusChecking}})
	}
	state, err := a.Upgrades.ReadState()
	if err == nil && state.Ready {
		return e.JSON(http.StatusOK, map[string]any{"data": upgrade.PublicStatus{State: upgrade.StatusReady, Phase: upgrade.StatusReady, Ready: true}})
	}
	return e.JSON(http.StatusServiceUnavailable, map[string]any{"data": upgrade.PublicStatus{State: upgrade.StatusChecking, Phase: upgrade.StatusChecking}})
}

func (a *API) runtimeLive(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, map[string]any{"data": map[string]bool{"live": true}})
}
