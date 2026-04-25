# 存储模型

存储模型应以 `execution-centric` 为中心，而不是以包或 handler 为中心。

主持久化单元不应是 `solo_session` 或 `multiplayer_room` 这样的现有形态名。

真正的一等持久化单元是：

- runtime instance
- versioned snapshot
- execution record
- 由 execution 派生出来的 artifact 与 delivery job

## 核心记录

- `runtime_instances`
  长生命周期 runtime aggregate 的 identity、mode、lifecycle status、head version 和 ownership metadata

- `runtime_snapshots`
  runtime instance 的 versioned canonical state payload

- `execution_records`
  一个已接受的 inbound action 或 scheduled step，沿着持久化 execution stage 前进，并携带 settlement / projection / delivery 之间恢复所需的 handoff metadata

- `execution_leases`
  in-flight execution 的 lease owner 与 expiry

- `runtime_events`
  执行过程中产生的 append-only fact

## Agent 与 Projection 相关记录

- `agent_tasks`
  面向 advisory / interpretive / narrative 工作的显式 task record

- `agent_artifacts`
  从 agent task 收集的输出，包括 summary、note 和结构化 advisory payload

- `projection_frames`
  基于 canonical state 和已批准 artifact 派生出的玩家可见或 operator 可见 render output

- `scene_projections`
  未来给 Godot 或内部工具消费的结构化 scene payload

- `delivery_jobs`
  可重试的 outbound transport job

- `delivery_attempts`
  每个 delivery job 的发送审计轨迹

## Requirement Asset 相关记录

- `rulesets`
- `prompt_packs`
- `content_constraints`
- `requirement_assets`
- `requirement_revisions`

这些记录都应有版本，并能被 runtime execution 显式加载。

在数据库化 registry 真正落地前，仓库内 `requirements/` 目录应作为这些 versioned asset 的基础来源，并保持与 runtime revision、测试和 changelog 可追踪。当前 runtime 侧的最小 loader / resolver 已通过 `internal/requirementregistry` 消费这组资产。

## 设计规则

- `runtime_instances` 是所有 mode 的 canonical identity layer
- `runtime_snapshots` 保存 versioned canonical state；mode-specific shape 应放在 typed payload 或 schema version 里，而不是过早拆成多个顶层表
- `execution_records` 是已接受工作项的权威 lifecycle journal
- `runtime_events` 必须 append-only 并可 replay
- `agent_tasks` 和 `agent_artifacts` 属于 operational data，不是 canonical state
- `projection_frames` 和 `scene_projections` 属于 replayable view，不是权威真相
- `delivery_jobs` 拥有 outbound retry 语义
- requirement asset 一旦进入验证流程，就必须能关联到测试和 runtime revision

## 反模式

- 不允许 transport handler 直接改状态
- 不允许只存在于 agent memory 文件中的 hidden state
- 不允许只为 recovery 单独造一套语义不同的表
- 不允许默认认为 solo / multiplayer 必须有两套完全不同的主存储模型

在真正落地数据库前，应基于这份模型生成 migration，并对 replay、recovery、delivery 需求做审查。
