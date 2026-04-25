package corehost_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
)

func TestVerifyFixtureTransportSoloStep(t *testing.T) {
	request, response, err := corehost.FixturePair("solo_step")
	if err != nil {
		t.Fatalf("FixturePair returned error: %v", err)
	}

	transport := fakeByteTransport{
		callFn: func(ctx context.Context, payload []byte) ([]byte, error) {
			var decoded corehost.SceneHostABIRequest
			if err := json.Unmarshal(payload, &decoded); err != nil {
				t.Fatalf("json.Unmarshal request returned error: %v", err)
			}
			if decoded.Operation != request.Operation {
				t.Fatalf("unexpected operation %q", decoded.Operation)
			}
			return json.Marshal(response)
		},
	}

	if err := corehost.VerifyFixtureTransport(context.Background(), transport, "solo_step"); err != nil {
		t.Fatalf("VerifyFixtureTransport returned error: %v", err)
	}
}

func TestVerifyFixtureTransportRoomStepMismatch(t *testing.T) {
	_, response, err := corehost.FixturePair("room_step")
	if err != nil {
		t.Fatalf("FixturePair returned error: %v", err)
	}

	var room types.MultiplayerRoom
	if err := json.Unmarshal(response.State, &room); err != nil {
		t.Fatalf("json.Unmarshal response state returned error: %v", err)
	}
	room.LastCommittedTurnID = "unexpected-turn"
	response.State = mustMarshal(t, room)

	transport := fakeByteTransport{
		callFn: func(ctx context.Context, payload []byte) ([]byte, error) {
			return json.Marshal(response)
		},
	}

	err = corehost.VerifyFixtureTransport(context.Background(), transport, "room_step")
	if err == nil {
		t.Fatalf("expected verification error")
	}
	if !strings.Contains(err.Error(), "state mismatch for room_step") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}
