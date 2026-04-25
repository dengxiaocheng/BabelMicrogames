package render

import (
	"context"
	"fmt"
	"strings"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/llm"
)

// SimpleRenderer adapts settlement output into a minimal user-facing frame.
type SimpleRenderer struct {
	LLM llm.Renderer
}

func (r SimpleRenderer) RenderSolo(ctx context.Context, session types.SoloSession, action types.PendingAction, result types.StepResult) (types.RenderFrame, error) {
	if r.LLM == nil {
		return types.RenderFrame{}, fmt.Errorf("missing llm renderer")
	}

	req := llm.SoloRenderRequest{
		RequestID:          result.Checkpoint,
		PromptPackID:       session.PromptPackID,
		TonePackID:         "default",
		ChapterID:          session.ChapterID,
		WorldClock:         result.Delta.NextClock,
		PlayerSummary:      buildPlayerSummary(session.PlayerState),
		EnvironmentSummary: buildEnvironmentSummary(session.EnvironmentState),
		RecentSummary:      buildRecentSummary(result.Delta.Consequences),
		Consequences:       result.Delta.Consequences,
		PlayerActionLabel:  action.UserInput,
		MaxSceneChars:      240,
		MaxOptionChars:     24,
		OptionCount:        3,
	}

	resp, err := r.LLM.RenderSolo(ctx, req)
	if err != nil {
		return types.RenderFrame{}, err
	}

	return types.RenderFrame{
		FrameID:         result.Checkpoint + ":frame",
		OwnerType:       string(types.RuntimeSolo),
		OwnerID:         session.SessionID,
		BasedOnActionID: action.ActionID,
		StateVersion:    result.StateVersion,
		VisibleText:     resp.SceneText,
		Options:         resp.Options,
	}, nil
}

func buildPlayerSummary(player types.PlayerState) string {
	return fmt.Sprintf(
		"actor=%s stamina=%d spirit=%d health=%d zone=%s",
		player.ActorID,
		player.Stats.Stamina,
		player.Stats.Spirit,
		player.Stats.Health,
		player.Location.ZoneID,
	)
}

func buildEnvironmentSummary(env types.EnvironmentState) string {
	return fmt.Sprintf(
		"site=%s weather=%s hazard=%d food=%d",
		env.SiteID,
		env.WeatherID,
		env.HazardLevel,
		env.FoodSupplyLevel,
	)
}

func buildRecentSummary(consequences []types.VisibleConsequence) string {
	if len(consequences) == 0 {
		return "no visible consequences"
	}

	parts := make([]string, 0, len(consequences))
	for _, consequence := range consequences {
		if consequence.TextHint == "" {
			continue
		}
		parts = append(parts, consequence.TextHint)
	}
	if len(parts) == 0 {
		return "visible consequences available"
	}
	return strings.Join(parts, "; ")
}
