package solo

import "babel-runtime/internal/core/types"

func ApplyStepResult(session types.SoloSession, action types.PendingAction, result types.StepResult, frameID string) types.SoloSession {
	updated := session
	updated.StateVersion = result.StateVersion
	updated.WorldClock = result.Delta.NextClock
	updated.PendingAction = nil
	updated.LastCommittedAction = action.ActionID
	updated.LastActionText = action.UserInput
	if frameID != "" {
		updated.LastRenderFrameID = frameID
	}
	if result.Delta.PlayerState != nil {
		updated.PlayerState = *result.Delta.PlayerState
	}
	if result.Delta.EnvironmentState != nil {
		updated.EnvironmentState = *result.Delta.EnvironmentState
	}
	if result.Delta.RelationshipState != nil {
		updated.RelationshipState = *result.Delta.RelationshipState
	}
	return updated
}
