package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/repository"
)

type sceneSessionManifest struct {
	RuntimeID       string       `json:"runtime_id"`
	ModeID          types.ModeID `json:"mode_id"`
	UpdatedAtUnix   int64        `json:"updated_at_unix"`
	PreferredWorker string       `json:"preferred_worker"`
	PrimaryDocument string       `json:"primary_document"`
	StateDocument   string       `json:"state_document"`
	RulesetID       string       `json:"ruleset_id,omitempty"`
	PromptPackID    string       `json:"prompt_pack_id,omitempty"`
	GameplayAssetID string       `json:"gameplay_asset_id,omitempty"`
}

func (b RuntimeBridge) syncSceneOperationalMemory(ctx context.Context, runtimeID string, modeID types.ModeID, reply string) error {
	if b.MemoryRoot == "" || b.Repo == nil {
		return nil
	}

	var runtime types.RuntimeRecord
	var snapshot types.RuntimeSnapshot
	err := b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		loadedRuntime, err := tx.LoadRuntime(ctx, runtimeID)
		if err != nil {
			return err
		}
		runtime = loadedRuntime
		loaded, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}
		snapshot = loaded
		return nil
	})
	if err != nil {
		return err
	}

	dir := filepath.Join(b.MemoryRoot, runtimeDirName(runtimeID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	manifestBody, err := json.MarshalIndent(sceneSessionManifest{
		RuntimeID:       runtimeID,
		ModeID:          modeID,
		UpdatedAtUnix:   b.now().Unix(),
		PreferredWorker: "claude_code",
		PrimaryDocument: "primary_context.md",
		StateDocument:   "scene_state.json",
		RulesetID:       runtime.RulesetID,
		PromptPackID:    runtime.PromptPackID,
		GameplayAssetID: runtime.GameplayAssetID,
	}, "", "  ")
	if err != nil {
		return err
	}

	primaryContext, err := buildScenePrimaryContext(runtime, snapshot, strings.TrimSpace(reply))
	if err != nil {
		return err
	}

	files := map[string][]byte{
		"session_manifest.json": append(manifestBody, '\n'),
		"scene_state.json":      append(snapshot.State, '\n'),
		"primary_context.md":    []byte(primaryContext),
		"recent_summary.md":     []byte(strings.TrimSpace(reply) + "\n"),
	}

	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func buildScenePrimaryContext(runtime types.RuntimeRecord, snapshot types.RuntimeSnapshot, reply string) (string, error) {
	switch snapshot.ModeID {
	case types.ModeSoloScene:
		state, err := types.DecodeRuntimeSnapshot[types.SoloSession](snapshot)
		if err != nil {
			return "", err
		}
		lines := []string{
			"# Primary Context",
			"",
			"- mode: `solo_scene`",
			fmt.Sprintf("- runtime: `%s`", snapshot.RuntimeID),
			fmt.Sprintf("- version: `%d`", snapshot.Version),
			"- preferred_worker: `claude_code`",
			fmt.Sprintf("- ruleset: `%s`", runtime.RulesetID),
			fmt.Sprintf("- prompt_pack: `%s`", runtime.PromptPackID),
			fmt.Sprintf("- gameplay_asset: `%s`", runtime.GameplayAssetID),
			"",
			"## Current Scene",
			"",
			fmt.Sprintf("- 第%d天·%s", state.WorldClock.DayIndex, state.WorldClock.Segment),
			fmt.Sprintf("- 位置：%s", state.PlayerState.Location.ZoneID),
			fmt.Sprintf("- 体力=%d 精神=%d 饱腹=%d", state.PlayerState.Stats.Stamina, state.PlayerState.Stats.Spirit, state.PlayerState.Stats.Satiety),
			"",
			"## Latest Action",
			"",
			state.LastActionText,
			"",
			"## Latest Reply",
			"",
			reply,
			"",
		}
		return strings.Join(lines, "\n"), nil
	case types.ModeRoomScene:
		state, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](snapshot)
		if err != nil {
			return "", err
		}
		playerNames := make([]string, 0, len(state.Roster))
		for _, member := range state.Roster {
			name := member.DisplayName
			if name == "" {
				name = member.PlayerID
			}
			playerNames = append(playerNames, name)
		}
		lines := []string{
			"# Primary Context",
			"",
			"- mode: `room_scene`",
			fmt.Sprintf("- runtime: `%s`", snapshot.RuntimeID),
			fmt.Sprintf("- version: `%d`", snapshot.Version),
			"- preferred_worker: `claude_code`",
			fmt.Sprintf("- ruleset: `%s`", runtime.RulesetID),
			fmt.Sprintf("- prompt_pack: `%s`", runtime.PromptPackID),
			fmt.Sprintf("- gameplay_asset: `%s`", runtime.GameplayAssetID),
			"",
			"## Room State",
			"",
			fmt.Sprintf("- 第%d天·%s", state.WorldClock.DayIndex, state.WorldClock.Segment),
			fmt.Sprintf("- 阶段：%s", state.SharedState.StageID),
			fmt.Sprintf("- 玩家：%s", strings.Join(playerNames, "、")),
			fmt.Sprintf("- 当前轮次：%d", state.SharedState.AggregateMetrics["open_round"]),
			fmt.Sprintf("- 已完成轮次：%d", state.SharedState.AggregateMetrics["completed_rounds"]),
			fmt.Sprintf("- 上轮结算：第%d轮 / %d 个行动", state.SharedState.AggregateMetrics["last_resolved_round"], state.SharedState.AggregateMetrics["last_resolved_actions"]),
			"",
			"## Recent Events",
			"",
			strings.Join(state.SharedState.PublicEvents, "\n"),
			"",
			"## Latest Reply",
			"",
			reply,
			"",
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", nil
	}
}

func runtimeDirName(runtimeID string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_")
	return replacer.Replace(runtimeID)
}

func unixTimeOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
