# Microgame Pages 404 Batch 2026-05-01

Last updated: 2026-05-01 19:02:16 +0800

Source feedback: `docs/project-feedback/batches/20260501-101028-feedback-batch.md`

Manager issue: #2121

## Result

- Status: completed
- Target repos: 11
- Worker lane: `ops-pages`
- Public URL verification: 11/11 returned HTTP 200
- Worktree verification: 10/11 target game workdirs clean; `JiaoshoujiaQiangxiu` gained unrelated post-Pages `integration` dirty files after the Pages fix was accepted and pushed
- Scope: deployment workflow, static entry/package wiring where needed, and README public/local run instructions only
- Direction Lock: no gameplay or Direction Lock changes accepted

## URL Verification

Command:

```bash
curl -L -s -o /dev/null -w '%{http_code}' https://dengxiaocheng.github.io/<repo>/
```

| Repository | Pages URL | HTTP | Pages fix commit | Current verified HEAD |
| --- | --- | --- | --- | --- |
| `BabelMicrogame-BingpengYezhen` | https://dengxiaocheng.github.io/BabelMicrogame-BingpengYezhen/ | 200 | `037f99a` | `037f99a` |
| `BabelMicrogame-DengyouFenpei` | https://dengxiaocheng.github.io/BabelMicrogame-DengyouFenpei/ | 200 | `a9e505d` | `a9e505d` |
| `BabelMicrogame-DuantiYunliao` | https://dengxiaocheng.github.io/BabelMicrogame-DuantiYunliao/ | 200 | `2bd03e2` | `2bd03e2` |
| `BabelMicrogame-GongpaiJiaohuan` | https://dengxiaocheng.github.io/BabelMicrogame-GongpaiJiaohuan/ | 200 | `2a4e20a` | `2a4e20a` |
| `BabelMicrogame-HeizhangXiaoce` | https://dengxiaocheng.github.io/BabelMicrogame-HeizhangXiaoce/ | 200 | `92faf0e` | `92faf0e` |
| `BabelMicrogame-HuijiangPeibi` | https://dengxiaocheng.github.io/BabelMicrogame-HuijiangPeibi/ | 200 | `b2601c6` | `b2601c6` |
| `BabelMicrogame-JiaoshoujiaQiangxiu` | https://dengxiaocheng.github.io/BabelMicrogame-JiaoshoujiaQiangxiu/ | 200 | `0175e9a` | `0175e9a` |
| `BabelMicrogame-ShuiyuanLunzhi` | https://dengxiaocheng.github.io/BabelMicrogame-ShuiyuanLunzhi/ | 200 | `5e6dbf0` | `5e6dbf0` |
| `BabelMicrogame-TiantiZuihouYiji` | https://dengxiaocheng.github.io/BabelMicrogame-TiantiZuihouYiji/ | 200 | `c601cd0` | `ffb86f8` |
| `BabelMicrogame-TibanMingdan` | https://dengxiaocheng.github.io/BabelMicrogame-TibanMingdan/ | 200 | `4ad7f2c` | `4ad7f2c` |
| `BabelMicrogame-ZhuiwuYujing` | https://dengxiaocheng.github.io/BabelMicrogame-ZhuiwuYujing/ | 200 | `c1bd2c5` | `c1bd2c5` |

## Notes

- `TiantiZuihouYiji` received a later content commit after the Pages fix; the Pages fix is included in the verified current HEAD.
- `JiaoshoujiaQiangxiu` received unrelated post-Pages `index.html` and `src/game.js` changes from its `integration` lane after the Pages fix; the Pages fix commit remains pushed, and the public URL returns 200.
- `HuijiangPeibi` required a follow-up Pages fix because the workflow used `npm ci`; the accepted fix added `package-lock.json`.
- `ShuiyuanLunzhi` had a pre-existing `npm test` failure in the worker report (`21/22 pass`) unrelated to the Pages/README change; Pages verification passed.
- `ZhuiwuYujing` duplicate retry state was cleaned after accepting and pushing `c1bd2c5`; `npm test` was re-run by manager and passed 28/28.

## Final Validation

- `sh /home/openclaw/babel-runtime/scripts/claudecode_manager_status.sh`: target Pages workers done; one unrelated post-Pages `JiaoshoujiaQiangxiu` integration dirty lane remains blocked for normal queue handling
- `git diff --check`: passed
