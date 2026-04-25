package multiplayer

import "babel-runtime/internal/core/types"

func TurnReady(room types.MultiplayerRoom) bool {
	if room.PendingTurn == nil {
		return false
	}
	if len(room.PendingTurn.RequiredPlayers) == 0 {
		return len(room.PendingTurn.SubmittedActions) > 0
	}
	for _, playerID := range room.PendingTurn.RequiredPlayers {
		if _, ok := room.PendingTurn.SubmittedActions[playerID]; !ok {
			return false
		}
	}
	return true
}

func ApplyTurnResult(room types.MultiplayerRoom, result types.TurnResult) types.MultiplayerRoom {
	updated := room
	updated.StateVersion = result.StateVersion
	updated.WorldClock = result.Delta.NextClock
	updated.PendingTurn = nil
	updated.LastCommittedTurnID = result.Checkpoint
	if result.Delta.SharedState != nil {
		updated.SharedState = *result.Delta.SharedState
	}
	if result.Delta.PrivateStates != nil {
		updated.HiddenState = result.Delta.PrivateStates
	}
	return updated
}
