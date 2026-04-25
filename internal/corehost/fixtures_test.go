package corehost_test

import (
	"encoding/json"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
)

func TestFixturePairSoloStep(t *testing.T) {
	req, resp, err := corehost.FixturePair("solo_step")
	if err != nil {
		t.Fatalf("FixturePair returned error: %v", err)
	}
	if req.ContractVersion != corehost.SceneHostContractVersion || resp.ContractVersion != corehost.SceneHostContractVersion {
		t.Fatalf("unexpected contract version")
	}
	var state types.SoloSession
	if err := json.Unmarshal(resp.State, &state); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if state.LastActionText != "继续搬砖" {
		t.Fatalf("unexpected last action text %q", state.LastActionText)
	}
}

func TestFixturePairRoomStep(t *testing.T) {
	req, resp, err := corehost.FixturePair("room_step")
	if err != nil {
		t.Fatalf("FixturePair returned error: %v", err)
	}
	if req.Operation != "room_step" || resp.Operation != "room_step" {
		t.Fatalf("unexpected operation")
	}
	var state types.MultiplayerRoom
	if err := json.Unmarshal(resp.State, &state); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if state.LastCommittedTurnID == "" {
		t.Fatalf("expected committed turn id")
	}
	if state.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared")
	}
}
