# Recovery Checkpoint 模型

## 目的

定义新系统如何在重启、崩溃或部分失败后继续运行，同时尽量避免明显的用户中断。

这个模型只服务于新运行时。

## 原则

1. 先持久化 intent，再开始执行
2. 在副作用之前先持久化 checkpoint
3. 最终 canonical state 必须先于 delivery 落盘
4. Delivery 必须可重放
5. Recovery 判断必须能由机器决定，而不是靠人工猜测

## 生命周期

每条有意义的执行路径都应遵循：

1. receive input
2. persist input event
3. acquire runtime lock
4. create checkpoint
5. execute deterministic step
6. persist canonical state
7. render output
8. persist render frame
9. deliver output
10. mark checkpoint committed

## Checkpoint 状态

```yaml
CheckpointStatus:
  pending_input
  validating
  simulating
  persisting_state
  rendering
  delivering
  committed
  failed
```

## Checkpoint 记录

```yaml
RuntimeCheckpoint:
  checkpoint_id: string
  runtime_type: enum(solo, multiplayer)
  runtime_id: string
  state_version_before: int
  state_version_after: int|null
  input_event_id: string
  step_name: string
  status: CheckpointStatus
  resume_policy: enum(retry_step, rerender_only, redeliver_only, manual_intervention)
  lease_owner: string|null
  lease_expires_at: timestamp|null
  created_at: timestamp
  updated_at: timestamp
```

## Input Event 记录

```yaml
InputEvent:
  event_id: string
  runtime_type: enum(solo, multiplayer)
  runtime_id: string
  actor_id: string
  idempotency_key: string
  payload: json
  received_at: timestamp
```

## Delivery 记录

```yaml
DeliveryRecord:
  delivery_id: string
  checkpoint_id: string
  frame_id: string
  recipient_id: string
  channel: string
  status: enum(pending, sent, acknowledged, failed)
  sent_at: timestamp|null
  acknowledged_at: timestamp|null
```

## Recovery 决策

### 当 checkpoint status 是 `pending_input`

- input 已持久化
- 还没有真正执行
- 可以安全地从 validation 继续

### 当 status 是 `simulating`

- deterministic step 可能还没有提交
- 应从 `state_version_before` 重新执行

### 当 status 是 `persisting_state`

- 这里是最敏感的 commit boundary
- 需要检查 `state_version_after` 是否已经存在
- 如果还没真正提交，就从旧 state 重跑

### 当 status 是 `rendering`

- 状态已经成为 canonical truth
- 应 rerender，或者优先复用已有 frame

### 当 status 是 `delivering`

- 不要重新计算状态
- 直接 redeliver 已保存 frame

## Runtime Lock 模型

使用基于 lease 的 runtime lock：

```yaml
RuntimeLease:
  runtime_id: string
  owner_id: string
  lease_expires_at: timestamp
```

如果进程死掉，下一位 worker 可以在 lease 到期后接管。

## 重启流程

服务启动时：

1. 加载所有未 `committed` 的 active checkpoint
2. 按 runtime id 排序
3. 重新获取或抢占过期 lease
4. 检查 checkpoint status
5. 应用 recovery policy
6. 在日志里记录 recovery 动作

## 健康暴露

health endpoint 至少应暴露：

- active checkpoint 数量
- recoverable runtime 数量
- 当前持有的 runtime lock 数量
- stale checkpoint 数量
- stale delivery 数量
- 上次 recovery sweep 时间

## 用户体验目标

### Solo

- 重启后，下一条用户消息应能继续同一个 session
- 如果 scene 已经渲染过，可以直接重放，而不是重新生成

### Multiplayer

- room 应在相同 stage 重开
- 已提交的玩家选择不能丢
- 已经 settled 的 turn 不能以不同结果再算一遍

## 必要派生元数据

为快速恢复，应额外保存以下小摘要：

- latest canonical state version
- latest render frame id
- latest committed turn / action id
- pending recipients for undelivered output

## 反模式

不要：

- 仅根据内存线程 id 推断 recovery
- 依赖模型对话作为 canonical continuity
- 在已有 saved frame 时重新生成 narration
- redeliver 时不带 dedupe marker

## MVP 恢复范围

### Phase 1

- solo action recovery
- last frame redelivery
- health endpoint summaries

### Phase 2

- multiplayer turn recovery
- per-player pending action recovery
- room reopen continuity

### Phase 3

- partial fanout recovery
- multi-node lease management
- replay tooling and operator repair console
