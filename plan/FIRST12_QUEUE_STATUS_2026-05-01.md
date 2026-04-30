# First 12 Queue Status - 2026-05-01

Last manager pass: `2026-05-01 06:51:59 CST`

Source queue: `/home/openclaw/babel-runtime/plan/MICROGAME_PRODUCTION_BATCH_2026-04-27.json`
Line context index: `.codex-runtime/microgame-line-context/INDEX.md`
Legacy takeover registry: `/home/openclaw/babel-runtime/plan/legacy-claude-takeover/legacy_takeover.json`

## Current Pass

- Re-read the compact JSON First 12 queue, the manager-local line context index, all twelve First 12 `LINE_BRIEF.md` files, and the legacy Claude takeover registry.
- Legacy takeover entries remain planner-only takeover work; they do not change this First 12 dispatch lane.
- All First 12 slugs have a `LINE_BRIEF.md` scene interaction contract. Choice-only implementations remain rejected for every lane.
- Ran strict packet audits before trusting currently waiting First 12 packets:
  - `ok shuiyuan-lunzhi/shuiyuan-lunzhi-state [queued]`
  - `ok shuiyuan-lunzhi/shuiyuan-lunzhi-ui [queued]`
  - `ok heizhang-xiaoce/heizhang-xiaoce-ui [rework]`
- Ran `/home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker`; it refused to start another worker because the configured cap is full: `game worker concurrency limit reached: 3 >= 3`.
- Refreshed manager status after the batch attempt: `games=14 dirty=0 dispatchable=3 review=0 queued=18 running=3 blocked=18 rework=1 done=43`.
- `review=0`, so there was no completed handoff to run through `microgame_worker_review_handoff.sh`.
- No dirty-worktree reconciliation was needed in this pass because the manager status is `dirty=0`.
- No worker cleanup was run because every active worker still reports `running`.

## Follow-up Pass 06:51 CST

- Re-read the compact JSON First 12 queue, the line-context index, all twelve `LINE_BRIEF.md` files, and the legacy takeover registry before dispatch decisions.
- Confirmed no First 12 slug is a legacy takeover game.
- Probed active First 12 workers once; no manual wait loop was started.
- `heizhang-xiaoce-ui` was selected by the batch dry-run, confirmed to have `MECHANIC_SPEC.md` and `SCENE_INTERACTION_SPEC.md`, and passed strict packet audit: `ok heizhang-xiaoce/heizhang-xiaoce-ui [rework]`.
- `heizhang-xiaoce-ui` was started with the shared worker launcher after audit. Current manager status now reports `running=4`, `dispatchable=2`, and `review=0`; stop dispatching until capacity is back at or below the configured cap.
- Refreshed the next waiting First 12 packet audit: `ok shuiyuan-lunzhi/shuiyuan-lunzhi-ui [queued]`.
- Final status shows `dirty=1` because `gongpai-jiaohuan-ui` is actively running and has worker-owned in-flight changes. Do not reconcile that worktree while the worker remains `running`.

## Running First 12 Workers

- `peigei-ri-integration`: `running`, started `2026-04-30T22:42:46Z`; probe shows zero-byte Claude output, missing report, clean git status, tmux/process present.
- `huijiang-peibi-integration`: `running`, started `2026-04-30T22:49:09Z`; probe shows zero-byte Claude output, missing report, clean git status, tmux/process present.
- `gongpai-jiaohuan-ui`: `running`, started `2026-04-30T22:47:11Z`; probe showed zero-byte Claude output and missing report, then final manager status reported worker-owned dirty state while the worker remains running.
- `heizhang-xiaoce-ui`: `running`, started after strict packet audit in the 06:51 CST pass.

## Waiting First 12 Packets

- `shuiyuan-lunzhi-state`: `queued`, strict audit passed. Line contract requires bucket fill amount plus route movement affecting spillage and queue/trust.
- `shuiyuan-lunzhi-ui`: `queued`, strict audit passed. This was audited too because manager status advertises `shuiyuan-lunzhi-ui/queued` while the registry also has `shuiyuan-lunzhi-state` queued.

Queued/rework packets still need to wait until worker capacity drops back below the configured cap. Do not dispatch while status reports four running workers.

## Idle Or Completed First 12 Lanes

- `duanti-yunliao`: clean, idle/seed-next-game; all standard workers are done.
- `dengyou-fenpei`: clean, idle/seed-next-game; all standard workers are done.
- `tiban-mingdan`: clean, idle/seed-next-game; all standard workers are done.
- `bingpeng-yezhen`: clean, idle/seed-next-game; all standard workers are done.

## Blocked First 12 Lanes

- `zhuiwu-yujing`: stopped because `plan/microgames/zhuiwu-yujing/MECHANIC_SPEC.md` and `plan/microgames/zhuiwu-yujing/SCENE_INTERACTION_SPEC.md` are missing.
- `jiaoshoujia-qiangxiu`: stopped because `plan/microgames/jiaoshoujia-qiangxiu/MECHANIC_SPEC.md` and `plan/microgames/jiaoshoujia-qiangxiu/SCENE_INTERACTION_SPEC.md` are missing.
- `tianti-zuihou-yiji`: stopped because `plan/microgames/tianti-zuihou-yiji/MECHANIC_SPEC.md` and `plan/microgames/tianti-zuihou-yiji/SCENE_INTERACTION_SPEC.md` are missing.

Each blocked lane has a valid `LINE_BRIEF.md`, but the lane must stay stopped until the s control plane repairs generation of both required spec files.

## Outside First 12 Packet Audit Block

`gongtou-dianming-ui` is visible as dispatchable in manager status, but it is outside the First 12 queue and a prior strict packet audit failed. Do not start this worker until the s control plane repairs packet generation.

Exact failed-audit findings recorded for `gongtou-dianming-ui`:

- missing Direction Lock anchor
- missing one-sentence direction summary
- missing core-loop direction summary
- missing required-state direction summary
- missing mandatory Direction Lock constraint
- missing stop-instead-of-drifting constraint
- missing Suggestions-only direction change rule
- missing no-direct-new-direction rule
- missing core-loop reinforcement acceptance
- missing `DIRECTION_LOCK.md` context file
- missing `MINI_GDD.md` context file
- missing `MECHANIC_SPEC.md` context file
- missing `SCENE_INTERACTION_SPEC.md` context file
- missing `TASK_BREAKDOWN.md` context file
- missing canonical worker finish command
- missing primary input interaction contract
- missing minimum interaction contract
- missing no-choice-only interaction rule

## Next Safe Actions

1. Let autorun and the registry advance the running workers; avoid manual probe/wait loops.
2. Do not dispatch another worker while manager status reports `running=4`; when capacity is available again, run `/home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker`.
3. Before trusting any packet not listed above as audited, run `/home/openclaw/babel-runtime/scripts/babel_ops.sh microgame audit-packets --game-workdir <game> --worker-id <worker> --apply --strict`.
4. When `review > 0`, use `/home/openclaw/babel-runtime/scripts/microgame_worker_review_handoff.sh` and verify git status, packet audit, test output, file budget, and report path coverage before acceptance.
5. Keep `zhuiwu-yujing`, `jiaoshoujia-qiangxiu`, and `tianti-zuihou-yiji` blocked until both missing spec files exist.
