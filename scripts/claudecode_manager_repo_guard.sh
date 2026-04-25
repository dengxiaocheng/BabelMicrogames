#!/usr/bin/env sh

set -eu

workdir=""
expected_repo="${CLAUDECODE_MANAGER_REPO:-dengxiaocheng/BabelMicrogames}"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --expected-repo)
      expected_repo="$2"
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

remote_url=$(git remote get-url origin 2>/dev/null || true)
repo=$(printf "%s" "$remote_url" |
  sed -e 's#^git@github.com:##' \
      -e 's#^https://github.com/##' \
      -e 's#^http://github.com/##' \
      -e 's#\.git$##')

if [ "$repo" != "$expected_repo" ]; then
  echo "wrong ClaudeCode manager repo: origin=$remote_url expected=$expected_repo" >&2
  echo "refusing to use shared s/m issue namespace" >&2
  exit 1
fi
