# 路线图

这份路线图把目标架构拆成可执行的构建顺序。

它有意采用 phase-based 方式，目标是在系统变大的同时仍然可控。

## 规划规则

每个阶段都应该：

- 有明确的完成条件
- 有明确的非目标
- 结束时让仓库保持可测试
- 避免新增 legacy debt

## Phase 0：设计重置

### 目标

用 `kernel-first`、`execution-centric` 的设计基线替换当前“先脚手架、后统一”的架构理解。

### 范围

- canonical docs 重置
- 统一 runtime instance 模型
- execution stage 模型
- persistence contract 重置
- 当前代码库的迁移预期

### 完成条件

- canonical docs 一致描述同一套目标系统
- 实现工作不再把当前包拆分当成固定事实
- 后续构建阶段能从一套统一 runtime model 出发

### 非目标

- 不做大规模 feature 扩张
- 不为了方便而保留坏边界

## Phase 1：Kernel Foundation

### 目标

构建可以接收输入、执行幂等控制、持久化 stage、并在中断后恢复的 execution kernel。

### 范围

- runtime instance repository
- execution record
- lease
- event journal
- snapshot persistence
- replay
- recovery 和 restart simulation

### 完成条件

- kernel 可以恢复未完成 execution
- persistence 和 recovery 使用与正常执行相同的 execution stage
- replay 和 restart simulation 成为常规开发工具

### 非目标

- 不追求完整产品 parity
- 不允许 mode shortcut 绕过 kernel

## Phase 2：Mode System and Product Surfaces

### 目标

把用户可见能力重建为运行在 kernel 之上的 mode module。

### 范围

- mode router
- free chat mode
- project consult mode
- solo scene mode
- room scene mode
- ingress adapter
- delivery dispatcher
- control-plane operator surface

### 完成条件

- 主要用户模式都跑在同一条 kernel lifecycle 上
- transport、mode behavior、delivery 保持清晰分离
- 文档和测试都按 mode 描述行为，而不是按 ad hoc handler 描述

### 非目标

- 暂不做 multi-agent scale-out
- 暂不追 Godot client parity

## Phase 3：Agent Task Supervision and Operational Memory

### 目标

支持长时段、多模式连续运行，但不把任何一个 live agent session 当成系统 authority。

### 范围

- agent task queue
- worker routing
- budget / timeout policy
- artifact persistence
- operational memory file generation
- agent replacement 和 recovery behavior

### 完成条件

- task 和 artifact 可恢复
- operational memory 是 file-backed 且可重建
- agent failure 不威胁 canonical runtime state

### 非目标

- 不做过早的 C++ 提取
- 不允许 hidden agent-owned truth

## Phase 4：Requirement Asset System

### 目标

把经过验证的 gameplay 行为沉淀成 versioned ruleset、prompt pack 和 requirement asset，而不是继续留在 prose 里。

### 范围

- ruleset registry
- prompt pack registry
- content constraint
- experiment recording
- accepted mechanic assetization
- 从 runtime behavior 到 requirement asset 的可追踪性

### 完成条件

- 已验证 mechanic 不再只存在于对话或 markdown 中
- runtime behavior 能关联到 asset revision 和测试
- requirement management 变成可运转的系统，而不是纯文档行为

### 非目标

- 不试图替代 Babel / C++ 对稳定 core 的所有权

### 当前落点

- 仓库内已建立 `requirements/` 基础目录、registry 和 bootstrap asset
- `internal/requirementregistry` 已提供 filesystem-backed loader / resolver
- kernel 已能在执行前解析 runtime 上声明的 ruleset / prompt pack / gameplay asset
- 本地 hook 与 GitHub CI 已能校验 requirement asset registry 与基础引用关系
- 下一步不是继续堆 prose，而是把 runtime loader、resolver 和 traceability 消费链接进内核

## Phase 5：Godot Projection Pipeline

### 目标

输出结构化 scene projection，让未来 Godot 端可以消费，而不改变 runtime authority。

### 范围

- structured scene payload
- actor / world presentation projection
- experiment-to-Godot mapping
- projection validation flow

### 完成条件

- structured projection 派生自 canonical snapshot
- 已验证的 real-time loop 可被导出为面向 Godot 的 requirement asset
- Godot 仍然是 client，而不是 simulation authority

### 非目标

- 不把 simulation authority 迁入 Godot

## Phase 6：C++ Extraction

### 目标

在不改变系统所有权边界的前提下，把已证明确实是热点的路径从 Go 提取到 C++。

### 范围

- 密集 simulation 热路径
- 重型 settlement step
- map / resource propagation 热点
- 大规模 relationship update 热点

### 完成条件

- 有测量依据的性能瓶颈被稳定接口后面的实现替换
- Go 继续拥有 orchestration、persistence 和 recovery

### 非目标

- 不把 orchestration 迁入 C++
- 不做无 profiling 依据的提取

## 持续性工作线

这些工作线贯穿所有阶段：

### 文档纪律

- 保持 canonical docs 最新
- 新设计文档统一进入 `docs/incoming/`
- 持续更新 feature inheritance 和 execution contract

### 测试纪律

- 每引入一个新子系统就补对应测试
- 优先使用 replayable、restart-aware 的 kernel harness
- 保持 deterministic path 和 projection path 易于验证

### 边界纪律

- kernel 拥有的 runtime state 始终是 canonical
- agent 始终是 assistive，而不是 authority
- transport 始终在 runtime 之外
- requirement asset 必须保持 versioned

## 近期实现队列

下一批实际工作应大致按下面顺序推进：

1. 冻结 kernel 和 repository 的数据契约
2. 实现 execution record、lease 和可恢复的 stage progression
3. 重建 mode router，以及 free chat / project consult module
4. 在同一 kernel 上重建 solo scene / room scene module
5. 接上 projection 和 delivery queue
6. 接上 agent task supervision 和 file-backed operational memory
7. 在已有 filesystem registry 基础上继续接 runtime consumption、resolver cache 和 traceability

## 停止条件

如果出现以下情况，实施应暂停并重新评估：

- 有子系统开始在 kernel 之外改 canonical state
- recovery path 和 normal path 出现分叉
- 设计变化进入仓库的速度已经超过 canonical docs 的吸收能力
- 功能在没有测试、execution contract 或 inheritance mapping 的前提下被直接加入
