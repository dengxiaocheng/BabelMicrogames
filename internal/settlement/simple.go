package settlement

import (
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mapcore"
	"babel-runtime/internal/relcore"
	"babel-runtime/internal/timecore"
)

type Dependencies struct {
	TimeCore timecore.Core
	MapCore  mapcore.Core
	RelCore  relcore.Core
}

type SimpleEngine struct {
	Deps Dependencies
}

func (e SimpleEngine) StepSolo(input types.SoloStepInput) (types.StepDelta, error) {
	if e.Deps.TimeCore == nil {
		return types.StepDelta{}, fmt.Errorf("missing time core")
	}
	if err := e.Deps.TimeCore.ValidateAction(input.Session.WorldClock, types.ActionSpec{
		ActionType: input.Action.ActionType,
		TargetIDs:  input.Action.TargetIDs,
		Parameters: input.Action.Parameters,
	}); err != nil {
		return types.StepDelta{}, err
	}

	player := input.Session.PlayerState
	if player.Stats.Stamina > 0 {
		player.Stats.Stamina--
	}
	nextClock := e.Deps.TimeCore.Advance(input.Session.WorldClock)

	var relGraph *types.RelationshipGraph
	if e.Deps.RelCore != nil {
		graph := e.Deps.RelCore.VisibleSlice(input.Session.RelationshipState, player.ActorID)
		relGraph = &graph
	}

	return types.StepDelta{
		NextClock:         nextClock,
		PlayerState:       &player,
		EnvironmentState:  &input.Session.EnvironmentState,
		RelationshipState: relGraph,
		Consequences: []types.VisibleConsequence{
			{
				Tag:      "action_applied",
				TextHint: "行动已结算",
				Severity: 1,
			},
		},
	}, nil
}

func (e SimpleEngine) StepRoom(input types.RoomStepInput) (types.StepDelta, error) {
	if e.Deps.TimeCore == nil {
		return types.StepDelta{}, fmt.Errorf("missing time core")
	}
	if input.Room.PendingTurn == nil && len(input.Actions) == 0 {
		return types.StepDelta{}, fmt.Errorf("missing room actions")
	}

	nextClock := e.Deps.TimeCore.Advance(input.Room.WorldClock)
	shared := input.Room.SharedState
	if shared.AggregateMetrics == nil {
		shared.AggregateMetrics = map[string]int{}
	}
	shared.AggregateMetrics["submitted_actions"] = len(input.Actions)
	shared.AggregateMetrics["completed_rounds"] = shared.AggregateMetrics["completed_rounds"] + 1
	shared.AggregateMetrics["last_resolved_round"] = shared.AggregateMetrics["completed_rounds"]
	shared.AggregateMetrics["last_resolved_actions"] = len(input.Actions)
	shared.StageID = "round_resolved"
	shared.PublicEvents = append(shared.PublicEvents, fmt.Sprintf("第%d轮已结算，共收到%d个行动。", shared.AggregateMetrics["completed_rounds"], len(input.Actions)))
	if len(shared.PublicEvents) > 3 {
		shared.PublicEvents = append([]string(nil), shared.PublicEvents[len(shared.PublicEvents)-3:]...)
	}
	delete(shared.AggregateMetrics, "open_round")

	privateStates := map[string]types.PrivateState{}
	for _, member := range input.Room.Roster {
		if state, ok := input.Room.HiddenState[member.PlayerID]; ok {
			privateStates[member.PlayerID] = state
		}
	}

	return types.StepDelta{
		NextClock:     nextClock,
		SharedState:   &shared,
		PrivateStates: privateStates,
		Consequences: []types.VisibleConsequence{
			{
				Tag:      "turn_applied",
				TextHint: "联机阶段已结算",
				Severity: 1,
			},
		},
	}, nil
}
