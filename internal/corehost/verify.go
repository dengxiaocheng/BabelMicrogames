package corehost

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"babel-runtime/internal/core/types"
)

// VerifyFixtureTransport runs a standard SceneHost ABI fixture against a transport
// and checks that the exported shared-library contract matches the expected shape.
func VerifyFixtureTransport(ctx context.Context, transport ByteTransport, operation string) error {
	if transport == nil {
		return fmt.Errorf("missing shared library transport")
	}

	request, expected, err := FixturePair(operation)
	if err != nil {
		return err
	}

	requestPayload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	responsePayload, err := transport.Call(ctx, requestPayload)
	if err != nil {
		return err
	}

	var actual SceneHostABIResponse
	if err := json.Unmarshal(responsePayload, &actual); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if actual.ContractVersion != expected.ContractVersion {
		return fmt.Errorf("unexpected contract version %q, want %q", actual.ContractVersion, expected.ContractVersion)
	}
	if actual.Operation != expected.Operation {
		return fmt.Errorf("unexpected operation %q, want %q", actual.Operation, expected.Operation)
	}
	if len(actual.State) == 0 {
		return fmt.Errorf("missing response state")
	}
	if err := verifyOperationState(operation, expected.State, actual.State); err != nil {
		return err
	}
	return nil
}

func VerifyFixtureLibrary(ctx context.Context, path string, operations ...string) error {
	transport, err := NewDlopenTransport(path)
	if err != nil {
		return err
	}
	if closer, ok := transport.(interface{ Close() }); ok {
		defer closer.Close()
	}

	if len(operations) == 0 {
		operations = []string{operationSoloStep, operationRoomStep}
	}
	for _, operation := range operations {
		if err := VerifyFixtureTransport(ctx, transport, operation); err != nil {
			return fmt.Errorf("%s verification failed: %w", operation, err)
		}
	}
	return nil
}

func verifyOperationState(operation string, expected, actual json.RawMessage) error {
	switch operation {
	case operationSoloStep:
		return compareFixtureStates[types.SoloSession](operation, expected, actual)
	case operationRoomStep:
		return compareFixtureStates[types.MultiplayerRoom](operation, expected, actual)
	default:
		return ErrUnknownFixtureOperation(operation)
	}
}

func compareFixtureStates[T any](operation string, expectedPayload, actualPayload json.RawMessage) error {
	var expected T
	if err := json.Unmarshal(expectedPayload, &expected); err != nil {
		return fmt.Errorf("decode expected %s state: %w", operation, err)
	}
	var actual T
	if err := json.Unmarshal(actualPayload, &actual); err != nil {
		return fmt.Errorf("decode actual %s state: %w", operation, err)
	}
	if reflect.DeepEqual(actual, expected) {
		return nil
	}

	expectedJSON, err := MarshalFixture(expected)
	if err != nil {
		return fmt.Errorf("marshal expected %s state: %w", operation, err)
	}
	actualJSON, err := MarshalFixture(actual)
	if err != nil {
		return fmt.Errorf("marshal actual %s state: %w", operation, err)
	}
	return fmt.Errorf("state mismatch for %s\nexpected:\n%s\nactual:\n%s", operation, expectedJSON, actualJSON)
}
