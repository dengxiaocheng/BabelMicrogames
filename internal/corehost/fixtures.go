package corehost

import (
	"encoding/json"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/timecore"
)

func FixturePair(operation string) (SceneHostABIRequest, SceneHostABIResponse, error) {
	switch operation {
	case operationSoloStep:
		return soloStepFixture()
	case operationRoomStep:
		return roomStepFixture()
	default:
		return SceneHostABIRequest{}, SceneHostABIResponse{}, ErrUnknownFixtureOperation(operation)
	}
}

type unknownFixtureOperationError string

func (e unknownFixtureOperationError) Error() string {
	return "unknown fixture operation: " + string(e)
}

func ErrUnknownFixtureOperation(operation string) error {
	return unknownFixtureOperationError(operation)
}

func soloStepFixture() (SceneHostABIRequest, SceneHostABIResponse, error) {
	requestState := types.SoloSession{
		SessionID:     "wechat:solo_scene:fixture_user",
		UserID:        "fixture_user",
		SchemaVersion: 1,
		StateVersion:  7,
		RulesetID:     "bootstrap.ruleset",
		PromptPackID:  "bootstrap.prompt_pack",
		Status:        types.RuntimeStatusActive,
		WorldClock:    types.WorldClock{DayIndex: 3, Segment: timecore.SegmentMorning, AbsoluteTick: 15},
		PlayerState: types.PlayerState{
			ActorID: "fixture_user",
			Stats:   types.Stats{Stamina: 4, Spirit: 5, Satiety: 3, Health: 5},
			Location: types.ActorLocation{
				ZoneID: "tower_base",
			},
		},
		EventFlags: map[string]types.FlagValue{
			"tower.open": {
				Kind:      types.FlagBool,
				BoolValue: true,
			},
		},
	}
	request := SceneHostABIRequest{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationSoloStep,
		Runtime: types.RuntimeRecord{
			RuntimeID:       requestState.SessionID,
			ModeID:          types.ModeSoloScene,
			SchemaVersion:   1,
			HeadVersion:     requestState.StateVersion,
			RulesetID:       "bootstrap.ruleset",
			PromptPackID:    "bootstrap.prompt_pack",
			GameplayAssetID: "bootstrap.gameplay_asset",
			Status:          types.RuntimeStatusActive,
		},
		Requirements: types.RuntimeRequirements{
			Ruleset:       &types.RulesetBundle{RulesetID: "bootstrap.ruleset", Version: "0.1.0"},
			PromptPack:    &types.PromptPackBundle{PromptPackID: "bootstrap.prompt_pack", Version: "0.1.0"},
			GameplayAsset: &types.GameplayAssetBundle{GameplayAssetID: "bootstrap.gameplay_asset", Version: "0.1.0"},
		},
		State: mustJSON(requestState),
		Action: types.PendingAction{
			ActionID:       "fixture-solo-exec",
			UserInput:      "继续搬砖",
			ActionType:     "free_text",
			IdempotencyKey: "fixture-solo-idem",
		},
	}
	responseState := requestState
	responseState.StateVersion = 8
	responseState.WorldClock = types.WorldClock{DayIndex: 3, Segment: timecore.SegmentNoon, AbsoluteTick: 16}
	responseState.PlayerState.Stats.Stamina = 3
	responseState.LastCommittedAction = "fixture-solo-exec"
	responseState.LastActionText = "继续搬砖"

	return request, SceneHostABIResponse{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationSoloStep,
		State:           mustJSON(responseState),
	}, nil
}

func roomStepFixture() (SceneHostABIRequest, SceneHostABIResponse, error) {
	requestState := types.MultiplayerRoom{
		RoomID:        "wechat:room_scene:main",
		SchemaVersion: 1,
		StateVersion:  11,
		RulesetID:     "bootstrap.ruleset",
		PromptPackID:  "bootstrap.prompt_pack",
		Status:        types.RuntimeStatusActive,
		WorldClock:    types.WorldClock{DayIndex: 5, Segment: timecore.SegmentAfternoon, AbsoluteTick: 22},
		Roster: []types.RoomMember{
			{PlayerID: "player_a", DisplayName: "player_a", Ready: true},
			{PlayerID: "player_b", DisplayName: "player_b", Ready: true},
		},
		SharedState: types.SharedSceneState{
			ZoneID:       "yard",
			StageID:      "round_open",
			PublicEvents: []string{"第2轮已开始，等待所有玩家提交行动。"},
			AggregateMetrics: map[string]int{
				"open_round":        2,
				"completed_rounds":  1,
				"submitted_actions": 1,
			},
		},
		HiddenState: map[string]types.PrivateState{
			"player_a": {ActorID: "worker:player_a"},
			"player_b": {ActorID: "worker:player_b"},
		},
		PendingTurn: &types.PendingTurn{
			TurnID:          "wechat:room_scene:main:turn:12",
			Segment:         timecore.SegmentAfternoon,
			RequiredPlayers: []string{"player_a", "player_b"},
			SubmittedActions: map[string]types.PendingAction{
				"player_a": {
					ActionID:       "fixture-room-existing",
					UserInput:      "我去抬木头",
					ActionType:     "free_text",
					IdempotencyKey: "fixture-room-existing-idem",
					Parameters:     map[string]string{"player_id": "player_a"},
				},
			},
		},
	}
	request := SceneHostABIRequest{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationRoomStep,
		Runtime: types.RuntimeRecord{
			RuntimeID:       requestState.RoomID,
			ModeID:          types.ModeRoomScene,
			SchemaVersion:   1,
			HeadVersion:     requestState.StateVersion,
			RulesetID:       "bootstrap.ruleset",
			PromptPackID:    "bootstrap.prompt_pack",
			GameplayAssetID: "bootstrap.gameplay_asset",
			Status:          types.RuntimeStatusActive,
		},
		Requirements: types.RuntimeRequirements{
			Ruleset:       &types.RulesetBundle{RulesetID: "bootstrap.ruleset", Version: "0.1.0"},
			PromptPack:    &types.PromptPackBundle{PromptPackID: "bootstrap.prompt_pack", Version: "0.1.0"},
			GameplayAsset: &types.GameplayAssetBundle{GameplayAssetID: "bootstrap.gameplay_asset", Version: "0.1.0"},
		},
		State: mustJSON(requestState),
		Action: types.PendingAction{
			ActionID:       "fixture-room-exec",
			UserInput:      "我去拉绳",
			ActionType:     "free_text",
			IdempotencyKey: "fixture-room-idem",
			Parameters:     map[string]string{"player_id": "player_b"},
		},
	}
	responseState := requestState
	responseState.StateVersion = 12
	responseState.WorldClock = types.WorldClock{DayIndex: 5, Segment: timecore.SegmentDusk, AbsoluteTick: 23}
	responseState.LastCommittedTurnID = requestState.PendingTurn.TurnID
	responseState.PendingTurn = nil
	responseState.SharedState.StageID = "round_resolved"
	responseState.SharedState.PublicEvents = []string{
		"第1轮已结算，共收到2个行动。",
	}
	responseState.SharedState.AggregateMetrics = map[string]int{
		"completed_rounds":      2,
		"last_resolved_round":   2,
		"last_resolved_actions": 2,
	}

	return request, SceneHostABIResponse{
		ContractVersion: SceneHostContractVersion,
		Operation:       operationRoomStep,
		State:           mustJSON(responseState),
	}, nil
}

func MarshalFixture(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
