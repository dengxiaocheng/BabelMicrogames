# Microgame Manager Dispatch Note - 2026-05-02 07:40:58 CST

## Scope

- Objective: drive the First 12 Queue from the compact JSON.
- Compact queue read: `/home/openclaw/babel-runtime/plan/MICROGAME_PRODUCTION_BATCH_2026-04-27.json`
- Line context index read: `.codex-runtime/microgame-line-context/INDEX.md`
- First 12 `LINE_BRIEF.md` files read and present:
  `peigei-ri`, `huijiang-peibi`, `duanti-yunliao`, `dengyou-fenpei`,
  `tiban-mingdan`, `bingpeng-yezhen`, `gongpai-jiaohuan`,
  `zhuiwu-yujing`, `heizhang-xiaoce`, `shuiyuan-lunzhi`,
  `jiaoshoujia-qiangxiu`, `tianti-zuihou-yiji`.
- Legacy takeover registry read:
  `/home/openclaw/babel-runtime/plan/legacy-claude-takeover/legacy_takeover.json`

## Preflight Status

`sh /home/openclaw/babel-runtime/scripts/claudecode_manager_status.sh`

- `games=20`
- `dirty=0`
- `dispatchable=0`
- `review=0`
- `queued=6`
- `running=1`
- `blocked=0`
- `launchable_games=0`
- `active_game_locks=1`
- `queued_behind_running=1`
- `packet_contract_repair=1`

Observed current active lane from status:

- `peigei-ri`: `peigei-ri-integration/running`, action `wait_running_worker`
- `gongtou-dianming`: `gongtou-dianming-ui/queued`, action `repair_worker_packet_contract`

No dirty microgame worktree blocker was reported, so dirty reconciliation was not run.

## Batch Command Result

Command:

```bash
sh /home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker --max-running 6
```

Result:

- exit code: `3`
- output: `no batch item requires preparation`

## Decision

There is no safe launchable item under the current queue and concurrency rules.

Per dispatch policy, after the batch command returned exit code `3`, I did not inspect
worker registries by hand, did not invent a fallback lane, and did not start workers
directly. The next safe manager action is to let the `s` control plane/autorun advance
the running or packet-repair lanes, then rerun the batch command or review handoffs when
they appear.
