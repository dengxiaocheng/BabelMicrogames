# Codex Manager 智能化路线

当前独立 Codex manager 已经能跑通“派发 ClaudeCode worker -> 回交 -> watcher 唤醒 -> 人工/半自动验收”的闭环，但它还不够智能。

本文记录当前不足和升级方向。

## 当前真实能力

已经具备：

- 独立 manager workdir
- per-game workdir
- per-game Claude session id
- worker packet / report
- issue handoff
- tmux 隔离
- quiet hours
- 基本 repo guard
- Pages 发布基础

这些能力解决的是“能稳定跑”，不是“能聪明管理”。

## 当前不智能的地方

### 1. incoming 计划不会自动消化

现在 `s` 已经有：

```bash
/home/openclaw/babel-runtime/docs/incoming/microgame-plans/
```

但 manager 还不会自动：

- 扫描新计划
- 判断新建还是继续
- 提取主题、玩法、规模和验收标准
- 生成标准 worker 队列

### 2. 全局状态不够集中

当前状态分散在：

- 每个小游戏的 `.codex-runtime/claudecode_workers.json`
- 每个小游戏的 issue state
- tmux 会话
- GitHub repo
- `s` 文档
- manager 文档

缺少一个 manager 视角的总表。

### 3. worker 调度只会按队列

当前主要是固定顺序：

```text
foundation -> state -> content -> ui -> integration -> qa
```

它不会真正评估：

- 哪个 worker 阻塞最多
- 哪个任务风险最高
- 哪个 worker 失败后应该降级
- 是否应该暂停某个游戏改做另一个游戏
- 多个游戏之间如何分配 ClaudeCode token

### 4. report 审查还不够结构化

manager 现在能看 report 和跑测试，但还没有形成强制评分：

- scope 是否越界
- budget 是否超额
- 测试是否可信
- 是否有隐藏浏览器依赖
- 是否引入不可维护结构
- 是否应该 rework 而不是继续

### 5. 失败恢复策略不够明确

当前失败一般靠人工判断。

缺少：

- worker 超时重试策略
- 同一 worker 多次失败后的拆分策略
- ClaudeCode 网络错误处理
- 脏工作树保护
- 自动取消和回滚规则

### 6. 没有生产节奏指标

现在还不能直接回答：

- 今天完成了几个 worker
- 哪些游戏卡住了
- 哪个 Claude session 失败率高
- 哪个 repo 没发布 Pages
- 当前队列还能跑多久

## 目标形态

manager 应该变成一个“生产调度脑”，但仍然不常驻大模型。

推荐形态：

```text
轻量 watcher 常驻
  -> 事件触发 Codex manager
  -> Codex manager 读取结构化状态
  -> 执行一次决策
  -> 派发/验收/记录
  -> 退出
```

也就是说：

- Go 脚本和状态文件负责连续性
- Codex manager 负责判断和策略
- ClaudeCode worker 负责小范围实现

## 第一阶段：建立 manager 总表

新增一个 manager 级状态文件：

```bash
/home/openclaw/claudecode-manager/.codex-runtime/microgame_manager_state.json
```

建议字段：

- games
- repos
- workdirs
- pages_url
- claude_session_id
- current_stage
- queued_workers
- running_workers
- blocked_reason
- last_success_commit
- last_test_result
- next_recommended_action

这个文件不替代 GitHub 和各游戏 registry，只做 manager 视角索引。

当前已经落地的入口：

```bash
sh scripts/claudecode_manager_refresh_state.sh
sh scripts/claudecode_manager_status.sh
```

约束：

- 总表只扫描 `/home/openclaw/babel-microgames/*`
- 每个小游戏自己的 `.codex-runtime/claudecode_workers.json` 仍是 worker 真源
- manager 占位仓自己的 `.codex-runtime/claudecode_workers.json` 不再允许作为调度真源
- 如果发现历史遗留状态，先执行 `sh scripts/claudecode_manager_clean_legacy_state.sh`

这样 Codex manager 可以从一个总表回答“哪个游戏可派发、哪个需要验收、哪个 workdir 脏、哪个 Claude session 绑定到哪个游戏”，但不会把所有 worker 状态集中写回 manager 仓库。

## 第二阶段：incoming 扫描器

manager 应提供脚本：

```bash
sh scripts/claudecode_manager_ingest_incoming.sh
```

它读取：

```bash
/home/openclaw/babel-runtime/docs/incoming/microgame-plans/
```

输出：

- 新游戏候选列表
- 已存在游戏的继续建议
- 缺信息计划列表
- 建议生成的 worker 拆分

不建议一开始全自动执行创建；先让 Codex manager 生成决策报告，再决定是否执行。

## 第三阶段：结构化验收

manager 应把每个 worker 的验收变成固定评分。

建议评分项：

- scope: 是否只改 write scope
- budget: 是否在文件数和行数预算内
- tests: 测试是否执行且通过
- build: 页面或模块是否可导入
- maintainability: 是否引入全局临时结构
- handoff: report 是否足够下一轮判断

评分结果写回 worker report 或 manager state。

## 第四阶段：失败降级

失败不应只重跑。

推荐规则：

- 第一次失败：同 worker 标 `rework`
- 第二次失败：拆成更小 worker
- 第三次失败：暂停该游戏，写 blocked reason
- 网络失败：不算代码失败，但记录 session/network risk
- scope 越界：必须 rework，不进入下一阶段

## 第五阶段：跨游戏调度

当同时有多个小游戏时，manager 应按优先级调度：

1. 正在接近完成的游戏优先
2. 有 Pages 但缺 QA 的游戏优先
3. 失败两次以上的游戏降级
4. 新 incoming 不应打断快完成的游戏
5. quiet hours 永远优先于调度欲望

## 决策边界

manager 可以自动做：

- 选择下一个 queued worker
- 启动一个 worker
- 跑测试
- scope 检查
- commit/push 已验收改动
- 打开下一阶段 issue

manager 必须请求用户或 `s` 介入：

- 新游戏主题不清楚
- 需要删除或重写已有游戏
- 单个 worker 超出 500 行预期
- 需要改变 repo 结构
- 需要使用 `s/m` issue 空间
- 连续失败超过阈值

## 近期最小改进

优先做这几个，不要一次性做复杂系统：

1. `microgame_manager_state.json`：已由 `scripts/claudecode_manager_refresh_state.sh` 生成
2. incoming 扫描报告脚本
3. worker 结构化验收脚本
4. blocked/rework 统一规则
5. `s` 一条命令查看所有小游戏状态

这五个完成后，独立 Codex manager 才算从“能调度”进入“会管理”。
