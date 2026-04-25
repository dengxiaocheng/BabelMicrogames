package corehost_test

import (
	"context"
	"encoding/json"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
)

type fakeByteTransport struct {
	callFn func(context.Context, []byte) ([]byte, error)
}

func (t fakeByteTransport) Call(ctx context.Context, request []byte) ([]byte, error) {
	return t.callFn(ctx, request)
}

func TestSharedLibrarySceneHostStepSolo(t *testing.T) {
	host := corehost.SharedLibrarySceneHost{
		Transport: fakeByteTransport{
			callFn: func(ctx context.Context, request []byte) ([]byte, error) {
				var decoded corehost.SceneHostABIRequest
				if err := json.Unmarshal(request, &decoded); err != nil {
					t.Fatalf("json.Unmarshal request returned error: %v", err)
				}
				if decoded.ContractVersion != corehost.SceneHostContractVersion {
					t.Fatalf("unexpected contract version %q", decoded.ContractVersion)
				}
				if decoded.Operation != "solo_step" {
					t.Fatalf("unexpected operation %q", decoded.Operation)
				}

				var state types.SoloSession
				if err := json.Unmarshal(decoded.State, &state); err != nil {
					t.Fatalf("json.Unmarshal state returned error: %v", err)
				}
				state.StateVersion++
				state.LastActionText = decoded.Action.UserInput

				return json.Marshal(corehost.SceneHostABIResponse{
					ContractVersion: corehost.SceneHostContractVersion,
					Operation:       "solo_step",
					State:           mustMarshal(t, state),
				})
			},
		},
	}

	updated, err := host.StepSolo(context.Background(), corehost.SoloStepInput{
		Runtime: types.RuntimeRecord{
			RuntimeID: "solo-1",
			ModeID:    types.ModeSoloScene,
		},
		Session: types.SoloSession{
			SessionID:     "solo-1",
			SchemaVersion: 1,
			StateVersion:  3,
		},
		Action: types.PendingAction{
			ActionID:   "exec-1",
			UserInput:  "继续搬砖",
			ActionType: "free_text",
		},
	})
	if err != nil {
		t.Fatalf("StepSolo returned error: %v", err)
	}
	if updated.StateVersion != 4 {
		t.Fatalf("expected version 4, got %d", updated.StateVersion)
	}
	if updated.LastActionText != "继续搬砖" {
		t.Fatalf("expected updated action text, got %q", updated.LastActionText)
	}
}

func TestSharedLibrarySceneHostStepRoom(t *testing.T) {
	host := corehost.SharedLibrarySceneHost{
		Transport: fakeByteTransport{
			callFn: func(ctx context.Context, request []byte) ([]byte, error) {
				var decoded corehost.SceneHostABIRequest
				if err := json.Unmarshal(request, &decoded); err != nil {
					t.Fatalf("json.Unmarshal request returned error: %v", err)
				}
				if decoded.Operation != "room_step" {
					t.Fatalf("unexpected operation %q", decoded.Operation)
				}

				var state types.MultiplayerRoom
				if err := json.Unmarshal(decoded.State, &state); err != nil {
					t.Fatalf("json.Unmarshal state returned error: %v", err)
				}
				state.LastCommittedTurnID = "turn-2"
				state.PendingTurn = nil

				return json.Marshal(corehost.SceneHostABIResponse{
					ContractVersion: corehost.SceneHostContractVersion,
					Operation:       "room_step",
					State:           mustMarshal(t, state),
				})
			},
		},
	}

	updated, err := host.StepRoom(context.Background(), corehost.RoomStepInput{
		Runtime: types.RuntimeRecord{
			RuntimeID: "room-1",
			ModeID:    types.ModeRoomScene,
		},
		Room: types.MultiplayerRoom{
			RoomID:        "room-1",
			SchemaVersion: 1,
			StateVersion:  5,
			PendingTurn: &types.PendingTurn{
				TurnID:           "turn-2",
				RequiredPlayers:  []string{"u1", "u2"},
				SubmittedActions: map[string]types.PendingAction{},
			},
		},
		Action: types.PendingAction{
			ActionID:   "exec-2",
			UserInput:  "拉绳",
			ActionType: "free_text",
		},
	})
	if err != nil {
		t.Fatalf("StepRoom returned error: %v", err)
	}
	if updated.LastCommittedTurnID != "turn-2" {
		t.Fatalf("expected committed turn, got %q", updated.LastCommittedTurnID)
	}
	if updated.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared")
	}
}

func mustMarshal(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return payload
}
