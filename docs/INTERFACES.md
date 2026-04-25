# 核心接口

本文记录 redesign-reset 之后运行时的目标接口契约。

当前代码里的包内接口仍然只是脚手架，不应被误读为最终 API surface。

## Kernel

```go
type Kernel interface {
    Accept(ctx context.Context, env InboundEnvelope) (ExecutionTicket, error)
    Resume(ctx context.Context, executionID string) error
}
```

## Mode Router

```go
type ModeRouter interface {
    Resolve(ctx context.Context, runtime RuntimeRecord, env InboundEnvelope) (ModeModule, error)
}
```

## Mode Module

```go
type ModeModule interface {
    ModeID() ModeID
    BuildCommand(ctx context.Context, runtime RuntimeView, env InboundEnvelope) (ModeCommand, error)
    Execute(ctx context.Context, input ModeExecutionInput) (ModeExecutionResult, error)
}
```

`ModeExecutionResult` 至少需要能描述：

- 要应用的 deterministic work
- 需要启动或等待的 agent task
- 要回折进 runtime-controlled state 的 artifact
- 要追加的 event
- projection hint
- delivery intent

## Deterministic Engine

```go
type DeterministicEngine interface {
    Apply(ctx context.Context, req DeterministicRequest) (DeterministicResult, error)
}
```

## Scene Host Adapter

```go
type SceneHost interface {
    StepSolo(ctx context.Context, input SoloStepInput) (types.SoloSession, error)
    StepRoom(ctx context.Context, input RoomStepInput) (types.MultiplayerRoom, error)
}
```

`SceneHost` 是当前 `online` 仓库为 Babel/C++ 宿主适配预留的 Go 侧固定边界。

要求：

- 输入必须显式带上 `RuntimeRecord` 和 resolved `RuntimeRequirements`
- C++ 侧只负责 deterministic scene state 推进，不负责 transport / projection / delivery
- Go 侧可以先用本地 fallback host 实现同一接口，后续再替换成 Babel/C++ adapter

当前约定的 shared-library loader skeleton 使用窄 `C ABI`：

```c
int babel_sim_step(
  const uint8_t* request_bytes,
  size_t request_len,
  uint8_t** response_bytes,
  size_t* response_len
);

void babel_sim_free(void* ptr);
```

其中：

- request / response 使用 JSON bytes，且必须携带 `contract_version`
- `operation` 当前至少包括 `solo_step` 与 `room_step`
- Go 侧 adapter 负责把 `RuntimeRecord`、resolved `RuntimeRequirements`、当前 scene state 和 action 打包进 request
- `babel_sim_step` 返回非 `0` 时，response buffer 可承载 UTF-8 error text，Go 侧仍需调用 `babel_sim_free`
- 当前仓库已提供标准 fixture，可通过 `go run ./cmd/babel-dev scene-host-fixture --mode <solo_step|room_step> --kind <request|response>` 直接导出给 Babel/C++ 对齐
- 当 Babel/C++ 产出 `.so` 后，Go 侧应通过 `go run ./cmd/babel-dev verify-scene-host-library --library <path> --mode all` 对共享库跑同一套 fixture 验证，而不是手工比对 JSON
- 当运行态配置了 `BABEL_SCENE_CORE_LIBRARY` 时，接入层启动前也必须先对共享库执行同一套 fixture 验证；`/healthz` 需显式暴露当前 `scene_host_mode / scene_host_source / scene_host_verified / scene_host_library / scene_host_contract`
- 当前运行态还支持 `BABEL_SCENE_CORE_LIBRARY=@collab`：此时 `babel-wechatd` 会从 collaboration MCP 的最新 `scene_host_library` artifact 解析共享库路径，以便 `babel-cpp` 会话发布产物后，`online` 侧无需再靠手工复制路径；若 artifact 缺失，则启动必须直接失败，不能静默回退到 Go fallback

## Agent Supervisor

```go
type AgentSupervisor interface {
    StartTasks(ctx context.Context, execution ExecutionRecord, tasks []AgentTaskSpec) error
    Collect(ctx context.Context, executionID string) ([]AgentArtifact, error)
}
```

## Projector

```go
type Projector interface {
    Project(ctx context.Context, input ProjectionInput) (ProjectionResult, error)
}
```

## Dispatcher

```go
type Dispatcher interface {
    Enqueue(ctx context.Context, plan DeliveryPlan) error
}
```

## Repository

```go
type Repository interface {
    RunExecutionTx(ctx context.Context, fn func(ctx context.Context, tx ExecutionTx) error) error
}
```

## Recovery

```go
type Recovery interface {
    ResumeStalled(ctx context.Context, limit int) (RecoveryReport, error)
}
```

## Requirement Registry

```go
type RequirementRegistry interface {
    ResolveRuleset(ctx context.Context, rulesetID string) (RulesetBundle, error)
    ResolvePromptPack(ctx context.Context, promptPackID string) (PromptPackBundle, error)
    ResolveGameplayAsset(ctx context.Context, assetID string) (GameplayAssetBundle, error)
    ResolveContentConstraint(ctx context.Context, constraintID string) (ContentConstraintBundle, error)
    ResolveExperimentTrace(ctx context.Context, traceID string) (ExperimentTraceBundle, error)
}
```

## 关键契约含义

- transport adapter 只产出 `InboundEnvelope`，不直接调用 mode logic
- mode module 拥有行为定义；kernel 拥有 execution ordering 和 persistence ordering
- deterministic state change 必须走 `DeterministicEngine`
- agent work 只能返回 artifact，不能直接写 canonical state
- projection / delivery 保持在 deterministic settlement 之外
- recovery 复用 kernel stage，不再创造一套平行逻辑
- requirement registry 当前已经能从仓库内 `requirements/` 目录解析 versioned asset，并由 kernel 显式传入 `ModeExecutionInput.Requirements`
