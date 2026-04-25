#!/usr/bin/env sh

set -eu

workdir=""
slug=""
title=""
creative_line=""
core_loop=""
emotion=""
target_runtime="web"
target_minutes="15"
failure_condition=""
success_condition=""
task_prefix=""
ui_root="src/ui/"
code_root="src/"
content_root="src/content/"
test_root="tests/"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --slug)
      slug="$2"
      shift 2
      ;;
    --title)
      title="$2"
      shift 2
      ;;
    --creative-line)
      creative_line="$2"
      shift 2
      ;;
    --core-loop)
      core_loop="$2"
      shift 2
      ;;
    --emotion)
      emotion="$2"
      shift 2
      ;;
    --target-runtime)
      target_runtime="$2"
      shift 2
      ;;
    --target-minutes)
      target_minutes="$2"
      shift 2
      ;;
    --failure-condition)
      failure_condition="$2"
      shift 2
      ;;
    --success-condition)
      success_condition="$2"
      shift 2
      ;;
    --task-prefix)
      task_prefix="$2"
      shift 2
      ;;
    --ui-root)
      ui_root="$2"
      shift 2
      ;;
    --code-root)
      code_root="$2"
      shift 2
      ;;
    --content-root)
      content_root="$2"
      shift 2
      ;;
    --test-root)
      test_root="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

if [ -z "$workdir" ]; then
  workdir=$(pwd)
fi

if [ -z "$slug" ] || [ -z "$title" ] || [ -z "$creative_line" ] || [ -z "$core_loop" ]; then
  echo "missing required args: --slug --title --creative-line --core-loop" >&2
  exit 2
fi

if [ -z "$task_prefix" ]; then
  task_prefix="$slug"
fi
if [ -z "$emotion" ]; then
  emotion="压迫、饥饿、秩序下的微弱反抗"
fi
if [ -z "$failure_condition" ]; then
  failure_condition="关键状态崩溃，或在本轮主循环中被系统淘汰"
fi
if [ -z "$success_condition" ]; then
  success_condition="在限定时长内完成主循环，并稳定进入至少一个可结算结局"
fi

cd "$workdir"
sh scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"

plan_dir="plan/microgames/$slug"
mkdir -p "$plan_dir"

creative_card="$plan_dir/CREATIVE_CARD.md"
mini_gdd="$plan_dir/MINI_GDD.md"
tasks_md="$plan_dir/TASKS.md"

cat > "$creative_card" <<EOF
# CREATIVE_CARD: $title

- slug: \`$slug\`
- creative_line: $creative_line
- target_runtime: $target_runtime
- target_minutes: $target_minutes
- core_emotion: $emotion
- core_loop: $core_loop
- failure_condition: $failure_condition
- success_condition: $success_condition

## Intent

- 做一个 Babel 相关的单创意线微游戏
- 只保留一个主循环，不扩成大项目
- 让 Claude worker 能按固定 packet 稳定并行
EOF

cat > "$mini_gdd" <<EOF
# MINI_GDD: $title

## Scope

- runtime: $target_runtime
- duration: ${target_minutes}min
- project_line: $creative_line
- single_core_loop: $core_loop

## Core Loop

1. 玩家进入当前情境
2. 玩家做一次关键选择或操作
3. 系统更新状态并反馈
4. 进入下一轮或结算

## State

- 主状态不超过 5 项
- 只保留服务于主循环的变量
- 不引入长链大型系统

## UI

- 只保留主界面、结果反馈、结算入口
- 不加多余菜单和后台页

## Content

- 用小型事件池支撑主循环
- 一次只验证一条 Babel 创意线

## Constraints

- 总体规模目标控制在 5000 行以内
- 单个 worker 任务必须服从 packet budget
- 如需扩线，交回 manager 重新拆
EOF

cat > "$tasks_md" <<EOF
# TASKS: $title

## Standard Worker Bundle

1. \`${task_prefix}-foundation\`
   - lane: foundation
   - level: M
   - goal: 建项目骨架、入口和最小运行流

2. \`${task_prefix}-state\`
   - lane: logic
   - level: M
   - goal: 建主循环状态和一次结算更新

3. \`${task_prefix}-content\`
   - lane: content
   - level: M
   - goal: 建最小事件池和文本/数据资产

4. \`${task_prefix}-ui\`
   - lane: ui
   - level: M
   - goal: 建主界面和操作反馈

5. \`${task_prefix}-integration\`
   - lane: integration
   - level: M
   - goal: 把状态、内容、UI 接成完整主循环

6. \`${task_prefix}-qa\`
   - lane: qa
   - level: S
   - goal: 跑测试、补缺口、确认结算与边界
EOF

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-foundation" \
  --lane foundation \
  --task-level M \
  --task-title "$title / Foundation" \
  --task-summary "建立 $title 的最小项目骨架和启动入口" \
  --goal "把微游戏的目录骨架、启动入口和最小主循环入口接起来" \
  --read-scope "$plan_dir/" \
  --write-scope "$code_root" \
  --write-scope "index.html" \
  --test-command "npm test" \
  --acceptance "项目可启动" \
  --acceptance "主循环入口存在" \
  --deliverable "基础代码骨架和 report"

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-state" \
  --lane logic \
  --task-level M \
  --task-title "$title / State" \
  --task-summary "建立 $title 的核心状态和一次循环更新" \
  --goal "把主状态、一次操作后的状态变化和结算条件补齐" \
  --read-scope "$plan_dir/" \
  --write-scope "$code_root" \
  --test-command "npm test" \
  --acceptance "核心状态可更新" \
  --acceptance "一次循环可结算" \
  --deliverable "状态更新逻辑和 report"

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-content" \
  --lane content \
  --task-level M \
  --task-title "$title / Content" \
  --task-summary "建立 $title 的最小内容池" \
  --goal "把创意线需要的文本、事件或小型数据资产补齐" \
  --read-scope "$plan_dir/" \
  --write-scope "$content_root" \
  --write-scope "$code_root" \
  --acceptance "至少有一轮可玩的内容" \
  --acceptance "内容与 Babel 创意线一致" \
  --deliverable "内容资产和 report"

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-ui" \
  --lane ui \
  --task-level M \
  --task-title "$title / UI" \
  --task-summary "建立 $title 的最小 UI 外壳" \
  --goal "把主界面、操作按钮和反馈区域接齐" \
  --read-scope "$plan_dir/" \
  --write-scope "$ui_root" \
  --write-scope "$code_root" \
  --acceptance "主界面可操作" \
  --acceptance "反馈可见" \
  --deliverable "UI 代码和 report"

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-integration" \
  --lane integration \
  --task-level M \
  --task-title "$title / Integration" \
  --task-summary "把状态、内容和 UI 接成完整主循环" \
  --goal "让玩家能完成至少一轮完整流程并进入结算" \
  --read-scope "$plan_dir/" \
  --write-scope "$code_root" \
  --write-scope "$ui_root" \
  --write-scope "$content_root" \
  --test-command "npm test" \
  --acceptance "完整主循环跑通" \
  --acceptance "结算入口可达" \
  --deliverable "集成代码和 report"

go run ./cmd/babel-issue-bridge worker-packet \
  --worker-id "${task_prefix}-qa" \
  --lane qa \
  --task-level S \
  --task-title "$title / QA" \
  --task-summary "验证边界、补测试并做最小收口" \
  --goal "确认 packet budget 没失控，测试能跑，已知问题被记录" \
  --read-scope "$plan_dir/" \
  --write-scope "$test_root" \
  --write-scope "$code_root" \
  --test-command "npm test" \
  --acceptance "测试有结果" \
  --acceptance "预算偏差已记录" \
  --deliverable "测试补充和 report"

echo "$plan_dir"
