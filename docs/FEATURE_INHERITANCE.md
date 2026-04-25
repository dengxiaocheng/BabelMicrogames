# 能力继承

这个项目继承的是旧 Python 系统的产品能力，不是它的实现形状。

不会继承旧系统的 process model、prompt plumbing、patch 历史，也不会因为迁移方便而保留阻碍重建的早期脚手架。

基本规则是：

- 继承用户可见能力
- 用 Go 重新设计每一层
- 把 deterministic state 留在 LLM 之外
- 只在热路径被证明之后再考虑 C++

## 迁移原则

对于每个 legacy feature：

1. 先提取真实的用户可见行为
2. 把它放入新的目标子系统
3. 在不携带旧债务的前提下重建
4. 用测试证明之后再宣布完成

## 能力地图

下面的能力地图反映的是 redesign-reset 之后的目标架构。

它不代表当前包布局就是最终形态。

### Runtime Core

- `single-player roleplay`
  新归属：`kernel`、`solo_scene` mode module、deterministic core、projection、repository、recovery
  状态：已部分启动
  当前覆盖：基于 kernel 的 `solo_scene` mode 已存在，具备 runtime snapshot、可恢复 execution record，以及对既有 deterministic settlement 的复用；这些 Go 侧 deterministic 代码当前应被视为验证脚手架，而不是 Babel/C++ 稳定核心的长期替代品。微信新服务现已接通 `单人角色 -> solo_scene` 的玩家链路，支持直接自由文本推进，以及通过 `决 / 想` 按钮把常见动作压成 transport-facing command；同时会把当前 scene snapshot 派生为 `primary_context.md / session_manifest.json / scene_state.json`，并带出 runtime 绑定的 `ruleset / prompt_pack / gameplay_asset` 引用，作为长会话工作记忆入口，但不改变 canonical state authority。Go 侧还新增了 `SceneHost` 与 `babel_sim_step` shared-library loader skeleton，便于后续直接切换到 Babel/C++ 导出的 deterministic scene core

- `multiplayer roleplay`
  新归属：`kernel`、`room_scene` mode module、deterministic core、projection、repository、recovery
  状态：已部分启动
  当前覆盖：基于 kernel 的 `room_scene` mode 已存在，具备 runtime snapshot、staged execution、action submission，以及通过共享执行流完成 turn-close 复用；微信新服务已补上最小 `联机` 入口，可在共享 room runtime 中做加入、状态查看和文本行动提交。当前 Go 验证链已收束为“至少两名活跃玩家才开局”，并会自动剔除超时未活跃成员；玩家当前可见 lobby / round / recent event 文本也已在 Go 侧生成，用于快速验证房间体验，且会额外显示已提交/待提交玩家和上轮结算摘要。共享房间的当前快照同样会派生出 `primary_context.md / session_manifest.json / scene_state.json`，并携带 requirement 引用，便于后续长会话协作读取同一份房间上下文。Go 侧 `SceneHost`/shared-library loader skeleton 已就位，后续可以在不改 mode/kernel 主链的前提下切向 Babel/C++ 宿主。长期稳定结算所有权仍应收敛到 Babel/C++ 核心

- `state recovery after restart`
  新归属：`repository`、`kernel`、`recovery`、replay tooling、restart simulation
  状态：已部分启动
  当前覆盖：新路径上已经有 execution record、lease、stale-execution scan 和 kernel 驱动的 resume supervisor

### LLM Orchestration

- `free chat`
  新归属：`free_chat` mode module、agent supervisor、projection、operational memory artifact
  状态：已部分启动
  当前覆盖：`free_chat` mode 已走通 staged kernel execution，具备 persisted agent task、artifact 回流、projection、delivery job、WeChat runtime bridge，以及可选的 file-backed operational memory；worker policy 和真实 tool-backed narration 仍然只是脚手架

- `project consultation`
  新归属：`project_consult` mode module、agent supervisor、tool adapter、requirement registry
  状态：已部分启动
  当前覆盖：`project_consult` mode 已走通 staged kernel execution，具备 persisted agent task、artifact 回流、projection、delivery job、WeChat runtime bridge，以及可选的 file-backed operational memory；真实 consult tooling 和更丰富的 agent policy 尚未接线

- `mode routing`
  新归属：`ingress adapter`、`mode router`、runtime instance metadata
  状态：已部分启动
  当前覆盖：新 kernel 已有 static mode router 和基于 route hint 的执行路径

- `model routing and fast/slow path policy`
  新归属：`agent supervisor`、mode policy、control-plane config
  状态：尚未重建

### Transport / Product Surface

- `wechat webhook handling`
  新归属：`ingress/wechat` adapter 加 delivery dispatcher
  状态：已部分启动
  当前覆盖：XML parsing、signature verification、仅面向用户模式的 route normalization、project-consult classification 和 kernel-backed runtime bridge 已存在；当 projector / dispatcher 被配置时，staged execution 已可投影回复并生成 delivery job；WeChat 测试号不再承载管理员命令入口，旧菜单键会在新的 Go `wechat` handler 中被收束为用户向会话切换或 retired reply

- `menu sync and transport-specific commands`
  新归属：transport adapter 加 control plane
  状态：已部分启动
  当前覆盖：微信测试号现已切到新的 Go 服务，并下发 `决 / 想 / 模式` 玩家菜单；transport adapter 已负责把按钮点击归一化为 `单人角色`、`联机`、`反馈` 和若干快捷 action command，而不是继续走旧管理员桥接

- `admin commands`
  新归属：control plane 加 operator-facing inspection endpoint
  状态：已部分启动
  当前方向：admin command 保持在 gameplay mode logic 之外

### Platform / Ops

- `hot config reload`
  新归属：control plane 加 mode / agent policy reload
  状态：已部分启动
  当前方向：配置只改变 policy，不应变成 hidden state authority

- `health reporting`
  新归属：control plane 加 repository / recovery visibility
  状态：已部分启动
  当前方向：按 kernel、execution、queue、delivery 健康度汇报，而不是按包内局部计数器汇报

- `delivery retry / idempotency / leases`
  新归属：`kernel`、`repository`、`delivery`、`recovery`
  状态：已部分启动
  当前覆盖：新执行路径中已经存在 execution record、idempotency lookup、lease timestamp、stale-execution resume supervision、projection frame、pending delivery intent 和 queued delivery job

### Requirement / Content System

- `validated mechanic capture`
  新归属：`requirement registry`、versioned ruleset、prompt pack、可测试的 asset revision
  状态：已启动
  当前覆盖：仓库内已建立 `requirements/` 基础目录、registry 和 bootstrap asset；`internal/requirementregistry` 已可解析 ruleset / prompt pack / gameplay asset；本地 / CI 校验已接入

- `experiment-to-asset conversion`
  新归属：gameplay validation flow 加 requirement asset
  状态：已启动
  当前覆盖：已建立 `experiment trace -> gameplay asset` 的基础引用形状，但还没有接入真实 validation 输出

- `runtime-facing requirement loading`
  新归属：`requirement registry`、kernel、mode execution input
  状态：已启动
  当前覆盖：kernel 已会解析 runtime 上声明的 requirement bundle，并把 resolved requirement 传给 mode

## 明确不继承的内容

以下内容不是迁移目标：

- legacy tmux 常驻进程 hack
- Python bridge 上的 prompt-version invalidation 小技巧
- 混杂的 config 来源和手工 patch 的 runtime global
- legacy CLI parsing 行为
- 累积下来的 bridge-specific workaround
- 当前这套会妨碍 kernel-centered 重建的 thin runtime split

这些内容可以帮助我们理解 requirement，但不会作为实现模式被照搬。
