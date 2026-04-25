package corehost_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

func TestLocalSceneHostStepSoloCarriesRequirementRefs(t *testing.T) {
	host := corehost.LocalSceneHost{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
	}

	updated, err := host.StepSolo(context.Background(), corehost.SoloStepInput{
		Runtime: types.RuntimeRecord{
			RuntimeID: "solo-1",
			ModeID:    types.ModeSoloScene,
		},
		Requirements: types.RuntimeRequirements{
			Ruleset:    &types.RulesetBundle{RulesetID: "bootstrap.ruleset"},
			PromptPack: &types.PromptPackBundle{PromptPackID: "bootstrap.prompt_pack"},
		},
		Session: types.SoloSession{
			SessionID:     "solo-1",
			SchemaVersion: 1,
			StateVersion:  3,
			WorldClock:    types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 2},
			PlayerState: types.PlayerState{
				ActorID: "solo-1",
				Stats:   types.Stats{Stamina: 5, Spirit: 5, Satiety: 5},
				Location: types.ActorLocation{
					ZoneID: "camp",
				},
			},
			EventFlags: map[string]types.FlagValue{},
		},
		Action: types.PendingAction{
			ActionID:       "exec-1",
			UserInput:      "继续搬砖",
			ActionType:     "free_text",
			IdempotencyKey: "idem-1",
		},
	})
	if err != nil {
		t.Fatalf("StepSolo returned error: %v", err)
	}
	if updated.RulesetID != "bootstrap.ruleset" {
		t.Fatalf("expected ruleset ref carried into state, got %q", updated.RulesetID)
	}
	if updated.PromptPackID != "bootstrap.prompt_pack" {
		t.Fatalf("expected prompt pack ref carried into state, got %q", updated.PromptPackID)
	}
	if updated.StateVersion != 4 {
		t.Fatalf("expected state version 4, got %d", updated.StateVersion)
	}
}

func TestLocalSceneHostStepRoomClosesTurnAndCarriesRequirementRefs(t *testing.T) {
	host := corehost.LocalSceneHost{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
	}

	updated, err := host.StepRoom(context.Background(), corehost.RoomStepInput{
		Runtime: types.RuntimeRecord{
			RuntimeID: "room-1",
			ModeID:    types.ModeRoomScene,
		},
		Requirements: types.RuntimeRequirements{
			Ruleset:    &types.RulesetBundle{RulesetID: "bootstrap.ruleset"},
			PromptPack: &types.PromptPackBundle{PromptPackID: "bootstrap.prompt_pack"},
		},
		Room: types.MultiplayerRoom{
			RoomID:        "room-1",
			SchemaVersion: 1,
			StateVersion:  5,
			Status:        types.RuntimeStatusActive,
			WorldClock:    types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 3},
			HiddenState: map[string]types.PrivateState{
				"u1": {ActorID: "worker-1"},
				"u2": {ActorID: "worker-2"},
			},
			SharedState: types.SharedSceneState{
				AggregateMetrics: map[string]int{"completed_rounds": 0},
			},
			PendingTurn: &types.PendingTurn{
				TurnID:          "turn-1",
				RequiredPlayers: []string{"u1", "u2"},
				SubmittedActions: map[string]types.PendingAction{
					"u1": {
						ActionID:   "a-1",
						UserInput:  "先占位",
						ActionType: "free_text",
						Parameters: map[string]string{"player_id": "u1"},
					},
				},
			},
		},
		Action: types.PendingAction{
			ActionID:       "a-2",
			UserInput:      "拉绳",
			ActionType:     "free_text",
			IdempotencyKey: "idem-2",
			Parameters:     map[string]string{"player_id": "u2"},
		},
	})
	if err != nil {
		t.Fatalf("StepRoom returned error: %v", err)
	}
	if updated.LastCommittedTurnID != "turn-1" {
		t.Fatalf("expected turn to close, got %q", updated.LastCommittedTurnID)
	}
	if updated.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared")
	}
	if updated.RulesetID != "bootstrap.ruleset" || updated.PromptPackID != "bootstrap.prompt_pack" {
		t.Fatalf("expected requirement refs carried into room state, got ruleset=%q prompt=%q", updated.RulesetID, updated.PromptPackID)
	}
}
