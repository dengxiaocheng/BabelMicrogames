#!/usr/bin/env sh

set -eu

workdir=""
presets="dianming shuiyuan huijiang yinji bingpeng"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --presets)
      presets="$2"
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

cd "$workdir"

echo "auto-seed disabled: microgames now require one repo and one workdir per game" >&2
exit 3

slug_for_preset() {
  case "$1" in
    peigei|ration)
      echo "peigei-ri"
      ;;
    qidao|prayer)
      echo "yejian-qidao"
      ;;
    dianming|rollcall)
      echo "gongtou-dianming"
      ;;
    shuiyuan|water)
      echo "shuiyuan-lunzhi"
      ;;
    huijiang|mortar)
      echo "huijiang-peibi"
      ;;
    yinji|mark)
      echo "yinji-shencha"
      ;;
    bingpeng|sickbay)
      echo "bingpeng-yezhen"
      ;;
    *)
      return 1
      ;;
  esac
}

for preset in $presets; do
  slug=$(slug_for_preset "$preset") || {
    echo "skip unknown preset: $preset" >&2
    continue
  }

  if [ -d "plan/microgames/$slug" ]; then
    continue
  fi

  sh scripts/microgame_factory_start.sh \
    --workdir "$workdir" \
    --preset "$preset" \
    --print-only >/dev/null
  echo "seeded microgame preset: $preset ($slug)"
  exit 0
done

echo "no microgame preset left to seed"
exit 3
