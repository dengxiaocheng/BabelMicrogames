# First 12 Queue Status - 2026-05-01

Last manager pass: `2026-05-01 06:41:29 CST`

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

## Running First 12 Workers

- `peigei-ri-integration`: `running`, started `2026-04-30T22:27:42Z`; probe shows zero-byte Claude output, missing report, clean git status, tmux/process present.
- `huijiang-peibi-integration`: `running`, started `2026-04-30T22:33:57Z`; probe shows zero-byte Claude output, missing report, clean git status, tmux/process present.
- `gongpai-jiaohuan-state`: `running`, started `2026-04-30T22:38:37Z`; `foundation` finished and autorun advanced the lane. Probe shows zero-byte Claude output, missing report, clean git status, tmux/process present.

## Waiting First 12 Packets

- `heizhang-xiaoce-ui`: `rework`, strict audit passed. Line contract requires encoding event fragments and selecting a concrete hiding spot; reject detailed/simple choice-only UI.
- `shuiyuan-lunzhi-state`: `queued`, strict audit passed. Line contract requires bucket fill amount plus route movement affecting spillage and queue/trust.
- `shuiyuan-lunzhi-ui`: `queued`, strict audit passed. This was audited too because manager status advertises `shuiyuan-lunzhi-ui/queued` while the registry also has `shuiyuan-lunzhi-state` queued.

Queued/rework packets still need to be dispatched through the high-level batch command after worker capacity drops below the configured cap.

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

1. Let autorun and the registry advance the three running workers; avoid manual probe/wait loops.
2. When capacity is available, run `/home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker`.
3. Before trusting any packet not listed above as audited, run `/home/openclaw/babel-runtime/scripts/babel_ops.sh microgame audit-packets --game-workdir <game> --worker-id <worker> --apply --strict`.
4. When `review > 0`, use `/home/openclaw/babel-runtime/scripts/microgame_worker_review_handoff.sh` and verify git status, packet audit, test output, file budget, and report path coverage before acceptance.
5. Keep `zhuiwu-yujing`, `jiaoshoujia-qiangxiu`, and `tianti-zuihou-yiji` blocked until both missing spec files exist.
