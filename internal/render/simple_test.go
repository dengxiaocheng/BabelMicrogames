package render_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/llm"
	"babel-runtime/internal/render"
	"babel-runtime/internal/testkit/fakellm"
)

func TestSimpleRendererRenderSolo(t *testing.T) {
	renderer := render.SimpleRenderer{
		LLM: &fakellm.Renderer{
			Response: llm.SoloRenderResponse{
				SceneText: "你抹了把汗，继续向前。",
				Options: []types.RenderOption{
					{OptionID: "rest", Label: "歇一下", ActionTag: "rest"},
				},
			},
		},
	}

	frame, err := renderer.RenderSolo(context.Background(), types.SoloSession{
		SessionID:    "solo-1",
		PromptPackID: "prompt-a",
		ChapterID:    "chapter-1",
		PlayerState: types.PlayerState{
			ActorID: "worker-1",
			Stats: types.Stats{
				Stamina: 5,
				Spirit:  4,
				Health:  6,
			},
			Location: types.ActorLocation{ZoneID: "yard"},
		},
		EnvironmentState: types.EnvironmentState{
			SiteID:          "babel",
			WeatherID:       "windy",
			HazardLevel:     2,
			FoodSupplyLevel: 3,
		},
	}, types.PendingAction{
		ActionID:  "act-1",
		UserInput: "继续搬砖",
	}, types.StepResult{
		StateVersion: 2,
		Checkpoint:   "cp-1",
		Delta: types.StepDelta{
			NextClock: types.WorldClock{DayIndex: 1, Segment: "midday", AbsoluteTick: 2},
			Consequences: []types.VisibleConsequence{
				{Tag: "action_applied", TextHint: "行动已结算", Severity: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderSolo returned error: %v", err)
	}

	if frame.OwnerID != "solo-1" {
		t.Fatalf("expected frame owner solo-1, got %q", frame.OwnerID)
	}
	if frame.VisibleText != "你抹了把汗，继续向前。" {
		t.Fatalf("expected rendered scene text, got %q", frame.VisibleText)
	}
	if len(frame.Options) != 1 || frame.Options[0].ActionTag != "rest" {
		t.Fatalf("expected one render option, got %#v", frame.Options)
	}
}
