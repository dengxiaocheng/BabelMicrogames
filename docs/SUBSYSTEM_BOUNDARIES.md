# 子系统边界

本文定义主要子系统之间的职责划分和接口边界。

它的目标是避免系统越长越大之后变得模糊、重叠和不可维护。

## 边界规则

如果某项职责可以被清楚地归属到某个子系统，就不应在别的子系统里以 ad hoc 方式重复实现。

## 1. Ingress Gateway

### 拥有

- inbound transport parsing
- signature verification
- request normalization
- quick acknowledgment behavior
- transport-specific correlation data
- 玩家菜单点击到快捷 command 的归一化
- transport-local 会话模式记忆与反馈落盘

### 不拥有

- canonical runtime state
- mode execution
- deterministic settlement
- delivery retry state
- Babel/C++ 稳定联机核心的长期规则所有权

## 2. Control Plane

### 拥有

- health reporting
- config surface
- admin command handling
- operational inspection endpoint
- operator 触发的 recovery / queue inspection 行为

### 不拥有

- gameplay state transition
- scene settlement
- LLM narrative decision

## 3. Runtime Kernel

### 拥有

- execution acceptance
- idempotency
- lease
- runtime instance loading
- persistence ordering
- stage progression
- repository transaction boundary
- recovery entrypoint
- 向 mode module 分派执行

### 不拥有

- transport parsing
- gameplay rule
- prompt composition 细节
- transport-specific outbound formatting

## 4. Mode Router and Mode Modules

### 拥有

- route-to-mode resolution
- mode command parsing
- mode policy
- mode 对 deterministic core、agent task、projection 的具体使用方式
- `free_chat`、`project_consult`、`solo_scene`、`room_scene` 等用户可见行为

### 不拥有

- transport validation
- global execution ordering
- 低层 deterministic algorithm
- durable retry 语义

## 5. Deterministic Core

### 拥有

- time progression
- map logic
- relationship propagation
- inventory / resource update
- deterministic world rule
- turn / action settlement
- Babel/C++ 已稳定模块的规则执行所有权

### 不拥有

- narration
- transport decision
- admin logic
- requirement publishing
- 为了方便而在 `online` 里重复实现 Babel 已稳定的核心模块

## 6. Agent Supervisor

### 拥有

- agent task routing
- worker budget / timeout policy
- artifact collection
- operational memory generation
- tool access policy

### 不拥有

- canonical state mutation
- transport plumbing
- global delivery retry logic

## 7. Projection

### 拥有

- 可见文本生成输入
- scene projection assembly
- operator summary projection
- visibility filtering

### 不拥有

- canonical settlement
- transport retry
- mode routing

## 8. Delivery

### 拥有

- outbound message formatting
- delivery queue
- retry
- send bookkeeping

### 不拥有

- canonical runtime state
- deterministic settlement
- input routing

## 9. Repository

### 拥有

- runtime instance persistence
- snapshot persistence
- execution record persistence
- event persistence
- task persistence
- projection persistence
- delivery job persistence

### 不拥有

- transport interpretation
- gameplay rule
- narration

## 10. Recovery and Replay

### 拥有

- stale execution inspection
- lease expiry handling
- replay / restart safety check
- resumable stage selection

### 不拥有

- transport parsing
- gameplay rule authorship
- manual product policy

## 11. Requirement Registry

### 拥有

- ruleset version
- prompt pack version
- content constraint
- validated gameplay asset revision
- experiment-to-asset traceability
- 仓库内 `requirements/registry/`、`requirements/rulesets/`、`requirements/prompt-packs/`、`requirements/content-constraints/`、`requirements/gameplay-assets/`、`requirements/experiments/` 的版本化资产集合
- `internal/requirementregistry/` 中面向 runtime 的 filesystem loader / resolver

### 不拥有

- live transport handling
- canonical runtime mutation
- ad hoc operator note

## 12. Godot Export

### 拥有

- Godot-facing scene projection shape
- client-facing export packaging
- 对已验证 real-time behavior 的 requirement mapping

### 不拥有

- gameplay truth
- execution ordering
- deterministic state mutation

## 所有权总结

系统应始终维持以下顶层真相：

- ingress 拥有 transport normalization
- control plane 拥有运维入口
- kernel 拥有 execution authority
- mode module 拥有用户可见行为
- deterministic core 拥有规则执行；当 Babel/C++ 已存在稳定实现时，`online` 只拥有编排与验证适配，不拥有重复实现权
- agent supervisor 拥有 worker tasking
- projection 拥有可见 view
- delivery 拥有 outbound retry
- repository 拥有 durable state
- recovery 拥有 survivability
- requirement registry 拥有 validated gameplay asset
- Godot export 拥有 client-facing scene packaging，而不是 simulation truth
