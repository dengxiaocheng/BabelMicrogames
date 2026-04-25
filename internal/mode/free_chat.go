package mode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
)

type FreeChatModule struct {
	Now func() time.Time
}

func (m FreeChatModule) ModeID() types.ModeID {
	return types.ModeFreeChat
}

func (m FreeChatModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	text := strings.TrimSpace(env.Text)
	if text == "" {
		return types.ModeCommand{}, fmt.Errorf("missing user text")
	}
	return types.ModeCommand{
		CommandType: "chat.user_text",
		Text:        text,
	}, nil
}

func (m FreeChatModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	_ = ctx
	if input.Snapshot == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing runtime snapshot")
	}

	state, err := types.DecodeRuntimeSnapshot[types.FreeChatState](*input.Snapshot)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}
	state.RuntimeID = input.Runtime.RuntimeID

	// Free chat is intentionally a two-pass mode in the new runtime:
	// first request narrative work as an artifact, then explicitly accept the
	// returned artifact into canonical snapshot state on the next stage.
	if input.Execution.Stage == types.ExecutionAwaitingArtifacts {
		reply, err := assistantTextArtifact(input.Artifacts)
		if err != nil {
			return types.ModeExecutionResult{}, err
		}
		state.TurnCount++
		state.LastUserText = input.Command.Text
		state.LastAssistantText = reply

		snapshot, err := types.EncodeRuntimeSnapshot(
			input.Runtime.RuntimeID,
			m.ModeID(),
			input.Runtime.HeadVersion+1,
			input.Runtime.SchemaVersion,
			state,
			m.now(),
		)
		if err != nil {
			return types.ModeExecutionResult{}, err
		}

		return types.ModeExecutionResult{
			NextStage: types.ExecutionSettled,
			Snapshot:  &snapshot,
			Events: []types.RuntimeEvent{
				{
					EventID: input.Execution.ExecutionID + ":free_chat.responded",
					Kind:    "free_chat.responded",
					Text:    state.LastAssistantText,
				},
			},
		}, nil
	}

	nextTurn := state.TurnCount + 1
	reply := buildFreeChatReply(input.Command.Text, nextTurn)
	return types.ModeExecutionResult{
		NextStage: types.ExecutionAwaitingArtifacts,
		AgentTasks: []types.AgentTaskSpec{
			{
				TaskID:       input.Execution.ExecutionID + ":reply",
				TaskType:     "free_chat.reply",
				RuntimeID:    input.Runtime.RuntimeID,
				Input:        reply,
				ArtifactType: "assistant_text",
			},
		},
	}, nil
}

func buildFreeChatReply(text string, turn int64) string {
	return fmt.Sprintf("free chat[%d]: %s", turn, text)
}

func (m FreeChatModule) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

var _ Module = (*FreeChatModule)(nil)
