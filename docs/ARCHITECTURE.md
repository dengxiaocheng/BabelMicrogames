# 架构概览

这个仓库现在应被视为一次设计重置后的运行时，而不是对现有包结构的延续性修补。

当前代码里仍然可能包含有价值的脚手架，但目标架构已经不再由现有包名决定。

## 设计立场

- 继承产品行为，不继承旧包布局
- 只保留一个权威运行时内核
- deterministic mutation 与 agent work 明确分离
- projection 和 delivery 不混入 settlement
- requirement assets 必须是一等实体，不能只停留在 prose
- `online` 侧优先承担 Godot 替代验证、服务器编排和 projection 产出，不把 Babel/C++ 已稳定的 deterministic core 全量重写进 Go
- `online` 的玩家面应优先把原本 Godot / 场景侧的可视化交互压成文字化角色扮演与 scene projection，用于快速验证 requirement 和玩法节奏
- 单人模式和全局模式都可以使用 `Claude Code` 主会话与主文档做 working memory，但关键操作和规则推进仍应逐步落回 `Go host + Babel/C++ core`
- AI 可以帮助提出和试探机制，但不应独占完整机制限制；真正的数值、系统约束和可执行规则仍应逐步回到 Go / Babel-C++

## 目标运行时形态

- `ingress adapters`
  负责校验 transport 细节，并输出统一的 `InboundEnvelope`

- `kernel`
  负责 acceptance、idempotency、leases、execution ordering、persistence boundary 和 recovery entrypoint

- `mode router`
  决定当前输入归哪个 mode module 处理

- `mode modules`
  实现 `free_chat`、`project_consult`、`solo_scene`、`room_scene` 以及未来更多 runtime-backed behavior

- `deterministic core`
  在 versioned ruleset 之下执行权威状态变更；当 Babel/C++ 已存在稳定核心时，`online` 应优先通过 requirement / host adapter 复用，而不是继续扩张 Go 的重复实现

- `scene host adapter`
  `solo_scene / room_scene` 面向 Babel/C++ 稳定核心的 Go 侧固定接缝；它显式接收 runtime、resolved requirements、当前 snapshot 和 action，并只返回更新后的 deterministic scene state。当前已同时存在本地 Go fallback host 与 shared-library loader skeleton，后续 Babel/C++ 只需要对齐同一条 `SceneHost`/`babel_sim_step` 边界。运行态若显式请求 shared-library（包括 `@collab` 产物解析），就不应静默回退到 Go fallback，而应把缺失或验证失败作为显式接入错误暴露出来。

- `agent supervisor`
  以显式、可恢复的任务形式运行 advisory / interpretation / narrative 工作

- `projection`
  构建玩家可见输出，以及未来面向 Godot 的结构化场景载荷

- `operational memory artifact`
  为长会话 worker 派生 `primary_context.md`、`session_manifest.json`、`scene_state.json` 这类文件化上下文；它们只能从 runtime snapshot / projection 派生，不能反向成为 canonical truth

- `delivery`
  负责 outbound transport job 和 retry state

- `repository`
  存储 runtime instance、snapshot、execution record、task、projection 和 delivery job

- `recovery`
  通过与正常执行相同的 stage machine 恢复未完成 execution

- `requirement registry`
  当前已由仓库内 `requirements/` 目录和 `internal/requirementregistry` 的 filesystem loader / resolver 承载 versioned ruleset、prompt pack、content constraint、gameplay asset 和 experiment trace 的基础资产集合

## 标准执行流

1. ingress adapter 校验 transport 输入并生成 `InboundEnvelope`
2. kernel 创建或复用 execution record，并写入 idempotency / lease 信息
3. kernel 加载 runtime snapshot，并解析归属的 mode module
4. mode module 把 envelope 转成 mode command，并决定 deterministic work、agent task 和 projection policy
5. deterministic core 执行 canonical state 变更
6. agent supervisor 运行或调度可选任务，但不持有 state authority
7. projection 生成可见输出和可选的结构化 scene payload
8. repository 提交新 snapshot、journal entry、projection 和 delivery job
9. delivery worker 发布 outbound message；recovery 可以从任何未完成 stage 接着执行
10. requirement registry 以 versioned asset 形式为 mode module、deterministic core 和后续 loader 提供可解析的规则与内容输入

当前微信入口里，`solo_scene / room_scene` 在返回 transport-facing 回复后，也会把当前 snapshot 派生写入 `primary_context.md / session_manifest.json / scene_state.json / recent_summary.md`。这些文件会带出 runtime 已绑定的 `ruleset / prompt_pack / gameplay_asset` 引用，作为后续 Babel/C++ host adapter 和长会话 worker 的共同读取入口；但它们仍然只服务于 working memory 和协作读取，不参与 settlement、idempotency 或 recovery 的权威判断。

当前最小实现里，kernel 已经会在执行前解析 runtime 上声明的 `ruleset_id`、`prompt_pack_id` 和可选的 `gameplay_asset_id`，并把 resolved bundle 传进 `ModeExecutionInput`。这还不是最终 loader 形态，但已经把 requirement foundation 从“文档占位”推进成了可执行接口。

## 直接含义

实现应逐步收敛到 `kernel / mode / task / projection / repository` 这条模型上。

`coordinator`、`solo`、`multiplayer` 这类现有包只有在能帮助迁移到目标架构时才有价值；它们本身不是架构契约。
