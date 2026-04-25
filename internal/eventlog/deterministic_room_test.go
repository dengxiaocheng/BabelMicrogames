package eventlog_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/eventlog"
	"babel-runtime/internal/mapcore"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/relcore"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/timecore"
)

func TestDeterministicRoomRunnerReplay(t *testing.T) {
	seedRoom := func() types.MultiplayerRoom {
		return types.MultiplayerRoom{
			RoomID:       "room-1",
			Status:       types.RuntimeStatusActive,
			StateVersion: 4,
			WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
			HiddenState: map[string]types.PrivateState{
				"u1": {ActorID: "worker-1"},
				"u2": {ActorID: "worker-2"},
			},
			PendingTurn: &types.PendingTurn{
				TurnID:           "turn-1",
				RequiredPlayers:  []string{"u1", "u2"},
				SubmittedActions: map[string]types.PendingAction{},
			},
		}
	}

	seed := seedRoom()
	harnessSeed := seedRoom()
	h := harness.NewMultiplayerHarness()
	h.SeedRoom(harnessSeed)
	if err := h.SubmitText(context.Background(), "room-1", "u1", "搬石头"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}
	if err := h.SubmitText(context.Background(), "room-1", "u2", "拉绳索"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}

	replay, err := (eventlog.Replayer{Store: h.Store}).ReplayRoom(context.Background(), "room-1")
	if err != nil {
		t.Fatalf("ReplayRoom returned error: %v", err)
	}

	runner := eventlog.DeterministicRoomRunner{
		Runtime: multiplayer.SimpleRuntime{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{
					TimeCore: timecore.SimpleCore{},
					MapCore:  mapcore.NewSimpleCore(),
					RelCore:  relcore.SimpleCore{},
				},
			},
		},
	}

	replayed, err := runner.Replay(context.Background(), seed, replay.History.Events)
	if err != nil {
		t.Fatalf("Replay returned error: %v", err)
	}

	if replayed.StateVersion != replay.Room.StateVersion {
		t.Fatalf("expected replayed state version %d, got %d", replay.Room.StateVersion, replayed.StateVersion)
	}
	if replayed.WorldClock != replay.Room.WorldClock {
		t.Fatalf("expected replayed world clock %#v, got %#v", replay.Room.WorldClock, replayed.WorldClock)
	}
	if replayed.LastCommittedTurnID != replay.Room.LastCommittedTurnID {
		t.Fatalf("expected last committed turn %q, got %q", replay.Room.LastCommittedTurnID, replayed.LastCommittedTurnID)
	}
	if replayed.PendingTurn != nil {
		t.Fatalf("expected no pending turn after replay, got %#v", replayed.PendingTurn)
	}
}
