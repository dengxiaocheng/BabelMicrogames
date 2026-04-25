package mode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
)

type ProjectConsultModule struct {
	Now func() time.Time
}

func (m ProjectConsultModule) ModeID() types.ModeID {
	return types.ModeProjectConsult
}

func (m ProjectConsultModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	text := sanitizeConsultText(env.Text)
	if text == "" {
		return types.ModeCommand{}, fmt.Errorf("missing consult text")
	}
	return types.ModeCommand{
		CommandType: "consult.ask",
		Text:        text,
	}, nil
}

func (m ProjectConsultModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	_ = ctx
	if input.Snapshot == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing runtime snapshot")
	}

	state, err := types.DecodeRuntimeSnapshot[types.ProjectConsultState](*input.Snapshot)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}
	state.RuntimeID = input.Runtime.RuntimeID

	// Consult follows the same two-pass contract as free chat so tool-driven or
	// model-driven outputs stay as artifacts until the mode accepts them explicitly.
	if input.Execution.Stage == types.ExecutionAwaitingArtifacts {
		reply, err := assistantTextArtifact(input.Artifacts)
		if err != nil {
			return types.ModeExecutionResult{}, err
		}
		state.QueryCount++
		state.LastQuestion = input.Command.Text
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
					EventID: input.Execution.ExecutionID + ":project_consult.responded",
					Kind:    "project_consult.responded",
					Text:    state.LastAssistantText,
				},
			},
		}, nil
	}

	nextQuery := state.QueryCount + 1
	reply := buildConsultReply(input.Command.Text, nextQuery)
	return types.ModeExecutionResult{
		NextStage: types.ExecutionAwaitingArtifacts,
		AgentTasks: []types.AgentTaskSpec{
			{
				TaskID:       input.Execution.ExecutionID + ":reply",
				TaskType:     "project_consult.reply",
				RuntimeID:    input.Runtime.RuntimeID,
				Input:        reply,
				ArtifactType: "assistant_text",
			},
		},
	}, nil
}

func sanitizeConsultText(text string) string {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "/consult "):
		return strings.TrimSpace(trimmed[len("/consult "):])
	case lower == "/consult":
		return ""
	case strings.HasPrefix(lower, "consult:"):
		return strings.TrimSpace(trimmed[len("consult:"):])
	default:
		return trimmed
	}
}

func buildConsultReply(question string, count int64) string {
	return fmt.Sprintf("consult[%d]: %s", count, question)
}

func (m ProjectConsultModule) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

var _ Module = (*ProjectConsultModule)(nil)
