package coordinator

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/solo"
	"babel-runtime/internal/store"
)

type SimpleCoordinator struct {
	Store       store.Store
	SoloRuntime solo.Runtime
	SoloRenderer interface {
		RenderSolo(ctx context.Context, session types.SoloSession, action types.PendingAction, result types.StepResult) (types.RenderFrame, error)
	}
	RoomRuntime multiplayer.Runtime
}

func (c SimpleCoordinator) HandleSoloInput(ctx context.Context, in types.SoloInput) error {
	if c.Store == nil || c.SoloRuntime == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return c.Store.RunTx(ctx, func(ctx context.Context, tx store.TxStore) error {
		session, err := tx.LoadSoloSession(ctx, in.RuntimeID)
		if err != nil {
			return err
		}
		event := types.InputEvent{
			EventID:        in.IdempotencyKey,
			RuntimeType:    types.RuntimeSolo,
			RuntimeID:      in.RuntimeID,
			ActorID:        in.UserID,
			IdempotencyKey: in.IdempotencyKey,
			Payload:        []byte(in.Text),
		}
		if err := tx.AppendEvent(ctx, event); err != nil {
			return err
		}
		action := types.PendingAction{
			ActionID:       in.IdempotencyKey,
			UserInput:      in.Text,
			ActionType:     "free_text",
			IdempotencyKey: in.IdempotencyKey,
		}
		result, err := c.SoloRuntime.Step(ctx, session, action)
		if err != nil {
			return err
		}
		updatedSession := solo.ApplyStepResult(session, action, result, "")
		if c.SoloRenderer != nil {
			frame, err := c.SoloRenderer.RenderSolo(ctx, session, action, result)
			if err != nil {
				return err
			}
			updatedSession.LastRenderFrameID = frame.FrameID
			if err := tx.SaveRenderFrame(ctx, frame); err != nil {
				return err
			}
		}
		if err := tx.SaveSoloSession(ctx, updatedSession); err != nil {
			return err
		}
		return tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{
			CheckpointID:       result.Checkpoint,
			RuntimeType:        types.RuntimeSolo,
			RuntimeID:          in.RuntimeID,
			StateVersionBefore: session.StateVersion,
			StateVersionAfter:  result.StateVersion,
			InputEventID:       event.EventID,
			StepName:           result.Checkpoint,
			Status:             types.CheckpointCommitted,
			ResumePolicy:       types.ResumeRetryStep,
		})
	})
}

func (c SimpleCoordinator) HandleRoomInput(ctx context.Context, in types.RoomInput) error {
	if c.Store == nil || c.RoomRuntime == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return c.Store.RunTx(ctx, func(ctx context.Context, tx store.TxStore) error {
		room, err := tx.LoadRoom(ctx, in.RuntimeID)
		if err != nil {
			return err
		}
		event := types.InputEvent{
			EventID:        in.IdempotencyKey,
			RuntimeType:    types.RuntimeMultiplayer,
			RuntimeID:      in.RuntimeID,
			ActorID:        in.UserID,
			IdempotencyKey: in.IdempotencyKey,
			Payload:        []byte(in.Text),
		}
		if err := tx.AppendEvent(ctx, event); err != nil {
			return err
		}
		action := types.PendingAction{
			ActionID:       in.IdempotencyKey,
			UserInput:      in.Text,
			ActionType:     "free_text",
			Parameters:     map[string]string{"player_id": in.UserID},
			IdempotencyKey: in.IdempotencyKey,
		}
		updatedRoom, err := c.RoomRuntime.Submit(ctx, room, action)
		if err != nil {
			return err
		}
		if multiplayer.TurnReady(updatedRoom) {
			closedRoom, turnResult, err := c.RoomRuntime.CloseTurn(ctx, updatedRoom)
			if err != nil {
				return err
			}
			if err := tx.SaveRoom(ctx, closedRoom); err != nil {
				return err
			}
			return tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{
				CheckpointID:       turnResult.Checkpoint,
				RuntimeType:        types.RuntimeMultiplayer,
				RuntimeID:          in.RuntimeID,
				StateVersionBefore: room.StateVersion,
				StateVersionAfter:  turnResult.StateVersion,
				InputEventID:       event.EventID,
				StepName:           turnResult.Checkpoint,
				Status:             types.CheckpointCommitted,
				ResumePolicy:       types.ResumeRetryStep,
			})
		}
		return tx.SaveRoom(ctx, updatedRoom)
	})
}
