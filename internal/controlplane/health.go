package controlplane

import (
	"context"

	"babel-runtime/internal/recovery"
	"babel-runtime/internal/store"
)

type HealthStatus struct {
	OK                 bool   `json:"ok"`
	ConfigVersion      string `json:"config_version"`
	Environment        string `json:"environment"`
	FreeChatEnabled    bool   `json:"free_chat_enabled"`
	ProjectConsultOn   bool   `json:"project_consult_enabled"`
	MultiplayerEnabled bool   `json:"multiplayer_enabled"`
	SoloSessions       int    `json:"solo_sessions"`
	Rooms              int    `json:"rooms"`
	StoredEvents       int    `json:"stored_events"`
	StoredCheckpoints  int    `json:"stored_checkpoints"`
	RecoveredRuntimes  int    `json:"recovered_runtimes"`
	StaleCheckpoints   int    `json:"stale_checkpoints"`
}

type HealthService struct {
	Config   interface{ Snapshot() RuntimeConfig }
	Store    interface{ Snapshot() store.MemorySnapshot }
	Recovery interface{ Sweep(ctx context.Context) (recovery.Report, error) }
}

func (s HealthService) Snapshot(ctx context.Context) (HealthStatus, error) {
	cfg := RuntimeConfig{}
	if s.Config != nil {
		cfg = s.Config.Snapshot()
	}

	storeSnap := store.MemorySnapshot{}
	if s.Store != nil {
		storeSnap = s.Store.Snapshot()
	}

	report := recovery.Report{}
	var err error
	if s.Recovery != nil {
		report, err = s.Recovery.Sweep(ctx)
		if err != nil {
			return HealthStatus{}, err
		}
	}

	return HealthStatus{
		OK:                 true,
		ConfigVersion:      cfg.Version,
		Environment:        cfg.Environment,
		FreeChatEnabled:    cfg.FreeChatEnabled,
		ProjectConsultOn:   cfg.ProjectConsultEnabled,
		MultiplayerEnabled: cfg.MultiplayerEnabled,
		SoloSessions:       len(storeSnap.SoloSessions),
		Rooms:              len(storeSnap.Rooms),
		StoredEvents:       len(storeSnap.Events),
		StoredCheckpoints:  len(storeSnap.Checkpoints),
		RecoveredRuntimes:  report.RecoveredRuntimes,
		StaleCheckpoints:   report.StaleCheckpoints,
	}, nil
}
