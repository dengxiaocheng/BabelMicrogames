# 节点运行环境

本文描述节点入口、Termux 运维角色、代理、启动器和服务器侧等待语义。

## 范围

本文覆盖：

- Termux 作为主要运维入口
- Windows 本地作为并行运维入口
- 登录快捷方式与节点角色
- Codex 启动和代理约定
- Termux 启动器规则
- 节点本地与仓库托管的边界
- 服务器侧等待语义

## Termux 的运维角色

Termux 应被视为这个系统面向操作人员的入口，而不只是一个一次性启动脚本所在的位置。

在实际操作中，Termux 用来：

- 登录运行时节点
- 在相关用户之间跳转
- 重写本地辅助脚本，例如 `~/bin/s`
- 在需要时重写 Babel 新会话入口 `~/bin/m`
- 在项目目录中启动 Codex
- 脱离主工作站时执行快速运维检查

## Windows 本地脚本入口

Windows 本地机器上的运行脚本，后续统一按与 Termux 类似的方式收口：

- canonical 脚本放 `scripts/windows/`
- canonical 运维说明放 `docs/operations/WINDOWS_LOCAL.md`

不要把 Windows 本地运行脚本散落在：

- 仓库根目录
- 零散聊天命令片段
- 只存在于某一台本地机器、仓库里没有归档的临时文件

如果某个 Windows 本地脚本需要反复执行，或复制时容易损坏格式，也应先归档到 `scripts/windows/`，再交付给用户。

## Thread ID 切换规则

以后切换 `s / m` 对应的固定 Codex thread id，不再手工改多处文件。

统一入口：

```bash
sh scripts/switch_launcher_thread.sh s <new-thread-id>
sh scripts/switch_launcher_thread.sh m <new-thread-id>
```

当前脚本会自动更新对应 lane 的：

- Termux 重写脚本
- 离线修复 / 重装脚本
- 当前 `.codex-runtime` 状态文件
- `s` 对应的远端 legacy alias

也就是说，后续换 id 的标准流程应收敛为：

1. 找到新的目标 thread id
2. 执行 `scripts/switch_launcher_thread.sh s|m <new-thread-id>`
3. 再让本地入口重新执行对应的重写 / 重装脚本

## 已知的 Termux 登录快捷方式

当前 Termux 环境预计存在以下运维快捷方式：

- `x`
  以 `root` 身份登录远端节点

- `c`
  以 `openclaw` 身份登录远端节点

- `d`
  与 Claude 用户 / 会话工作流相关的登录路径

这些只是操作人员便利工具，不是仓库托管的二进制程序。

## Codex 启动和代理

仓库不应在可复用的启动脚本里硬编码代理端口。

如果运行时节点上的 Codex 需要代理，应通过下列文件配置：

- `/home/openclaw/.config/babel-runtime/codex-proxy.env`

预期形状例如：

```bash
export HTTP_PROXY=http://127.0.0.1:10809
export HTTPS_PROXY=http://127.0.0.1:10809
export ALL_PROXY=socks5://127.0.0.1:10809
```

## Termux 启动器规则

Termux 启动器应当：

- 能在 Termux 运维工作流中正常工作
- 要么直接 SSH 到运行时节点，要么与当前快捷方式保持一致
- 在需要时切换到 `openclaw`
- 进入 `/home/openclaw/babel-runtime`
- 如果存在可选代理环境文件，就先 source 它
- 用目标安全 / reasoning 参数启动 Codex

另外增加一条交付规则：

- 面向 Termux 的可复用脚本、重写脚本、排障命令，默认先提交到仓库 `scripts/`
- 如果只是让用户执行某个操作，优先给出 `scripts/...` 的稳定路径，再给一条最短执行命令
- 如果用户明确说明当前手机 Termux 不能联网、不能直接取 GitHub raw、也不能直接跑服务器路径，那么除了先提交 `scripts/` 里的 canonical 脚本，还必须同时给出一份完整可手工复制的脚本文本
- 这种离线/手工复制场景下，不能只回复远程路径、raw 链接或服务器侧命令

原因是手机 / Termux 的真实约束有两种：

- 平时：聊天复制容易损坏，所以 canonical source 应该先落仓库脚本
- 离线时：用户又确实可能只能手工复制，因此交付层必须额外给出完整可复制版本

## 一次性启动器

当前仓库标准是使用：

- `scripts/termux_rewrite_s_one_shot.sh`
- `scripts/termux_patch_existing_s_thread.sh`
- `scripts/termux_reinstall_m_offline.sh`
- `scripts/termux_rewrite_m_one_shot.sh`
- `scripts/termux_rewrite_probe_one_shot.sh`
- `scripts/termux_commands.sh`
- `scripts/termux_check_local_state.sh`
- `scripts/termux_exit_probe.sh`
- `scripts/termux_manual_remote.sh`

它用于一次性重写 `$PREFIX/bin/s`，之后日常运维直接执行：

```bash
s
```

如果是在手机 Termux 本地直接安装，不要依赖服务器路径，优先执行：

```bash
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_s_one_shot.sh | sh
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_m_one_shot.sh | sh
```

而 `scripts/termux_check_local_state.sh` 用于在 Termux 本地快速排查：

- 当前是否还有连向节点的本地 `ssh` 客户端
- `$TMPDIR/codex-manual/` 下是否残留临时文件
- 当前 `$PREFIX/bin/s` 和 `$PREFIX/bin/m` 是否存在或被重写

Termux 本地只保留原始脚本和命令清单脚本：

- `scripts/termux_rewrite_s_one_shot.sh`
- `scripts/termux_patch_existing_s_thread.sh`
- `scripts/termux_reinstall_m_offline.sh`
- `scripts/termux_rewrite_m_one_shot.sh`
- `scripts/termux_rewrite_probe_one_shot.sh`
- `scripts/termux_check_local_state.sh`
- `scripts/termux_exit_probe.sh`
- `scripts/termux_commands.sh`

如果手机本地的 `$PREFIX/bin/s` 已经存在，但内部 thread id 没有切到仓库当前要求的新值，可以直接在手机本地执行：

```bash
sh /home/openclaw/babel-runtime/scripts/termux_patch_existing_s_thread.sh
```

如果手机本地的 `$PREFIX/bin/m` 还是旧 launcher，或者你怀疑它没有带上当前的断开/收尸逻辑，可以直接在手机本地执行：

```bash
sh /home/openclaw/babel-runtime/scripts/termux_reinstall_m_offline.sh
```

`scripts/termux_exit_probe.sh` 用于在 Termux 本地直接验证“通知栏 Exit 是否真的结束了当前脚本”：

- 前台运行后，它会把自己的 `pid / ppid / pgid / tty` 和每秒 tick 持续写进 `$TMPDIR/codex-manual/termux-exit-probe-*.log`
- 你按平时的方式点通知栏 `Exit`
- 之后重新进 Termux 读取该日志，就能判断脚本是在 `Exit` 时终止，还是继续活到了 `Exit` 之后
- 脚本已经内置固定子命令：
  - `run`
  - `latest`
  - `list`
  - `clean`
- 当前推荐执行命令如下：

```bash
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh clean
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh run
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh latest
```

如果你不想在手机本地保留仓库路径，可以先执行：

```bash
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_probe_one_shot.sh | sh
```

然后直接在 Termux 本地用：

```bash
probe clean
probe run
probe latest
probe list
```

其中：

- `s`
  恢复当前 `babel-runtime` 线程，走 `manual-resume`

- `m`
  恢复 Babel 的固定专用 Codex 会话，进入 `/home/openclaw/Babel`

`m` 服务的是 Babel 仓库，但其 Termux 节点入口由当前 online 仓库统一托管，避免同类启动器散落在多个仓库里。它和 `s` 的格式、权限和入口形态保持一致，只是绑定的是另一条固定线程。

当前标准 Termux 启动器还带一个额外约束：

- `s` / `m` 在远端都会把手动会话放进独立进程组
- `s` / `m` 都会优先执行仓库内预编译的 `.codex-runtime/bin/babel-issue-bridge`
- 如果该二进制不存在，启动器只在首轮启动时 `go build` 一次，再复用后续启动
- `s` / `m` 在 Termux 本地都只依赖 `sh`
- `s` / `m` 会用一个前台子壳先写入本地 `ssh` pidfile，再 `exec ssh ...`，让真正的 `ssh` 仍然占用前台 tty，避免后台 `ssh` 破坏交互显示
- `s` / `m` 在 Termux 本地都会启动一个轻量 heartbeat，按固定间隔对远端 `touch-manual-lease`
- `s` / `m` 还会启动一个轻量 watchdog；watchdog 同时观察“当前脚本壳是否还活着”和“本地 tty 是否已经挂断”，不参与业务逻辑
- `s` / `m` 真正运行的 `ssh` 客户端不会继承这个 FIFO 写端，因此一旦状态栏 `Exit` 先结束本地脚本壳，watchdog 就会把本地 `ssh` 直接杀掉
- 如果 SSH 连接被服务器确认断开，远端对应的 `manual-resume -> codex resume` 进程组应被一起杀掉
- watcher 仍然保留；被清掉的只是不该残留的交互式手动客户端

当前节点还额外依赖 SSH 服务端的 dead-client 探测：

- `/etc/ssh/sshd_config.d/90-babel-runtime-session.conf`
- `TCPKeepAlive yes`
- `ClientAliveInterval 15`
- `ClientAliveCountMax 2`

这意味着“状态栏 `Exit` 已结束本地 Termux session”之后，标准路径不再只是等待远端 `sshd` 自己发现死连接：

- 第一层：本地 watchdog 应先直接杀掉 `ssh` 客户端
- 第二层：如果本地 watchdog 没跑到，远端仍有 `ClientAlive*` 作为兜底清理

因此，目标语义已经从“只靠 SSH keepalive 在有界时间内清理”收紧成：

- 正常情况下，本地 `Exit` 应很快带走本地 `ssh`
- 就算本地 `ssh` 没立刻结束，heartbeat 停止后，远端 lease 也会在 TTL 到期后主动收掉手动链
- 异常情况下，服务器仍会在约 `30` 秒量级内确认死连接并收尾

因此，重新执行一次 `scripts/termux_rewrite_s_one_shot.sh` 和 `scripts/termux_rewrite_m_one_shot.sh` 后，“直接退出 Termux”可以视为一种有效的快速退出方式，但其收尸语义应理解为：

- 本地先由 watchdog 直接结束 `ssh`
- 远端再由 SSH keepalive 在有界时间内完成剩余清理

如果希望提前消除首次启动的编译开销，可在仓库里先执行：

```bash
go run ./cmd/babel-dev install-ops-binaries
```

它会把当前仓库高频运维入口预编译到 `.codex-runtime/bin/`。当前默认安装的是 `babel-issue-bridge`。

`m` 现在重新回到“纯入口”职责：

- 它只负责进入 Babel 的固定专用线程
- `babel-cpp` 的 heartbeat 由 Babel 仓库自己的 `manual-resume / watch / open-stage / close-active` 生命周期直接写入 collaboration MCP

这样 `m` 不是唯一状态来源，Babel 仓库自己也能在 watcher 恢复、manual-resume 进入/退出、stage issue 进入 waiting 时直接刷新 `babel-cpp` 状态。

同时要注意：

- `m` 的存在不代表当前这条 `online` 会话应继续直接修改 `/home/openclaw/Babel`
- 它只是进入 Babel 专用会话的统一节点入口
- 除非用户明确要求当前会话去改 Babel，否则当前会话应继续聚焦 `/home/openclaw/babel-runtime`

## 节点本地与仓库托管的边界

仓库托管内容：

- 启动器形状
- 运维约定
- 文档

节点本地内容：

- 代理端口选择
- 实际代理服务进程
- secrets
- 机器特有环境细节
- `x`、`c`、`d` 这类快捷方式

## 当前节点基线

以下内容是当前服务器节点的运维基线快照，供后续排障和节点漂移检查使用。它描述的是“当前节点应该长什么样”，而不是项目 canonical state。

基线时间：

- `2026-04-23`

机器与系统：

- hostname: `hcss-ecs-f7e9`
- virtualization: `kvm`
- OS: `Ubuntu 24.04.4 LTS`
- kernel: `Linux 6.8.0-101-generic`
- architecture: `x86_64`
- root filesystem: `/dev/vda1`
- root disk capacity: `40G`

当前观测到的长期节点特征：

- 服务器侧业务入口由 `babel-wechatd` 提供，对外监听 `*:8080`
- SSH 入口监听 `*:22`
- `sshd` 当前显式启用了 dead-client 探测：
  - `/etc/ssh/sshd_config.d/90-babel-runtime-session.conf`
  - `TCPKeepAlive yes`
  - `ClientAliveInterval 15`
  - `ClientAliveCountMax 2`
- 当前没有 `openclaw` 用户级 `systemd --user` service
- 当前没有用户级 `crontab`
- 当前也没有仓库自带的本地 `systemd user unit` 文件

当前已知的 repo-local 运维状态目录：

- `/home/openclaw/babel-runtime/.codex-runtime/`
- `/home/openclaw/Babel/.codex-runtime/`
- `/home/openclaw/.codex-runtime/collab/`

当前 `babel-runtime` repo-local `.codex-runtime/` 已知用途：

- `bin/`
  预编译运维入口，例如 `babel-issue-bridge`
- `github-token.env`
  节点本地 GitHub token 文件
- `issue_bridge_state.json`
  当前阶段 issue 状态
- `thread_control.json`
  当前线程控制状态
- `issue_bridge_events.jsonl`
  issue bridge 结构化事件日志
- `wechat-memory/`
  微信运行态的 file-backed working memory

说明：

- 这些目录和文件属于节点运维状态，不是项目 canonical state
- 内容会随手动会话、watcher、微信运行态变化
- 结构可以记录在文档里，具体运行值应通过现场检查获得

## 当前节点代理基线

当前 Codex 代理环境文件：

- `/home/openclaw/.config/babel-runtime/codex-proxy.env`

当前可见内容基线：

```bash
export HTTP_PROXY=http://127.0.0.1:10809
export HTTPS_PROXY=http://127.0.0.1:10809
export ALL_PROXY=socks5://127.0.0.1:10808
```

当前代理与相关本地端口观测：

- `127.0.0.1:10808`
  `v2ray` 提供的本地 SOCKS/相关入口
- `127.0.0.1:10809`
  `v2ray` 提供的本地 HTTP 代理入口
- `127.0.0.1:29338`
  `uniagentd` 占用
- `127.0.0.1:29339`
  `uniagentd` 占用

当前可见代理/相关进程基线：

- `v2ray`
  进程形态：`/usr/local/bin/v2ray run -config /usr/local/etc/v2ray/config.json`
- `v2ray.service`
  system service 状态：`enabled`
- `uniagentd`
  进程路径：`/usr/local/uniagentd/bin/uniagentd`

当前节点只把以下代理信息视为可写入仓库的运维基线：

- 代理环境文件路径
- 本地监听端口
- service / 进程名称
- 可见配置文件路径

不要把以下内容提交进仓库：

- 上游代理账号或凭据
- 远端代理出口地址与敏感订阅内容
- 任何 token、secret 或私有节点标识

## 节点迁移清单

如果目标是把当前服务器节点迁移到另一台机器，当前最小迁移单元不应靠聊天记忆，而应按下列清单逐项恢复。

### 必须迁移的节点本地配置

- `/etc/ssh/sshd_config.d/90-babel-runtime-session.conf`
  节点对 Termux/SSH 手动会话的 dead-client 探测配置
- `/home/openclaw/.config/babel-runtime/codex-proxy.env`
  Codex 代理环境文件
- `/usr/local/etc/v2ray/config.json`
  当前 `v2ray` 使用的本地配置文件
- `/home/openclaw/babel-runtime/.codex-runtime/github-token.env`
  `babel-runtime` 仓库的 GitHub token 文件
- `/home/openclaw/Babel/.codex-runtime/github-token.env`
  `Babel` 仓库的 GitHub token 文件

### 需要保留或重建的节点目录

- `/home/openclaw/babel-runtime/.codex-runtime/`
- `/home/openclaw/Babel/.codex-runtime/`
- `/home/openclaw/.codex-runtime/collab/`

说明：

- `github-token.env`、代理配置、任何第三方凭据都属于节点 secret；应通过安全复制或重新签发恢复，不应直接写入 git
- `issue_bridge_state.json`、`thread_control.json`、`issue_bridge_events.jsonl` 属于运行态文件，可迁移，也可在新节点冷启动后重建
- `.codex-runtime/bin/babel-issue-bridge` 这类二进制不是必须人工复制的资产；可通过 `go run ./cmd/babel-dev install-ops-binaries` 重建

### 需要确认的系统服务

- `v2ray.service`
  当前状态应为 `enabled`
- `sshd`
  机器远程入口，当前监听 `*:22`

当前没有要求迁移的内容：

- `openclaw` 用户级 `systemd --user` service
- 用户级 `crontab`
- repo 内托管的本地 `systemd user unit`

### 迁移后应重做的仓库级准备

在 `/home/openclaw/babel-runtime`：

```bash
go run ./cmd/babel-dev install-hooks
go run ./cmd/babel-dev install-ops-binaries
bash scripts/termux_rewrite_s_one_shot.sh
bash scripts/termux_rewrite_m_one_shot.sh
```

在 `/home/openclaw/Babel`：

```bash
mkdir -p .codex-runtime/bin
go build -o .codex-runtime/bin/babel-issue-bridge ./cmd/babel-issue-bridge
```

### 迁移后应验证的本地端口

- `127.0.0.1:10808`
- `127.0.0.1:10809`
- `127.0.0.1:29338`
- `127.0.0.1:29339`
- `*:22`
- `*:8080`

## 节点清单命令

为了让当前节点信息可以重复获取，后续运维应优先使用固定命令，而不是重新手写排查步骤。

机器和系统：

```bash
hostnamectl
uptime
df -h /
free -h
```

监听端口与归属：

```bash
sudo ss -lntup
sudo ss -lntup | rg '10808|10809|29338|29339|8080|:22'
```

代理基线：

```bash
sed -n '1,120p' /home/openclaw/.config/babel-runtime/codex-proxy.env
systemctl list-unit-files | rg 'v2ray|uniagent|proxy'
ps -fp $(sudo ss -lntup | awk '/10808|10809|29338|29339/ {gsub(/users:\\(\\(|\\)\\)/,\"\"); if (match($0,/pid=[0-9]+/)) {print substr($0,RSTART+4,RLENGTH-4)}}' | sort -u | tr '\n' ' ')
```

会话与运维状态：

```bash
tmux ls
ps -eo pid,user,ppid,pgid,sid,etimes,stat,cmd --sort=etimes | rg 'codex|babel-issue-bridge|claude|tmux|sshd: .*@pts'
go run ./cmd/babel-issue-bridge status
(cd /home/openclaw/Babel && go run ./cmd/babel-issue-bridge status)
```

repo-local 与节点级 `.codex-runtime`：

```bash
find /home/openclaw/babel-runtime/.codex-runtime -maxdepth 3 -mindepth 1 \\( -type f -o -type d \\) | sort
find /home/openclaw/Babel/.codex-runtime -maxdepth 3 -mindepth 1 \\( -type f -o -type d \\) | sort
find /home/openclaw/.codex-runtime -maxdepth 3 -mindepth 1 \\( -type f -o -type d \\) | sort
```

## 当前环境变量与私密信息快照

以下内容是当前 `openclaw` 运行时 shell 的直接快照，用于节点配置迁移。这里记录的是当前机器现场值，而不是抽象说明。

观测时间：

- `2026-04-23`

### 当前环境变量快照

```bash
ALL_PROXY=socks5://127.0.0.1:10808
BABEL_GITHUB_TOKEN=<redacted>
CODEX_CI=1
CODEX_MANAGED_BY_NPM=1
CODEX_THREAD_ID=019dbcf2-d632-7770-9cbc-ae1f963eb2e0
COLORTERM=
GH_PAGER=cat
GIT_PAGER=cat
HISTSIZE=10000
HISTTIMEFORMAT=%F %T openclaw 
HOME=/home/openclaw
HTTPS_PROXY=http://127.0.0.1:10809
HTTP_PROXY=http://127.0.0.1:10809
LANG=en_US.UTF-8
LC_ALL=C.UTF-8
LC_CTYPE=C.UTF-8
LOGNAME=openclaw
MAIL=/var/mail/openclaw
NO_COLOR=1
OLDPWD=/home/openclaw
PAGER=cat
PATH=/home/openclaw/.local/bin:/home/openclaw/.codex/tmp/arg0/codex-arg0AbJiTh:/home/openclaw/.nvm/versions/node/v22.22.1/lib/node_modules/@openai/codex/node_modules/@openai/codex-linux-x64/vendor/x86_64-unknown-linux-musl/path:/home/openclaw/sdk/go/bin:/home/openclaw/.local/bin:/home/openclaw/.nvm/versions/node/v22.22.1/bin:/home/openclaw/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin
PWD=/home/openclaw/babel-runtime
SHELL=/bin/bash
SHLVL=1
TERM=xterm-256color
USER=openclaw
XDG_DATA_DIRS=/usr/local/share:/usr/share:/var/lib/snapd/desktop
_=/usr/bin/env
```

说明：

- 当前没有在 `~/.bashrc`、`~/.profile`、`~/.bash_profile`、`~/.zshrc` 中发现代理或 token 相关环境变量注入
- 当前 live shell 中的 `BABEL_GITHUB_TOKEN` 来自 repo-local token 文件，而不是 shell init
- 当前 live shell 中的 `HTTP_PROXY / HTTPS_PROXY / ALL_PROXY` 来自 `/home/openclaw/.config/babel-runtime/codex-proxy.env`

### 当前私密文件实际内容

`/home/openclaw/babel-runtime/.codex-runtime/github-token.env`

```bash
export BABEL_GITHUB_TOKEN=<redacted>
```

`/home/openclaw/Babel/.codex-runtime/github-token.env`

```bash
export BABEL_GITHUB_TOKEN=<redacted>
```

`/home/openclaw/.config/babel-runtime/codex-proxy.env`

```bash
export HTTP_PROXY=http://127.0.0.1:10809
export HTTPS_PROXY=http://127.0.0.1:10809
export ALL_PROXY=socks5://127.0.0.1:10808
```

### 当前私密文件权限

- `/home/openclaw/babel-runtime/.codex-runtime/github-token.env`
  `0600`
- `/home/openclaw/Babel/.codex-runtime/github-token.env`
  `0600`
- `/home/openclaw/.config/babel-runtime/codex-proxy.env`
  当前观测权限：`0664`

## 服务器侧等待语义

如果：

- 当前活动 issue 还没有关闭
- 且用户也没有手动打开 `s`

那么系统应处于“无活跃 Codex 客户端、仅 watcher 睡眠等待事件”的状态。

也就是说，不需要 Termux 常开；真正的 Codex `resume` 客户端只会在两种唤醒源出现时启动：

- issue 被评论后关闭
- 用户手动执行 `s`

## 双节点协同语义

当前默认存在两类节点角色：

- 服务器节点
  挂 watcher、保存当前 Codex 线程的等待点，并在需要时继续执行

- 实时开发节点
  例如用户自己实时操控的 Windows 电脑；它可以直接使用同一套代码仓做开发和提交，但不需要在本地重复 issue watcher 仪式

双节点协同时，应遵守：

- 实时开发节点活跃期间，服务器节点应保持 watcher-only 或 idle，不要同时再跑一个活跃的实现客户端
- 实时开发节点可以直接在自己的仓库副本里编辑、提交、推送
- 当实时开发节点推送后，需要把执行权交还服务器节点时，应使用当前已经打开的 stage issue 作为 handoff 通道
- handoff comment 应明确告诉服务器节点先同步代码，再继续执行

当前保留一个标准短指令：

- `拉取`
  代表“先同步服务器当前工作区到当前跟踪分支，读取刚拉下来的代码变动，再继续当前任务”

推荐 comment 形状例如：

```text
已在 Windows 节点推送到 origin/main。
请先执行 `git pull --ff-only origin main`，确认工作区干净后继续下一步。
```

如果用户只写 `拉取`，服务器节点应按默认同步路径处理：

1. 先检查当前分支和 upstream
2. 优先对当前分支执行 fast-forward 同步
3. 同步后读取新增 commit 和关键 diff
4. 然后继续当前任务

如果实际 handoff 需要指定分支、远端或 commit，也应直接写在 comment 里，而不是假设服务器节点会自动猜测；只有在 comment 没有额外说明时，`拉取` 才表示这条默认路径。

## 推荐的服务器常驻形态

当前最简单的形态是：

1. 在服务器上启动 watcher
2. 让 watcher 常驻在 `tmux` 会话里
3. 用户通过 GitHub issue、另一台节点的 handoff comment，或手动 `s` 触发继续执行

如果后续需要更稳定的长期运行，可以再包进：

- `systemd --user`
- `supervisord`
- 其他节点本地守护器
