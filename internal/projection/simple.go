package projection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
)

type Projector interface {
	Project(ctx context.Context, input types.ProjectionInput) (types.ProjectionResult, error)
}

type SimpleProjector struct {
	DefaultTransport string
	Now              func() time.Time
}

func (p SimpleProjector) Project(ctx context.Context, input types.ProjectionInput) (types.ProjectionResult, error) {
	_ = ctx
	if input.Snapshot == nil {
		return types.ProjectionResult{}, fmt.Errorf("missing runtime snapshot")
	}

	body, err := p.bodyFor(input)
	if err != nil {
		return types.ProjectionResult{}, err
	}
	if body == "" {
		return types.ProjectionResult{}, nil
	}

	frameID := input.Execution.ExecutionID + ":frame"
	frame := types.ProjectionFrame{
		FrameID:       frameID,
		RuntimeID:     input.Runtime.RuntimeID,
		ExecutionID:   input.Execution.ExecutionID,
		ModeID:        input.Runtime.ModeID,
		Body:          body,
		CreatedAtUnix: p.now().Unix(),
	}

	result := types.ProjectionResult{
		Frames: []types.ProjectionFrame{frame},
	}
	transport := input.Execution.Transport
	if transport == "" {
		transport = p.DefaultTransport
	}
	if transport != "" {
		result.DeliveryPlans = []types.DeliveryPlan{
			{
				PlanID:      input.Execution.ExecutionID + ":delivery:" + transport,
				RuntimeID:   input.Runtime.RuntimeID,
				ExecutionID: input.Execution.ExecutionID,
				RecipientID: input.Execution.ActorID,
				Transport:   transport,
				FrameID:     frameID,
				Payload:     body,
			},
		}
	}
	return result, nil
}

func (p SimpleProjector) bodyFor(input types.ProjectionInput) (string, error) {
	switch input.Runtime.ModeID {
	case types.ModeFreeChat:
		state, err := types.DecodeRuntimeSnapshot[types.FreeChatState](*input.Snapshot)
		if err != nil {
			return "", err
		}
		return state.LastAssistantText, nil
	case types.ModeProjectConsult:
		state, err := types.DecodeRuntimeSnapshot[types.ProjectConsultState](*input.Snapshot)
		if err != nil {
			return "", err
		}
		return state.LastAssistantText, nil
	case types.ModeSoloScene:
		state, err := types.DecodeRuntimeSnapshot[types.SoloSession](*input.Snapshot)
		if err != nil {
			return "", err
		}
		lastActionLine := "你刚才完成了一步行动。"
		if state.LastActionText != "" {
			lastActionLine = "刚才：" + state.LastActionText
		}
		return fmt.Sprintf(
			"【单人角色】\n第%d天·%s\n你现在在%s。\n体力=%d 精神=%d 饱腹=%d\n%s\n可继续直接描述行动，也可使用「决 / 想」按钮。",
			state.WorldClock.DayIndex,
			state.WorldClock.Segment,
			state.PlayerState.Location.ZoneID,
			state.PlayerState.Stats.Stamina,
			state.PlayerState.Stats.Spirit,
			state.PlayerState.Stats.Satiety,
			lastActionLine,
		), nil
	case types.ModeRoomScene:
		state, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](*input.Snapshot)
		if err != nil {
			return "", err
		}
		required := 0
		submitted := 0
		if state.PendingTurn != nil {
			required = len(state.PendingTurn.RequiredPlayers)
			submitted = len(state.PendingTurn.SubmittedActions)
		}
		players := make([]string, 0, len(state.Roster))
		for _, member := range state.Roster {
			if member.DisplayName != "" {
				players = append(players, member.DisplayName)
				continue
			}
			players = append(players, member.PlayerID)
		}
		playersLine := "暂无"
		if len(players) > 0 {
			playersLine = strings.Join(players, "、")
		}
		eventLine := "暂无"
		if len(state.SharedState.PublicEvents) > 0 {
			eventLine = state.SharedState.PublicEvents[len(state.SharedState.PublicEvents)-1]
		}
		resolvedLine := "暂无"
		if resolvedRound := state.SharedState.AggregateMetrics["last_resolved_round"]; resolvedRound > 0 {
			resolvedLine = fmt.Sprintf(
				"第%d轮/%d个行动",
				resolvedRound,
				state.SharedState.AggregateMetrics["last_resolved_actions"],
			)
		}
		return fmt.Sprintf(
			"【联机】\n第%d天·%s\n阶段=%s\n当前人数=%d\n玩家=%s\n本轮提交=%d/%d\n上轮结算：%s\n最近事件：%s",
			state.WorldClock.DayIndex,
			state.WorldClock.Segment,
			state.SharedState.StageID,
			len(state.Roster),
			playersLine,
			submitted,
			required,
			resolvedLine,
			eventLine,
		), nil
	default:
		return "", nil
	}
}

func (p SimpleProjector) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

var _ Projector = (*SimpleProjector)(nil)
