# 微游戏工厂流程

本文是 Babel 微游戏工厂的端到端流程文档。

目标是把 `s`、独立 Codex manager、ClaudeCode worker、每个小游戏仓库之间的关系固定下来，避免后续再把源码、issue、会话和工作目录混在一起。

## 角色边界

| 角色 | 工作目录 / 仓库 | 职责 |
| --- | --- | --- |
| `s` 总控 | `/home/openclaw/babel-runtime` / `dengxiaocheng/BabelOnline-GoCpp` | 接收 incoming 计划、看全局进程、唤起 manager，不直接写小游戏 |
| Codex manager | `/home/openclaw/claudecode-manager` / `dengxiaocheng/BabelMicrogames` | 拆包、派发 ClaudeCode、审查、提交、推进下一阶段 |
| ClaudeCode worker | 每个小游戏 workdir 内的固定 Claude session | 只完成 packet 声明的窄任务 |
| 小游戏源码 | `/home/openclaw/babel-microgames/<game>` / `dengxiaocheng/BabelMicrogame-*` | 保存游戏源码、测试、Pages 工作流、worker runtime state |
| `m` Babel 本体 | `/home/openclaw/Babel` / `dengxiaocheng/Babel` | Babel 主项目，不承载微游戏 worker 队列 |

## 仓库规则

`BabelMicrogames` 是 manager 资料仓。

它可以保存：

- manager 规则
- manager 脚本
- 流程文档
- 微游戏工厂模板
- manager 级计划和状态说明

它不能保存：

- 某个小游戏的正式源码
- 某个小游戏的长期 worker issue 队列
- `s` 或 `m` 的长期会话 issue

小游戏源码和小游戏 worker issue 必须进入：

```text
dengxiaocheng/BabelMicrogame-*
```

## 主流程

### 1. 计划进入 s

新的小游戏计划先进入：

```bash
/home/openclaw/babel-runtime/docs/incoming/microgame-plans/
```

`s` 只负责读取、归类、判断是否交给 manager。

`s` 不直接启动 `claude`，也不在 `babel-runtime` 里写小游戏源码。

### 2. s 唤起 Codex manager

`s` 需要继续微游戏工厂时，目标不是自己执行，而是唤起或检查：

```bash
/home/openclaw/claudecode-manager
```

推荐入口：

```bash
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_start_watcher.sh
sh /home/openclaw/claudecode-manager/scripts/claudecode_quiet_hours_guard.sh
```

如果要打开新阶段 issue，使用：

```bash
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_open_game_stage.sh
```

### 3. manager 建立或继续小游戏

manager 判断 incoming 计划后，选择：

- 创建新小游戏 workdir
- 继续已有小游戏
- 暂停并要求用户补充规则

创建新小游戏必须满足：

- 独立 GitHub repo
- 独立本地 workdir
- 独立 Claude session id
- 独立 worker registry
- 如是 JS 小游戏，配置 GitHub Pages

### 4. manager 生成 worker packet

每个 worker packet 必须声明：

- worker id
- lane
- task level
- read scope
- write scope
- test command
- acceptance
- finish protocol

默认拆分：

- `foundation`
- `state`
- `content`
- `ui`
- `integration`
- `qa`

单个 worker 只做一个窄任务，不承担项目级管理。

### 5. manager 打开阶段 issue

每个阶段 issue 应在对应小游戏 repo 中打开。

关键要求：

- issue state 写在小游戏 workdir
- `resume-workdir` 必须是 `/home/openclaw/claudecode-manager`
- watcher 关闭 issue 后恢复的是 Codex manager，不是小游戏目录

这条规则防止 manager 被恢复进游戏源码目录。

### 6. manager 启动 ClaudeCode worker

无人值守 worker 用独立 tmux 会话运行：

```bash
sh /home/openclaw/claudecode-manager/scripts/claudecode_worker_start_tmux.sh \
  --workdir /home/openclaw/babel-microgames/<game> \
  --worker-prefix <game-slug>-
```

ClaudeCode 只允许在 packet 的 write scope 内改代码。

### 7. worker 回交

worker 完成后必须：

1. 写 report
2. 执行 finish command
3. 通过 manager handoff 关闭当前阶段 issue

worker 不做：

- 项目归档
- 下阶段决策
- manager 状态判断
- 跨游戏调度

### 8. watcher 唤醒 Codex manager

watcher 检测到 worker 关闭 issue 并留下 report 后，恢复 Codex manager。

恢复后的 manager 必须：

- 读取 report
- 检查 diff 是否越界
- 跑测试
- 决定 `done / rework / cancelled`
- 提交并推送小游戏 repo
- 必要时打开下一阶段 issue

### 9. 发布与索引

JS 小游戏应该配置 GitHub Pages。

manager 验收后要确认：

- repo 已 push
- test 通过
- Pages workflow 存在
- 可点击链接返回 200

链接和当前状态应记录在 manager 资料或 `s` 的总控文档中。

## 日常停止规则

每日 `14:00-18:00` 不运行 ClaudeCode worker。

执行入口：

```bash
sh /home/openclaw/claudecode-manager/scripts/claudecode_quiet_hours_guard.sh
```

quiet hours 中可以保留文档整理和状态查看，但不应继续启动 ClaudeCode 生产任务。

## 当前最低验收

每个 worker 完成后，manager 至少检查：

- `git diff --check`
- packet write scope
- 测试命令
- changed files 数量
- report 是否填写
- 是否误改 manager 或 `s/m` 目录

如果检查失败，标为 `rework`，不要直接继续下一阶段。
