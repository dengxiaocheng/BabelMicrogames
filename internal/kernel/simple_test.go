package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/delivery"
	"babel-runtime/internal/kernel"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/projection"
	"babel-runtime/internal/requirementregistry"
	"babel-runtime/internal/repository"
)

type fakeModule struct {
	id          types.ModeID
	commandType string
	executeFn   func(types.ModeExecutionInput) (types.ModeExecutionResult, error)
}

func (m fakeModule) ModeID() types.ModeID {
	return m.id
}

func (m fakeModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	return types.ModeCommand{
		CommandType: m.commandType,
		Text:        env.Text,
	}, nil
}

func (m fakeModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	_ = ctx
	if m.executeFn != nil {
		return m.executeFn(input)
	}
	return types.ModeExecutionResult{
		NextStage:           types.ExecutionSettled,
		RuntimeVersionDelta: 1,
	}, nil
}

func TestAcceptCreatesPlannedExecutionAndIsIdempotent(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:   "runtime-1",
		ModeID:      types.ModeFreeChat,
		HeadVersion: 3,
		Status:      types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeFreeChat,
		commandType: "chat.user_text",
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
	}

	first, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		EnvelopeID:     "env-1",
		IdempotencyKey: "idem-1",
		Text:           "你好",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if first.Reused {
		t.Fatalf("expected first accept to create execution")
	}
	if first.Stage != types.ExecutionPlanned {
		t.Fatalf("expected planned stage, got %q", first.Stage)
	}

	second, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		EnvelopeID:     "env-1-repeat",
		IdempotencyKey: "idem-1",
		Text:           "你好",
	})
	if err != nil {
		t.Fatalf("second Accept returned error: %v", err)
	}
	if !second.Reused {
		t.Fatalf("expected second accept to reuse execution")
	}
	if second.ExecutionID != first.ExecutionID {
		t.Fatalf("expected reused execution id %q, got %q", first.ExecutionID, second.ExecutionID)
	}

	snapshot := repo.Snapshot()
	if len(snapshot.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(snapshot.Executions))
	}
	execution := snapshot.Executions[first.ExecutionID]
	if execution.CommandType != "chat.user_text" {
		t.Fatalf("expected command type chat.user_text, got %q", execution.CommandType)
	}
	if execution.ActorID != "" {
		t.Fatalf("expected empty actor id when envelope user is empty, got %q", execution.ActorID)
	}
	if len(snapshot.Events) != 2 {
		t.Fatalf("expected 2 lifecycle events, got %d", len(snapshot.Events))
	}
}

func TestAcceptPersistsEnvelopeActorID(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeRoomScene,
		Status:    types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeRoomScene,
		commandType: "room.user_text",
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		UserID:         "u2",
		IdempotencyKey: "idem-actor",
		Text:           "发言",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	execution := repo.Snapshot().Executions[ticket.ExecutionID]
	if execution.ActorID != "u2" {
		t.Fatalf("expected actor id u2, got %q", execution.ActorID)
	}
}

func TestResumeAdvancesOneStageAndUpdatesRuntimeVersion(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:   "runtime-1",
		ModeID:      types.ModeSoloScene,
		HeadVersion: 7,
		Status:      types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeSoloScene,
		commandType: "scene.advance",
		executeFn: func(input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
			if input.Command.CommandType != "scene.advance" {
				t.Fatalf("expected scene.advance command, got %q", input.Command.CommandType)
			}
			return types.ModeExecutionResult{
				NextStage:           types.ExecutionSettled,
				RuntimeVersionDelta: 2,
				Events: []types.RuntimeEvent{
					{EventID: "evt-domain-1", Kind: "scene.applied"},
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
		Now: func() time.Time {
			return time.Unix(1700000100, 0)
		},
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-2",
		Text:           "继续",
		RouteHint:      string(types.ModeSoloScene),
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}

	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}

	snapshot := repo.Snapshot()
	runtime := snapshot.Runtimes["runtime-1"]
	if runtime.HeadVersion != 9 {
		t.Fatalf("expected runtime head version 9, got %d", runtime.HeadVersion)
	}
	if len(snapshot.Snapshots) != 0 {
		t.Fatalf("expected no snapshots for fake module path, got %d", len(snapshot.Snapshots))
	}
	execution := snapshot.Executions[ticket.ExecutionID]
	if execution.Stage != types.ExecutionSettled {
		t.Fatalf("expected settled execution, got %q", execution.Stage)
	}
	if execution.LeaseOwner == "" {
		t.Fatalf("expected settled execution to retain lease owner")
	}
	if len(snapshot.Events) != 4 {
		t.Fatalf("expected 4 events after accept+resume, got %d", len(snapshot.Events))
	}
}

func TestResumePersistsFailedExecutionState(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeProjectConsult,
		Status:    types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeProjectConsult,
		commandType: "consult.ask",
		executeFn: func(input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
			return types.ModeExecutionResult{}, errors.New("agent unavailable")
		},
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-3",
		Text:           "帮我审文档",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	err = engine.Resume(context.Background(), ticket.ExecutionID)
	if err == nil || err.Error() != "agent unavailable" {
		t.Fatalf("expected resume error agent unavailable, got %v", err)
	}

	snapshot := repo.Snapshot()
	execution := snapshot.Executions[ticket.ExecutionID]
	if execution.Stage != types.ExecutionFailed {
		t.Fatalf("expected failed execution stage, got %q", execution.Stage)
	}
	if execution.LastError == "" {
		t.Fatalf("expected failed execution to persist last error")
	}
}

func TestAcceptFailsWhenRuntimeRequirementCannotBeResolved(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:   "runtime-req-missing",
		ModeID:      types.ModeFreeChat,
		RulesetID:   "missing.ruleset",
		Status:      types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeFreeChat,
		commandType: "chat.user_text",
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	engine := kernel.SimpleEngine{
		Repo:         repo,
		Router:       router,
		Requirements: requirementregistry.FilesystemRegistry{RepoRoot: filepath.Join("..", "..")},
	}

	_, err = engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-req-missing",
		IdempotencyKey: "idem-missing-req",
		Text:           "你好",
	})
	if err == nil {
		t.Fatalf("expected Accept to fail when requirement asset is missing")
	}
}

func TestResumePassesResolvedRequirementsIntoModeExecution(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:        "runtime-req-ok",
		ModeID:           types.ModeProjectConsult,
		RulesetID:        "bootstrap.ruleset",
		PromptPackID:     "bootstrap.prompt_pack",
		GameplayAssetID:  "bootstrap.gameplay_asset",
		HeadVersion:      1,
		Status:           types.RuntimeStatusActive,
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeProjectConsult,
		commandType: "consult.ask",
		executeFn: func(input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
			if input.Requirements.Ruleset == nil || input.Requirements.Ruleset.RulesetID != "bootstrap.ruleset" {
				t.Fatalf("expected resolved ruleset, got %+v", input.Requirements.Ruleset)
			}
			if input.Requirements.PromptPack == nil || input.Requirements.PromptPack.PromptPackID != "bootstrap.prompt_pack" {
				t.Fatalf("expected resolved prompt pack, got %+v", input.Requirements.PromptPack)
			}
			if input.Requirements.GameplayAsset == nil || input.Requirements.GameplayAsset.GameplayAssetID != "bootstrap.gameplay_asset" {
				t.Fatalf("expected resolved gameplay asset, got %+v", input.Requirements.GameplayAsset)
			}
			return types.ModeExecutionResult{
				NextStage:           types.ExecutionSettled,
				RuntimeVersionDelta: 1,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	engine := kernel.SimpleEngine{
		Repo:         repo,
		Router:       router,
		Requirements: requirementregistry.FilesystemRegistry{RepoRoot: filepath.Join("..", "..")},
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-req-ok",
		IdempotencyKey: "idem-req-ok",
		Text:           "检查 requirement",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}
}

func TestResumeFailsWhenAgentTaskIsRequestedWithoutSupervisor(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:     "runtime-1",
		ModeID:        types.ModeFreeChat,
		SchemaVersion: 1,
		HeadVersion:   0,
		Status:        types.RuntimeStatusActive,
	})
	snapshot, err := types.EncodeRuntimeSnapshot(
		"runtime-1",
		types.ModeFreeChat,
		0,
		1,
		types.FreeChatState{RuntimeID: "runtime-1"},
		time.Unix(1, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}
	repo.SeedRuntimeSnapshot(snapshot)
	router, err := mode.NewStaticRouter(mode.FreeChatModule{})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-taskless",
		Text:           "你好",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	err = engine.Resume(context.Background(), ticket.ExecutionID)
	if err == nil || err.Error() != "agent supervisor not initialized" {
		t.Fatalf("expected missing supervisor error, got %v", err)
	}

	execution := repo.Snapshot().Executions[ticket.ExecutionID]
	if execution.Stage != types.ExecutionFailed {
		t.Fatalf("expected failed execution stage, got %q", execution.Stage)
	}
}

func TestResumePersistsUpdatedSnapshotFromMode(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:   "runtime-1",
		ModeID:      types.ModeSoloScene,
		HeadVersion: 1,
		Status:      types.RuntimeStatusActive,
	})
	repo.SeedRuntimeSnapshot(types.RuntimeSnapshot{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeSoloScene,
		Version:   1,
		State:     []byte(`{"state_version":1}`),
	})
	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeSoloScene,
		commandType: "scene.advance",
		executeFn: func(input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
			if input.Snapshot == nil {
				t.Fatalf("expected snapshot passed into mode execution")
			}
			return types.ModeExecutionResult{
				NextStage: types.ExecutionSettled,
				Snapshot: &types.RuntimeSnapshot{
					RuntimeID: "runtime-1",
					ModeID:    types.ModeSoloScene,
					Version:   2,
					State:     []byte(`{"state_version":2}`),
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}
	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-4",
		Text:           "继续",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}

	snapshot := repo.Snapshot()
	if snapshot.Runtimes["runtime-1"].HeadVersion != 2 {
		t.Fatalf("expected runtime head version 2, got %d", snapshot.Runtimes["runtime-1"].HeadVersion)
	}
	if snapshot.Snapshots["runtime-1"].Version != 2 {
		t.Fatalf("expected stored snapshot version 2, got %d", snapshot.Snapshots["runtime-1"].Version)
	}
	if snapshot.Executions[ticket.ExecutionID].Stage != types.ExecutionSettled {
		t.Fatalf("expected execution settled after one resume, got %q", snapshot.Executions[ticket.ExecutionID].Stage)
	}
}

func TestResumeProjectsAndQueuesDelivery(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:     "runtime-1",
		ModeID:        types.ModeFreeChat,
		SchemaVersion: 1,
		HeadVersion:   0,
		Status:        types.RuntimeStatusActive,
	})
	snapshot, err := types.EncodeRuntimeSnapshot(
		"runtime-1",
		types.ModeFreeChat,
		0,
		1,
		types.FreeChatState{RuntimeID: "runtime-1"},
		time.Unix(1, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}
	repo.SeedRuntimeSnapshot(snapshot)
	router, err := mode.NewStaticRouter(mode.FreeChatModule{})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:       repo,
		Router:     router,
		Supervisor: &agent.SimpleSupervisor{},
		Projector:  projection.SimpleProjector{DefaultTransport: "wechat"},
		Dispatcher: delivery.QueueDispatcher{Now: func() time.Time { return time.Unix(50, 0) }},
		Now:        func() time.Time { return time.Unix(40, 0) },
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		UserID:         "user-1",
		IdempotencyKey: "idem-proj",
		Text:           "你好",
		Transport:      "wechat",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}

	snap := repo.Snapshot()
	if snap.Executions[ticket.ExecutionID].Stage != types.ExecutionAwaitingArtifacts {
		t.Fatalf("expected awaiting_artifacts stage after first resume, got %q", snap.Executions[ticket.ExecutionID].Stage)
	}
	if len(snap.Frames) != 0 {
		t.Fatalf("expected 0 frames before projection, got %d", len(snap.Frames))
	}
	if len(snap.DeliveryJobs) != 0 {
		t.Fatalf("expected 0 delivery jobs before projection, got %d", len(snap.DeliveryJobs))
	}
	if len(snap.AgentTasks) != 1 {
		t.Fatalf("expected 1 persisted agent task, got %d", len(snap.AgentTasks))
	}
	if len(snap.AgentArtifacts) != 0 {
		t.Fatalf("expected 0 agent artifacts before collection, got %d", len(snap.AgentArtifacts))
	}

	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("second Resume returned error: %v", err)
	}

	snap = repo.Snapshot()
	if snap.Executions[ticket.ExecutionID].Stage != types.ExecutionSettled {
		t.Fatalf("expected settled stage after second resume, got %q", snap.Executions[ticket.ExecutionID].Stage)
	}
	if len(snap.AgentArtifacts) != 1 {
		t.Fatalf("expected 1 collected agent artifact, got %d", len(snap.AgentArtifacts))
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("third Resume returned error: %v", err)
	}

	snap = repo.Snapshot()
	if snap.Executions[ticket.ExecutionID].Stage != types.ExecutionProjected {
		t.Fatalf("expected projected stage after third resume, got %q", snap.Executions[ticket.ExecutionID].Stage)
	}
	if len(snap.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(snap.Frames))
	}
	if snap.Frames[0].Body != "free chat[1]: 你好" {
		t.Fatalf("unexpected frame body %q", snap.Frames[0].Body)
	}
	if len(snap.Executions[ticket.ExecutionID].PendingDelivery) != 1 {
		t.Fatalf("expected 1 pending delivery plan, got %d", len(snap.Executions[ticket.ExecutionID].PendingDelivery))
	}
	if len(snap.DeliveryJobs) != 0 {
		t.Fatalf("expected 0 delivery jobs before delivery step, got %d", len(snap.DeliveryJobs))
	}

	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("fourth Resume returned error: %v", err)
	}

	snap = repo.Snapshot()
	if snap.Executions[ticket.ExecutionID].Stage != types.ExecutionDelivered {
		t.Fatalf("expected delivered stage after fourth resume, got %q", snap.Executions[ticket.ExecutionID].Stage)
	}
	if len(snap.Executions[ticket.ExecutionID].PendingDelivery) != 0 {
		t.Fatalf("expected pending delivery plans cleared after delivery")
	}
	if len(snap.DeliveryJobs) != 1 {
		t.Fatalf("expected 1 delivery job, got %d", len(snap.DeliveryJobs))
	}
	if snap.DeliveryJobs[0].Status != types.DeliveryQueued {
		t.Fatalf("expected queued delivery job, got %q", snap.DeliveryJobs[0].Status)
	}
}

func TestResumeIgnoresOperationalMemoryWriteFailure(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:     "runtime-1",
		ModeID:        types.ModeFreeChat,
		SchemaVersion: 1,
		HeadVersion:   0,
		Status:        types.RuntimeStatusActive,
	})
	snapshot, err := types.EncodeRuntimeSnapshot(
		"runtime-1",
		types.ModeFreeChat,
		0,
		1,
		types.FreeChatState{RuntimeID: "runtime-1"},
		time.Unix(1, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}
	repo.SeedRuntimeSnapshot(snapshot)
	router, err := mode.NewStaticRouter(mode.FreeChatModule{})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	errFile := filepath.Join(t.TempDir(), "memory-root-file")
	if err := os.WriteFile(errFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var memoryErr error
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
		Supervisor: &agent.SimpleSupervisor{
			MemoryRoot: errFile,
			OnMemoryWriteError: func(err error) {
				memoryErr = err
			},
		},
	}

	ticket, err := engine.Accept(context.Background(), types.InboundEnvelope{
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-memory-fail",
		Text:           "你好",
	})
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("first Resume returned error: %v", err)
	}
	if err := engine.Resume(context.Background(), ticket.ExecutionID); err != nil {
		t.Fatalf("second Resume returned error: %v", err)
	}

	snap := repo.Snapshot()
	if snap.Executions[ticket.ExecutionID].Stage != types.ExecutionSettled {
		t.Fatalf("expected execution settled, got %q", snap.Executions[ticket.ExecutionID].Stage)
	}
	if snap.Snapshots["runtime-1"].Version != 1 {
		t.Fatalf("expected snapshot version 1, got %d", snap.Snapshots["runtime-1"].Version)
	}
	if memoryErr == nil {
		t.Fatalf("expected operational memory write error callback")
	}
}
