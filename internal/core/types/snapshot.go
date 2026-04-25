package types

import (
	"encoding/json"
	"time"
)

func EncodeRuntimeSnapshot(runtimeID string, modeID ModeID, version int64, schemaVersion int, state any, updatedAt time.Time) (RuntimeSnapshot, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	return RuntimeSnapshot{
		RuntimeID:     runtimeID,
		ModeID:        modeID,
		SchemaVersion: schemaVersion,
		Version:       version,
		State:         body,
		UpdatedAtUnix: updatedAt.Unix(),
	}, nil
}

func DecodeRuntimeSnapshot[T any](snapshot RuntimeSnapshot) (T, error) {
	var value T
	if len(snapshot.State) == 0 {
		return value, nil
	}
	err := json.Unmarshal(snapshot.State, &value)
	return value, err
}
