package controlplane_test

import (
	"context"
	"strings"
	"testing"

	"babel-runtime/internal/controlplane"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/recovery"
	"babel-runtime/internal/store"
)

func TestHealthServiceSnapshot(t *testing.T) {
	cfg := controlplane.NewConfigStore(controlplane.RuntimeConfig{
		Version:               "v1",
		Environment:           "test",
		FreeChatEnabled:       true,
		ProjectConsultEnabled: true,
		MultiplayerEnabled:    true,
	})
	mem := store.NewMemoryStore()
	mem.SeedSoloSession(types.SoloSession{SessionID: "solo-1"})
	mem.SeedRoom(types.MultiplayerRoom{RoomID: "room-1"})
	if err := mem.RunTx(context.Background(), func(ctx context.Context, tx store.TxStore) error {
		return tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{
			CheckpointID: "cp-1",
			Status:       types.CheckpointSimulating,
			ResumePolicy: types.ResumeRetryStep,
		})
	}); err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}

	service := controlplane.HealthService{
		Config: cfg,
		Store:  mem,
		Recovery: recovery.SimpleSupervisor{
			Store: mem,
		},
	}

	status, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if !status.OK || status.ConfigVersion != "v1" {
		t.Fatalf("unexpected health status: %#v", status)
	}
	if status.SoloSessions != 1 || status.Rooms != 1 {
		t.Fatalf("unexpected store counts: %#v", status)
	}
	if status.StaleCheckpoints != 1 {
		t.Fatalf("expected one stale checkpoint, got %#v", status)
	}
}

func TestAdminServiceHandle(t *testing.T) {
	cfg := controlplane.NewConfigStore(controlplane.RuntimeConfig{
		Version:               "v2",
		Environment:           "dev",
		FreeChatEnabled:       true,
		ProjectConsultEnabled: false,
		MultiplayerEnabled:    true,
	})
	health := controlplane.HealthService{Config: cfg}
	service := controlplane.AdminService{
		Health: health,
		Config: cfg,
	}

	resp, err := service.Handle(context.Background(), "/config")
	if err != nil {
		t.Fatalf("Handle config returned error: %v", err)
	}
	if resp.Command != controlplane.CommandConfig || !strings.Contains(resp.Text, "version=v2") {
		t.Fatalf("unexpected config response: %#v", resp)
	}

	resp, err = service.Handle(context.Background(), "/health")
	if err != nil {
		t.Fatalf("Handle health returned error: %v", err)
	}
	if resp.Command != controlplane.CommandHealth || !strings.Contains(resp.Text, "env=dev") {
		t.Fatalf("unexpected health response: %#v", resp)
	}
}
