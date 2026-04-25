#!/usr/bin/env sh

set -eu

workdir=""
preset=""
slug=""
task_prefix=""
session_id=""
model=""
print_only="0"
max_running="1"
allow_same_lane="0"
interactive="0"
timeout_seconds="1800"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --preset)
      preset="$2"
      shift 2
      ;;
    --slug)
      slug="$2"
      shift 2
      ;;
    --task-prefix)
      task_prefix="$2"
      shift 2
      ;;
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --model)
      model="$2"
      shift 2
      ;;
    --print-only)
      print_only="1"
      shift
      ;;
    --interactive)
      interactive="1"
      shift
      ;;
    --timeout-seconds)
      timeout_seconds="$2"
      shift 2
      ;;
    --max-running)
      max_running="$2"
      shift 2
      ;;
    --allow-same-lane)
      allow_same_lane="1"
      shift
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

if [ -z "$preset" ]; then
  echo "missing required arg: --preset" >&2
  exit 2
fi

cd "$workdir"

title=""
creative_line=""
core_loop=""
emotion=""
target_minutes="15"
failure_condition=""
success_condition=""

case "$preset" in
  peigei|ration)
    if [ -z "$slug" ]; then
      slug="peigei-ri"
    fi
    title="配给日"
    creative_line="饥饿与配给"
    core_loop="分配口粮并结算饱腹、关系和风险变化"
    emotion="饥饿、愧疚、短缺下的秩序与偏私"
    failure_condition="关键人物断粮或玩家自身状态崩溃"
    success_condition="在有限口粮下完成一轮分配并进入稳定结算"
    ;;
  qidao|prayer)
    if [ -z "$slug" ]; then
      slug="yejian-qidao"
    fi
    title="夜间祈祷"
    creative_line="夜间祈祷"
    core_loop="在每晚祈祷、沉默与附和之间选择，并结算精神与怀疑度"
    emotion="恐惧、虔敬、压抑中的自我保全"
    failure_condition="精神崩溃或怀疑度失控"
    success_condition="撑过连续夜晚并稳定进入一个祈祷结局"
    ;;
  dianming|rollcall)
    if [ -z "$slug" ]; then
      slug="gongtou-dianming"
    fi
    title="工头点名"
    creative_line="工头点名"
    core_loop="在点名、抽查和掩饰间做选择并结算嫌疑与工分"
    emotion="紧张、羞耻、随时暴露的高压劳动"
    failure_condition="嫌疑度爆表或在抽查中直接失手"
    success_condition="挺过点名和抽查并保持可继续劳动的状态"
    ;;
  shuiyuan|water)
    if [ -z "$slug" ]; then
      slug="shuiyuan-lunzhi"
    fi
    title="水源轮值"
    creative_line="水源轮值"
    core_loop="在取水、排队、藏水和举报之间选择，并结算体力、水量与信任"
    emotion="干渴、猜疑、排队秩序里的小型背叛"
    failure_condition="水量耗尽、体力崩溃或被轮值队排除"
    success_condition="撑过轮值并保住最低水量与邻近工友信任"
    ;;
  huijiang|mortar)
    if [ -z "$slug" ]; then
      slug="huijiang-peibi"
    fi
    title="灰浆配比"
    creative_line="灰浆配比"
    core_loop="在快拌、偷料、补水和遮掩之间选择，并结算质量、疲劳与追责风险"
    emotion="赶工、失误恐惧、把坏材料抹进墙里的负罪感"
    failure_condition="墙段质量崩坏或追责风险失控"
    success_condition="完成一段灰浆任务并把质量风险压到可遮掩范围"
    ;;
  yinji|mark)
    if [ -z "$slug" ]; then
      slug="yinji-shencha"
    fi
    title="印记审查"
    creative_line="印记审查"
    core_loop="在验印、遮印、替人作证和沉默之间选择，并结算身份风险与亲疏"
    emotion="身份暴露、互相作证、制度缝隙里的侥幸"
    failure_condition="身份疑点被锁定或关键同伴被牵连"
    success_condition="通过一轮审查并保住至少一个可信关系"
    ;;
  bingpeng|sickbay)
    if [ -z "$slug" ]; then
      slug="bingpeng-yezhen"
    fi
    title="病棚夜诊"
    creative_line="病棚夜诊"
    core_loop="在救治、分药、隐瞒病情和叫醒看守之间选择，并结算病势、药量与暴露"
    emotion="夜里的喘息、稀缺药物、救人与自保的冲突"
    failure_condition="病势恶化、药量断绝或夜诊被看守发现"
    success_condition="撑过夜诊并让至少一名工友稳定到天亮"
    ;;
  *)
    echo "unknown preset: $preset" >&2
    echo "available presets: peigei|ration, qidao|prayer, dianming|rollcall, shuiyuan|water, huijiang|mortar, yinji|mark, bingpeng|sickbay" >&2
    exit 2
    ;;
esac

if [ -z "$task_prefix" ]; then
  task_prefix="$slug"
fi

sh scripts/microgame_factory_create.sh \
  --workdir "$workdir" \
  --slug "$slug" \
  --title "$title" \
  --creative-line "$creative_line" \
  --core-loop "$core_loop" \
  --emotion "$emotion" \
  --target-minutes "$target_minutes" \
  --failure-condition "$failure_condition" \
  --success-condition "$success_condition" \
  --task-prefix "$task_prefix" >/dev/null

echo "factory ready: plan/microgames/$slug" >&2

set -- sh scripts/claudecode_manager_next.sh --workdir "$workdir" --max-running "$max_running"
set -- "$@" --worker-prefix "${task_prefix}-"
if [ "$interactive" = "0" ]; then
  set -- "$@" --run-once --timeout-seconds "$timeout_seconds"
fi
if [ "$allow_same_lane" = "1" ]; then
  set -- "$@" --allow-same-lane
fi
if [ -n "$session_id" ]; then
  set -- "$@" --session-id "$session_id"
fi
if [ -n "$model" ]; then
  set -- "$@" --model "$model"
fi
if [ "$print_only" = "1" ]; then
  set -- "$@" --print-only
fi

exec "$@"
