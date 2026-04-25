package wechat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/kernel"
	"babel-runtime/internal/repository"
)

type RuntimeBridge struct {
	Repo           repository.Repository
	Kernel         kernel.Engine
	MemoryRoot     string
	MaxResumeSteps int
	Now            func() time.Time
}

const (
	minRoomPlayers      = 2
	roomMemberTTL       = 2 * time.Minute
	defaultRulesetID    = "bootstrap.ruleset"
	defaultPromptPackID = "bootstrap.prompt_pack"
	defaultGameplayID   = "bootstrap.gameplay_asset"
)

func (b RuntimeBridge) HandleFreeChat(ctx context.Context, msg NormalizedMessage) (string, error) {
	return b.handleMode(ctx, msg, types.ModeFreeChat)
}

func (b RuntimeBridge) HandleProjectConsult(ctx context.Context, msg NormalizedMessage) (string, error) {
	return b.handleMode(ctx, msg, types.ModeProjectConsult)
}

func (b RuntimeBridge) HandleSoloScene(ctx context.Context, msg NormalizedMessage) (string, error) {
	return b.handleMode(ctx, msg, types.ModeSoloScene)
}

func (b RuntimeBridge) HandleRoomScene(ctx context.Context, msg NormalizedMessage) (string, error) {
	if b.Repo == nil || b.Kernel == nil {
		return "", fmt.Errorf("runtime bridge not initialized")
	}
	runtimeID := runtimeIDFor(types.ModeRoomScene, msg.UserID)
	if err := b.ensureRuntime(ctx, runtimeID, types.ModeRoomScene); err != nil {
		return "", err
	}
	if err := b.ensureRoomMember(ctx, runtimeID, msg.UserID); err != nil {
		return "", err
	}
	if err := b.ensureRoomPendingTurn(ctx, runtimeID); err != nil {
		return "", err
	}
	ready, err := b.roomHasPendingTurn(ctx, runtimeID)
	if err != nil {
		return "", err
	}
	if !ready {
		status, err := b.readRoomStatus(ctx, runtimeID, msg.UserID)
		if err != nil {
			return "", err
		}
		_ = b.syncSceneOperationalMemory(ctx, runtimeID, types.ModeRoomScene, status)
		return status, nil
	}

	switch strings.TrimSpace(msg.Text) {
	case "", "联机", "联机状态", "加入联机":
		status, err := b.readRoomStatus(ctx, runtimeID, msg.UserID)
		if err != nil {
			return "", err
		}
		_ = b.syncSceneOperationalMemory(ctx, runtimeID, types.ModeRoomScene, status)
		return status, nil
	}

	ticket, err := b.Kernel.Accept(ctx, types.InboundEnvelope{
		EnvelopeID:     runtimeID + ":" + msg.IdempotencyKey,
		RuntimeID:      runtimeID,
		UserID:         msg.UserID,
		IdempotencyKey: msg.IdempotencyKey,
		Transport:      "wechat",
		RouteHint:      string(types.ModeRoomScene),
		Text:           msg.Text,
		ReceivedAtUnix: b.now().Unix(),
	})
	if err != nil {
		return "", err
	}
	if err := b.resumeExecution(ctx, ticket.ExecutionID); err != nil {
		return "", err
	}
	if err := b.ensureRoomPendingTurn(ctx, runtimeID); err != nil {
		return "", err
	}
	status, err := b.readRoomStatus(ctx, runtimeID, msg.UserID)
	if err != nil {
		return "", err
	}
	reply := status + "\n你刚才提交：" + msg.Text
	_ = b.syncSceneOperationalMemory(ctx, runtimeID, types.ModeRoomScene, reply)
	return reply, nil
}

func (b RuntimeBridge) handleMode(ctx context.Context, msg NormalizedMessage, modeID types.ModeID) (string, error) {
	if b.Repo == nil || b.Kernel == nil {
		return "", fmt.Errorf("runtime bridge not initialized")
	}

	runtimeID := runtimeIDFor(modeID, msg.UserID)
	if err := b.ensureRuntime(ctx, runtimeID, modeID); err != nil {
		return "", err
	}

	ticket, err := b.Kernel.Accept(ctx, types.InboundEnvelope{
		EnvelopeID:     runtimeID + ":" + msg.IdempotencyKey,
		RuntimeID:      runtimeID,
		UserID:         msg.UserID,
		IdempotencyKey: msg.IdempotencyKey,
		Transport:      "wechat",
		RouteHint:      string(modeID),
		Text:           msg.Text,
		ReceivedAtUnix: b.now().Unix(),
	})
	if err != nil {
		return "", err
	}
	// WeChat expects an immediate reply path, so the bridge drives the staged
	// execution loop until a terminal stage is reached or the step budget is spent.
	if err := b.resumeExecution(ctx, ticket.ExecutionID); err != nil {
		return "", err
	}

	reply, err := b.readReply(ctx, runtimeID, modeID)
	if err != nil {
		return "", err
	}
	_ = b.syncSceneOperationalMemory(ctx, runtimeID, modeID, reply)
	return reply, nil
}

func (b RuntimeBridge) ensureRuntime(ctx context.Context, runtimeID string, modeID types.ModeID) error {
	return b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		if _, err := tx.LoadRuntime(ctx, runtimeID); err == nil {
			return nil
		} else if err != repository.ErrRuntimeNotFound {
			return err
		}

		record := defaultRuntimeRecord(runtimeID, modeID)
		if err := tx.SaveRuntime(ctx, record); err != nil {
			return err
		}

		snapshot, err := initialSnapshot(record, b.now())
		if err != nil {
			return err
		}
		return tx.SaveRuntimeSnapshot(ctx, snapshot)
	})
}

func (b RuntimeBridge) readReply(ctx context.Context, runtimeID string, modeID types.ModeID) (string, error) {
	// Prefer the latest projected frame because it reflects the transport-facing
	// response. Snapshot fallback exists for stages that have not projected yet.
	frames, err := b.Repo.ListProjectionFrames(ctx, runtimeID)
	if err == nil && len(frames) > 0 {
		return frames[len(frames)-1].Body, nil
	}

	var reply string
	err = b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}

		switch modeID {
		case types.ModeFreeChat:
			state, err := types.DecodeRuntimeSnapshot[types.FreeChatState](snapshot)
			if err != nil {
				return err
			}
			reply = state.LastAssistantText
		case types.ModeProjectConsult:
			state, err := types.DecodeRuntimeSnapshot[types.ProjectConsultState](snapshot)
			if err != nil {
				return err
			}
			reply = state.LastAssistantText
		default:
			return fmt.Errorf("unsupported reply mode: %s", modeID)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if reply == "" {
		return "queued", nil
	}
	return reply, nil
}

func initialSnapshot(runtime types.RuntimeRecord, now time.Time) (types.RuntimeSnapshot, error) {
	switch runtime.ModeID {
	case types.ModeFreeChat:
		return types.EncodeRuntimeSnapshot(runtime.RuntimeID, runtime.ModeID, 0, 1, types.FreeChatState{
			RuntimeID: runtime.RuntimeID,
		}, now)
	case types.ModeProjectConsult:
		return types.EncodeRuntimeSnapshot(runtime.RuntimeID, runtime.ModeID, 0, 1, types.ProjectConsultState{
			RuntimeID: runtime.RuntimeID,
		}, now)
	case types.ModeSoloScene:
		return types.EncodeRuntimeSnapshot(runtime.RuntimeID, runtime.ModeID, 0, 1, initialSoloSession(runtime), now)
	case types.ModeRoomScene:
		return types.EncodeRuntimeSnapshot(runtime.RuntimeID, runtime.ModeID, 0, 1, initialRoom(runtime), now)
	default:
		return types.RuntimeSnapshot{}, fmt.Errorf("unsupported runtime mode: %s", runtime.ModeID)
	}
}

func initialSoloSession(runtime types.RuntimeRecord) types.SoloSession {
	return types.SoloSession{
		SessionID:     runtime.RuntimeID,
		UserID:        runtime.RuntimeID,
		SchemaVersion: 1,
		RulesetID:     runtime.RulesetID,
		PromptPackID:  runtime.PromptPackID,
		Status:        types.RuntimeStatusActive,
		WorldClock:    types.WorldClock{DayIndex: 1, Segment: "dawn", AbsoluteTick: 0},
		PlayerState: types.PlayerState{
			ActorID: runtime.RuntimeID,
			Stats:   types.Stats{Stamina: 5, Spirit: 5, Satiety: 5, Health: 5},
			Location: types.ActorLocation{
				ZoneID: "camp",
			},
		},
		EventFlags: map[string]types.FlagValue{},
	}
}

func runtimeIDFor(modeID types.ModeID, userID string) string {
	if modeID == types.ModeRoomScene {
		return "wechat:" + string(modeID) + ":main"
	}
	return "wechat:" + string(modeID) + ":" + userID
}

func initialRoom(runtime types.RuntimeRecord) types.MultiplayerRoom {
	return types.MultiplayerRoom{
		RoomID:        runtime.RuntimeID,
		SchemaVersion: 1,
		RulesetID:     runtime.RulesetID,
		PromptPackID:  runtime.PromptPackID,
		Status:        types.RuntimeStatusLobby,
		WorldClock:    types.WorldClock{DayIndex: 1, Segment: "dawn", AbsoluteTick: 0},
		SharedState: types.SharedSceneState{
			ZoneID:           "yard",
			StageID:          "lobby",
			PublicEvents:     []string{"联机大厅已开放，至少两名活跃玩家才能开局。"},
			AggregateMetrics: map[string]int{"completed_rounds": 0},
		},
		HiddenState: map[string]types.PrivateState{},
	}
}

func defaultRuntimeRecord(runtimeID string, modeID types.ModeID) types.RuntimeRecord {
	record := types.RuntimeRecord{
		RuntimeID:     runtimeID,
		ModeID:        modeID,
		SchemaVersion: 1,
		HeadVersion:   0,
		Status:        types.RuntimeStatusActive,
	}
	switch modeID {
	case types.ModeSoloScene, types.ModeRoomScene:
		record.RulesetID = defaultRulesetID
		record.PromptPackID = defaultPromptPackID
		record.GameplayAssetID = defaultGameplayID
	}
	return record
}

func (b RuntimeBridge) ensureRoomMember(ctx context.Context, runtimeID, userID string) error {
	return b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}
		room, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](snapshot)
		if err != nil {
			return err
		}
		room = pruneStaleRoomMembers(room, b.now())

		joined := false
		for i, member := range room.Roster {
			if member.PlayerID == userID {
				room.Roster[i].LastActiveUnix = b.now().Unix()
				joined = true
				break
			}
		}
		if !joined {
			room.Roster = append(room.Roster, types.RoomMember{
				PlayerID:       userID,
				DisplayName:    userID,
				Ready:          true,
				LastActiveUnix: b.now().Unix(),
			})
			room.SharedState.PublicEvents = appendRoomEvent(room.SharedState.PublicEvents, fmt.Sprintf("%s 已加入联机大厅。", userID))
		}
		if room.HiddenState == nil {
			room.HiddenState = map[string]types.PrivateState{}
		}
		if _, ok := room.HiddenState[userID]; !ok {
			room.HiddenState[userID] = types.PrivateState{ActorID: "worker:" + userID}
		}
		if room.PendingTurn != nil && len(room.PendingTurn.SubmittedActions) == 0 {
			room.PendingTurn.RequiredPlayers = roomPlayerIDs(room)
		}
		if len(room.Roster) < minRoomPlayers {
			room.SharedState.StageID = "lobby"
		}

		encoded, err := types.EncodeRuntimeSnapshot(runtimeID, types.ModeRoomScene, room.StateVersion, room.SchemaVersion, room, b.now())
		if err != nil {
			return err
		}
		return tx.SaveRuntimeSnapshot(ctx, encoded)
	})
}

func (b RuntimeBridge) ensureRoomPendingTurn(ctx context.Context, runtimeID string) error {
	return b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}
		room, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](snapshot)
		if err != nil {
			return err
		}
		room = pruneStaleRoomMembers(room, b.now())
		if room.PendingTurn != nil {
			return nil
		}
		if len(room.Roster) < minRoomPlayers {
			room.Status = types.RuntimeStatusLobby
			room.SharedState.StageID = "lobby"
			encoded, err := types.EncodeRuntimeSnapshot(runtimeID, types.ModeRoomScene, room.StateVersion, room.SchemaVersion, room, b.now())
			if err != nil {
				return err
			}
			return tx.SaveRuntimeSnapshot(ctx, encoded)
		}
		nextRound := room.SharedState.AggregateMetrics["completed_rounds"] + 1
		room.PendingTurn = &types.PendingTurn{
			TurnID:           fmt.Sprintf("%s:turn:%d", runtimeID, room.StateVersion+1),
			Segment:          room.WorldClock.Segment,
			RequiredPlayers:  roomPlayerIDs(room),
			SubmittedActions: map[string]types.PendingAction{},
		}
		if room.Status == types.RuntimeStatusLobby {
			room.Status = types.RuntimeStatusActive
		}
		room.SharedState.StageID = "round_open"
		if room.SharedState.AggregateMetrics == nil {
			room.SharedState.AggregateMetrics = map[string]int{}
		}
		room.SharedState.AggregateMetrics["open_round"] = nextRound
		room.SharedState.PublicEvents = appendRoomEvent(room.SharedState.PublicEvents, fmt.Sprintf("第%d轮已开始，等待所有玩家提交行动。", nextRound))
		encoded, err := types.EncodeRuntimeSnapshot(runtimeID, types.ModeRoomScene, room.StateVersion, room.SchemaVersion, room, b.now())
		if err != nil {
			return err
		}
		return tx.SaveRuntimeSnapshot(ctx, encoded)
	})
}

func (b RuntimeBridge) readRoomStatus(ctx context.Context, runtimeID, userID string) (string, error) {
	var room types.MultiplayerRoom
	err := b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}
		loaded, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](snapshot)
		if err != nil {
			return err
		}
		room = pruneStaleRoomMembers(loaded, b.now())
		return nil
	})
	if err != nil {
		return "", err
	}

	required := 0
	submitted := 0
	waiting := "发送任意消息即可提交本轮行动。"
	submittedPlayersLine := ""
	waitingPlayersLine := ""
	if room.PendingTurn != nil {
		required = len(room.PendingTurn.RequiredPlayers)
		submitted = len(room.PendingTurn.SubmittedActions)
		if _, ok := room.PendingTurn.SubmittedActions[userID]; ok {
			waiting = "你已提交本轮行动，等待其他人。"
		}
		submittedPlayers, waitingPlayers := roomTurnParticipantNames(room)
		if len(submittedPlayers) > 0 {
			submittedPlayersLine = "已提交玩家=" + strings.Join(submittedPlayers, "、") + "\n"
		}
		if len(waitingPlayers) > 0 {
			waitingPlayersLine = "待提交玩家=" + strings.Join(waitingPlayers, "、") + "\n"
		}
	} else if len(room.Roster) < minRoomPlayers {
		waiting = fmt.Sprintf("等待更多玩家加入，至少需要%d人。", minRoomPlayers)
	}

	stageLabel := roomStageLabel(room.SharedState.StageID)
	playersLine := "暂无"
	if len(room.Roster) > 0 {
		playersLine = roomPlayerNames(room)
	}
	eventLine := "最近房间事件：暂无"
	if len(room.SharedState.PublicEvents) > 0 {
		eventLine = "最近房间事件：" + room.SharedState.PublicEvents[len(room.SharedState.PublicEvents)-1]
	}
	roundLine := ""
	if openRound := room.SharedState.AggregateMetrics["open_round"]; openRound > 0 {
		roundLine = fmt.Sprintf("当前轮次=%d\n", openRound)
	}
	resolvedLine := ""
	if resolvedRound := room.SharedState.AggregateMetrics["last_resolved_round"]; resolvedRound > 0 {
		resolvedLine = fmt.Sprintf(
			"上轮结算=第%d轮/%d个行动\n",
			resolvedRound,
			room.SharedState.AggregateMetrics["last_resolved_actions"],
		)
	}

	return fmt.Sprintf(
		"【联机】\n第%d天·%s\n阶段=%s\n当前人数=%d\n玩家=%s\n%s%s本轮提交=%d/%d\n%s%s%s\n%s",
		room.WorldClock.DayIndex,
		room.WorldClock.Segment,
		stageLabel,
		len(room.Roster),
		playersLine,
		roundLine,
		resolvedLine,
		submitted,
		required,
		submittedPlayersLine,
		waitingPlayersLine,
		waiting,
		eventLine,
	), nil
}

func roomPlayerIDs(room types.MultiplayerRoom) []string {
	playerIDs := make([]string, 0, len(room.Roster))
	for _, member := range room.Roster {
		playerIDs = append(playerIDs, member.PlayerID)
	}
	return playerIDs
}

func roomPlayerNames(room types.MultiplayerRoom) string {
	names := make([]string, 0, len(room.Roster))
	for _, member := range room.Roster {
		name := member.DisplayName
		if name == "" {
			name = member.PlayerID
		}
		names = append(names, name)
	}
	return strings.Join(names, "、")
}

func roomTurnParticipantNames(room types.MultiplayerRoom) ([]string, []string) {
	if room.PendingTurn == nil {
		return nil, nil
	}
	labels := make(map[string]string, len(room.Roster))
	for _, member := range room.Roster {
		label := member.DisplayName
		if label == "" {
			label = member.PlayerID
		}
		labels[member.PlayerID] = label
	}
	submitted := make([]string, 0, len(room.PendingTurn.SubmittedActions))
	waiting := make([]string, 0, len(room.PendingTurn.RequiredPlayers))
	for _, playerID := range room.PendingTurn.RequiredPlayers {
		label := labels[playerID]
		if label == "" {
			label = playerID
		}
		if _, ok := room.PendingTurn.SubmittedActions[playerID]; ok {
			submitted = append(submitted, label)
			continue
		}
		waiting = append(waiting, label)
	}
	return submitted, waiting
}

func roomStageLabel(stageID string) string {
	switch stageID {
	case "lobby":
		return "大厅等待"
	case "round_open":
		return "回合进行中"
	case "round_resolved":
		return "回合已结算"
	default:
		return "未命名阶段"
	}
}

func appendRoomEvent(events []string, event string) []string {
	if strings.TrimSpace(event) == "" {
		return events
	}
	events = append(events, event)
	if len(events) <= 3 {
		return events
	}
	return append([]string(nil), events[len(events)-3:]...)
}

func (b RuntimeBridge) roomHasPendingTurn(ctx context.Context, runtimeID string) (bool, error) {
	var ready bool
	err := b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, runtimeID)
		if err != nil {
			return err
		}
		if !ok {
			return repository.ErrSnapshotNotFound
		}
		room, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](snapshot)
		if err != nil {
			return err
		}
		ready = room.PendingTurn != nil
		return nil
	})
	return ready, err
}

func pruneStaleRoomMembers(room types.MultiplayerRoom, now time.Time) types.MultiplayerRoom {
	if len(room.Roster) == 0 {
		return room
	}

	cutoff := now.Add(-roomMemberTTL).Unix()
	keptRoster := make([]types.RoomMember, 0, len(room.Roster))
	activePlayers := make(map[string]struct{}, len(room.Roster))
	for _, member := range room.Roster {
		if member.LastActiveUnix != 0 && member.LastActiveUnix < cutoff {
			continue
		}
		keptRoster = append(keptRoster, member)
		activePlayers[member.PlayerID] = struct{}{}
	}
	room.Roster = keptRoster

	if len(room.HiddenState) > 0 {
		filteredHidden := make(map[string]types.PrivateState, len(room.HiddenState))
		for playerID, state := range room.HiddenState {
			if _, ok := activePlayers[playerID]; ok {
				filteredHidden[playerID] = state
			}
		}
		room.HiddenState = filteredHidden
	}

	if room.PendingTurn != nil {
		required := make([]string, 0, len(room.PendingTurn.RequiredPlayers))
		for _, playerID := range room.PendingTurn.RequiredPlayers {
			if _, ok := activePlayers[playerID]; ok {
				required = append(required, playerID)
			}
		}
		submitted := make(map[string]types.PendingAction, len(room.PendingTurn.SubmittedActions))
		for playerID, action := range room.PendingTurn.SubmittedActions {
			if _, ok := activePlayers[playerID]; ok {
				submitted[playerID] = action
			}
		}
		room.PendingTurn.RequiredPlayers = required
		room.PendingTurn.SubmittedActions = submitted
		if len(required) < minRoomPlayers {
			room.PendingTurn = nil
			room.Status = types.RuntimeStatusLobby
		}
	}

	return room
}

func (b RuntimeBridge) resumeExecution(ctx context.Context, executionID string) error {
	for step := 0; step < b.maxResumeSteps(); step++ {
		execution, err := b.loadExecution(ctx, executionID)
		if err != nil {
			return err
		}
		if isTerminalExecutionStage(execution.Stage) {
			return nil
		}
		if err := b.Kernel.Resume(ctx, executionID); err != nil {
			return err
		}
	}
	return nil
}

func (b RuntimeBridge) loadExecution(ctx context.Context, executionID string) (types.ExecutionRecord, error) {
	var execution types.ExecutionRecord
	err := b.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		loaded, err := tx.LoadExecution(ctx, executionID)
		if err != nil {
			return err
		}
		execution = loaded
		return nil
	})
	return execution, err
}

func (b RuntimeBridge) maxResumeSteps() int {
	if b.MaxResumeSteps > 0 {
		return b.MaxResumeSteps
	}
	return 8
}

func (b RuntimeBridge) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

func isTerminalExecutionStage(stage types.ExecutionStage) bool {
	switch stage {
	case types.ExecutionCommitted, types.ExecutionDelivered, types.ExecutionFailed:
		return true
	default:
		return false
	}
}

var _ FreeChatResponder = (*RuntimeBridge)(nil)
var _ ProjectConsultResponder = (*RuntimeBridge)(nil)
var _ SoloSceneResponder = (*RuntimeBridge)(nil)
var _ RoomSceneResponder = (*RuntimeBridge)(nil)
