package eventlog_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/eventlog"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/solo"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/timecore"
)

func TestDeterministicSoloRunnerReplay(t *testing.T) {
	seed := types.SoloSession{
		SessionID:    "solo-1",
		UserID:       "user-1",
		Status:       types.RuntimeStatusActive,
		StateVersion: 1,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
		PlayerState: types.PlayerState{
			ActorID: "worker-1",
			Stats:   types.Stats{Stamina: 4},
		},
	}

	h := harness.NewSoloHarness()
	h.SeedSession(seed)
	if err := h.SubmitText(context.Background(), "solo-1", "user-1", "继续干活"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}
	if err := h.SubmitText(context.Background(), "solo-1", "user-1", "继续前进"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}

	replay, err := (eventlog.Replayer{Store: h.Store}).ReplaySolo(context.Background(), "solo-1")
	if err != nil {
		t.Fatalf("ReplaySolo returned error: %v", err)
	}

	runner := eventlog.DeterministicSoloRunner{
		Runtime: solo.SimpleRuntime{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{TimeCore: timecore.SimpleCore{}},
			},
		},
	}
	replayed, err := runner.Replay(context.Background(), seed, replay.History.Events)
	if err != nil {
		t.Fatalf("Replay returned error: %v", err)
	}

	if replayed.StateVersion != replay.Session.StateVersion {
		t.Fatalf("expected replayed state version %d, got %d", replay.Session.StateVersion, replayed.StateVersion)
	}
	if replayed.WorldClock != replay.Session.WorldClock {
		t.Fatalf("expected replayed world clock %#v, got %#v", replay.Session.WorldClock, replayed.WorldClock)
	}
	if replayed.PlayerState.Stats.Stamina != replay.Session.PlayerState.Stats.Stamina {
		t.Fatalf("expected replayed stamina %d, got %d", replay.Session.PlayerState.Stats.Stamina, replayed.PlayerState.Stats.Stamina)
	}
	if replayed.LastCommittedAction != replay.Session.LastCommittedAction {
		t.Fatalf("expected last committed action %q, got %q", replay.Session.LastCommittedAction, replayed.LastCommittedAction)
	}
}
