# First 12 Queue Dispatch Note - 2026-05-02

## Follow-up: 2026-05-02T11:28:16+0800

Scope:
- Compact queue read first from `/home/openclaw/babel-runtime/plan/MICROGAME_PRODUCTION_BATCH_2026-04-27.json`.
- Manager-local line context index read from `/home/openclaw/claudecode-manager/.codex-runtime/microgame-line-context/INDEX.md`.
- All twelve First 12 `LINE_BRIEF.md` files read before dispatch consideration.
- Legacy takeover registry read from `/home/openclaw/babel-runtime/plan/legacy-claude-takeover/legacy_takeover.json`; it remains separate from the First 12 queue and was not used as fallback.

First 12 queue order:
- `peigei-ri`, `huijiang-peibi`, `duanti-yunliao`, `dengyou-fenpei`, `tiban-mingdan`, `bingpeng-yezhen`, `gongpai-jiaohuan`, `zhuiwu-yujing`, `heizhang-xiaoce`, `shuiyuan-lunzhi`, `jiaoshoujia-qiangxiu`, `tianti-zuihou-yiji`.

Pre-dispatch status observed before the batch command:
- `games=20 dirty=1 dispatchable=0 review=0 queued=5 running=0 blocked=0 rework=1 done=99`
- queue detail: `launchable_games=0 active_game_locks=0 queued_behind_running=0 packet_contract_repair=1 idle_or_seed=15`
- Status-visible dirty lane: `peigei-ri` at `peigei-ri-qa/rework`, action `clean_worktree_before_dispatch`, note `direction check incomplete: missing Direction Check section`.

Dirty-worktree handling:

```bash
sh /home/openclaw/babel-runtime/scripts/babel_ops.sh microgame reconcile-dirty --apply --review --reset-review-failed
```

Result:

```text
[
  {
    "slug": "",
    "workdir": "",
    "dirty_files": null,
    "status": "clean",
    "action": "none",
    "note": "no dirty microgame repos"
  }
]
```

Preferred dispatch command:

```bash
sh /home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker --max-running 6
```

Result:

```text
exit code: 3
no batch item requires preparation
```

Decision:
- No safe launchable First 12 item is available under the current compact queue and concurrency rules.
- Per manager instruction after the exit-3 result, no registry hand-inspection, fallback lane invention, direct worker start, packet-audit/start fallback, raw kill, stale-session cleanup, or legacy-lane fallback was performed.
- Packet audit: no worker packet was prepared by this pass, so there was no packet to trust or audit.

Scope:
- Compact queue read: `/home/openclaw/babel-runtime/plan/MICROGAME_PRODUCTION_BATCH_2026-04-27.json`
- Line context index read: `.codex-runtime/microgame-line-context/INDEX.md`
- First 12 `LINE_BRIEF.md` files read before dispatch consideration.
- Legacy takeover registry read: `/home/openclaw/babel-runtime/plan/legacy-claude-takeover/legacy_takeover.json`

Dispatch command:

```bash
sh /home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker --max-running 6
```

Result:

```text
exit code: 3
no batch item requires preparation
```

Decision:
- No safe launchable First 12 item is available under the current compact queue and concurrency rules.
- Per manager instruction, no registry fallback inspection was performed and no direct worker start was attempted.
