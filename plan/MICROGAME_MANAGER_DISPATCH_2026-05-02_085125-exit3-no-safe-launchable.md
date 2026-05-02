# Microgame Manager Dispatch Note

- timestamp_local: 2026-05-02 08:51:25 Asia/Shanghai
- compact_queue: `/home/openclaw/babel-runtime/plan/MICROGAME_PRODUCTION_BATCH_2026-04-27.json`
- line_context_index: `/home/openclaw/claudecode-manager/.codex-runtime/microgame-line-context/INDEX.md`
- legacy_takeover_registry: `/home/openclaw/babel-runtime/plan/legacy-claude-takeover/legacy_takeover.json`

## First 12 Queue Context Read

Confirmed local `LINE_BRIEF.md` context exists and was read for all First 12 slugs:

1. `peigei-ri`
2. `huijiang-peibi`
3. `duanti-yunliao`
4. `dengyou-fenpei`
5. `tiban-mingdan`
6. `bingpeng-yezhen`
7. `gongpai-jiaohuan`
8. `zhuiwu-yujing`
9. `heizhang-xiaoce`
10. `shuiyuan-lunzhi`
11. `jiaoshoujia-qiangxiu`
12. `tianti-zuihou-yiji`

Each line has an explicit scene interaction contract and non-choice primary input in the local brief.

## Dispatch Attempt

Command:

```bash
CLAUDECODE_MAX_RUNNING=6 /home/openclaw/babel-runtime/scripts/microgame_batch_prepare_next.sh --start-worker --max-running 6
```

Result:

```text
exit code: 3
no batch item requires preparation
```

## Blocked Reason

There is no safe launchable item under the current compact queue and concurrency rules.

The pre-dispatch control-plane status showed `dispatchable=0`, `running=1`, `queued=6`, `review=0`, and `packet_contract_repair=1`. The active running lane was `peigei-ri` at `peigei-ri-integration/running`; the queue also reported `active_game_locks=1` and `queued_behind_running=1`.

Per manager rule, after the high-level batch command returned exit code 3, no manual registry inspection, fallback lane invention, or direct worker start was performed.
