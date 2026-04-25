package corehost

import (
	"context"
	"encoding/json"
	"fmt"

	"babel-runtime/internal/core/types"
)

const SceneHostContractVersion = "scene_host_v1"

const (
	operationSoloStep = "solo_step"
	operationRoomStep = "room_step"
)

type ByteTransport interface {
	Call(ctx context.Context, request []byte) ([]byte, error)
}

type SharedLibrarySceneHost struct {
	Transport ByteTransport
}

type SceneHostABIRequest struct {
	ContractVersion string                    `json:"contract_version"`
	Operation       string                    `json:"operation"`
	Runtime         types.RuntimeRecord       `json:"runtime"`
	Requirements    types.RuntimeRequirements `json:"requirements,omitempty"`
	State           json.RawMessage           `json:"state"`
	Action          types.PendingAction       `json:"action"`
}

type SceneHostABIResponse struct {
	ContractVersion string          `json:"contract_version"`
	Operation       string          `json:"operation"`
	State           json.RawMessage `json:"state"`
}

func NewSharedLibrarySceneHost(path string) (SharedLibrarySceneHost, error) {
	transport, err := NewDlopenTransport(path)
	if err != nil {
		return SharedLibrarySceneHost{}, err
	}
	return SharedLibrarySceneHost{Transport: transport}, nil
}

func (h SharedLibrarySceneHost) StepSolo(ctx context.Context, input SoloStepInput) (types.SoloSession, error) {
	var state types.SoloSession
	if err := callSharedLibrary(ctx, h.Transport, SceneHostABIRequest{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationSoloStep,
		Runtime:         input.Runtime,
		Requirements:    input.Requirements,
		Action:          input.Action,
		State:           mustJSON(input.Session),
	}, &state); err != nil {
		return types.SoloSession{}, err
	}
	return state, nil
}

func (h SharedLibrarySceneHost) StepRoom(ctx context.Context, input RoomStepInput) (types.MultiplayerRoom, error) {
	var state types.MultiplayerRoom
	if err := callSharedLibrary(ctx, h.Transport, SceneHostABIRequest{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationRoomStep,
		Runtime:         input.Runtime,
		Requirements:    input.Requirements,
		Action:          input.Action,
		State:           mustJSON(input.Room),
	}, &state); err != nil {
		return types.MultiplayerRoom{}, err
	}
	return state, nil
}

func callSharedLibrary(ctx context.Context, transport ByteTransport, request SceneHostABIRequest, out any) error {
	if transport == nil {
		return fmt.Errorf("missing shared library transport")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	responsePayload, err := transport.Call(ctx, payload)
	if err != nil {
		return err
	}
	var response SceneHostABIResponse
	if err := json.Unmarshal(responsePayload, &response); err != nil {
		return err
	}
	if response.ContractVersion != SceneHostContractVersion {
		return fmt.Errorf("unexpected contract version %q", response.ContractVersion)
	}
	if response.Operation != request.Operation {
		return fmt.Errorf("unexpected operation %q", response.Operation)
	}
	if len(response.State) == 0 {
		return fmt.Errorf("missing response state")
	}
	return json.Unmarshal(response.State, out)
}

func mustJSON(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}
