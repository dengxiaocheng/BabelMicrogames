package eventlog

import (
	"context"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/store"
)

type Snapshotter interface {
	Snapshot() store.MemorySnapshot
}

type ReplayRecord struct {
	Events      []types.InputEvent        `json:"events"`
	Checkpoints []types.RuntimeCheckpoint `json:"checkpoints"`
	Frames      []types.RenderFrame       `json:"frames"`
}

type SoloReplay struct {
	Session      types.SoloSession `json:"session"`
	History      ReplayRecord      `json:"history"`
	LastEventID  string            `json:"last_event_id,omitempty"`
	LastFrameID  string            `json:"last_frame_id,omitempty"`
	CheckpointID string            `json:"checkpoint_id,omitempty"`
}

type RoomReplay struct {
	Room         types.MultiplayerRoom `json:"room"`
	History      ReplayRecord          `json:"history"`
	LastEventID  string                `json:"last_event_id,omitempty"`
	CheckpointID string                `json:"checkpoint_id,omitempty"`
}

type Replayer struct {
	Store Snapshotter
}

func (r Replayer) ReplaySolo(ctx context.Context, sessionID string) (SoloReplay, error) {
	_ = ctx
	if r.Store == nil {
		return SoloReplay{}, nil
	}

	snapshot := r.Store.Snapshot()
	replay := SoloReplay{
		Session: snapshot.SoloSessions[sessionID],
		History: ReplayRecord{
			Events:      filterEvents(snapshot.Events, types.RuntimeSolo, sessionID),
			Checkpoints: filterCheckpoints(snapshot.Checkpoints, types.RuntimeSolo, sessionID),
			Frames:      filterFrames(snapshot.Frames, string(types.RuntimeSolo), sessionID),
		},
	}
	if n := len(replay.History.Events); n > 0 {
		replay.LastEventID = replay.History.Events[n-1].EventID
	}
	if n := len(replay.History.Frames); n > 0 {
		replay.LastFrameID = replay.History.Frames[n-1].FrameID
	}
	if n := len(replay.History.Checkpoints); n > 0 {
		replay.CheckpointID = replay.History.Checkpoints[n-1].CheckpointID
	}
	return replay, nil
}

func (r Replayer) ReplayRoom(ctx context.Context, roomID string) (RoomReplay, error) {
	_ = ctx
	if r.Store == nil {
		return RoomReplay{}, nil
	}

	snapshot := r.Store.Snapshot()
	replay := RoomReplay{
		Room: snapshot.Rooms[roomID],
		History: ReplayRecord{
			Events:      filterEvents(snapshot.Events, types.RuntimeMultiplayer, roomID),
			Checkpoints: filterCheckpoints(snapshot.Checkpoints, types.RuntimeMultiplayer, roomID),
			Frames:      filterFrames(snapshot.Frames, string(types.RuntimeMultiplayer), roomID),
		},
	}
	if n := len(replay.History.Events); n > 0 {
		replay.LastEventID = replay.History.Events[n-1].EventID
	}
	if n := len(replay.History.Checkpoints); n > 0 {
		replay.CheckpointID = replay.History.Checkpoints[n-1].CheckpointID
	}
	return replay, nil
}

func filterEvents(events []types.InputEvent, runtimeType types.RuntimeType, runtimeID string) []types.InputEvent {
	out := make([]types.InputEvent, 0)
	for _, event := range events {
		if event.RuntimeType == runtimeType && event.RuntimeID == runtimeID {
			out = append(out, event)
		}
	}
	return out
}

func filterCheckpoints(checkpoints []types.RuntimeCheckpoint, runtimeType types.RuntimeType, runtimeID string) []types.RuntimeCheckpoint {
	out := make([]types.RuntimeCheckpoint, 0)
	for _, checkpoint := range checkpoints {
		if checkpoint.RuntimeType == runtimeType && checkpoint.RuntimeID == runtimeID {
			out = append(out, checkpoint)
		}
	}
	return out
}

func filterFrames(frames []types.RenderFrame, ownerType, ownerID string) []types.RenderFrame {
	out := make([]types.RenderFrame, 0)
	for _, frame := range frames {
		if frame.OwnerType == ownerType && frame.OwnerID == ownerID {
			out = append(out, frame)
		}
	}
	return out
}
