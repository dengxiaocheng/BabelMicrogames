package mode_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

func TestSoloSceneModuleExecutesAgainstSnapshot(t *testing.T) {
	module := mode.SoloSceneModule{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
		Now: func() time.Time {
			return time.Unix(1700000200, 0)
		},
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		"solo-1",
		types.ModeSoloScene,
		1,
		1,
		types.SoloSession{
			SessionID:     "solo-1",
			SchemaVersion: 1,
			StateVersion:  1,
			Status:        types.RuntimeStatusActive,
			WorldClock:    types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
			PlayerState:   types.PlayerState{ActorID: "u1", Stats: types.Stats{Stamina: 3}},
		},
		time.Unix(1700000000, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	result, err := module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:   "solo-1",
			ModeID:      types.ModeSoloScene,
			HeadVersion: 1,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "solo-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeSoloScene,
			Stage:          types.ExecutionPlanned,
		},
		Snapshot: &snapshot,
		Command: types.ModeCommand{
			CommandType: "scene.user_text",
			Text:        "向前探索",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.NextStage != types.ExecutionSettled {
		t.Fatalf("expected settled stage, got %q", result.NextStage)
	}
	if result.Snapshot == nil {
		t.Fatalf("expected updated snapshot")
	}

	updated, err := types.DecodeRuntimeSnapshot[types.SoloSession](*result.Snapshot)
	if err != nil {
		t.Fatalf("DecodeRuntimeSnapshot returned error: %v", err)
	}
	if updated.StateVersion != 2 {
		t.Fatalf("expected state version 2, got %d", updated.StateVersion)
	}
	if updated.PlayerState.Stats.Stamina != 2 {
		t.Fatalf("expected stamina 2, got %d", updated.PlayerState.Stats.Stamina)
	}
	if updated.WorldClock.AbsoluteTick != 2 {
		t.Fatalf("expected absolute tick 2, got %d", updated.WorldClock.AbsoluteTick)
	}
}
