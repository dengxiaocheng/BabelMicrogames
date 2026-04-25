# AGENTS.md

This file defines repository-level working rules for the ClaudeCode manager.

It exists to reduce drift while the system is growing quickly.

## Repository Identity

This repository is the dedicated Babel microgame factory and ClaudeCode manager lane.

Its remote repository is:

- `dengxiaocheng/BabelMicrogames`

That repository is for manager documentation, scripts, rules, and manager-level coordination. It is not a microgame source repository.

This is the only workdir for the Codex process that manages ClaudeCode workers:

- `/home/openclaw/claudecode-manager`

The Codex manager must not be resumed inside a game source workdir.

Game source workdirs are separate and must stay game-only:

- `/home/openclaw/babel-microgames/peigei-ri`
- `/home/openclaw/babel-microgames/yejian-qidao`
- `/home/openclaw/babel-microgames/gongtou-dianming`

Manager responsibilities:

- keep `/home/openclaw/claudecode-manager/.codex-runtime/microgame_manager_state.json` refreshed as the manager-level index
- read worker reports from game `.codex-runtime/claudecode_workers/`
- review diffs inside the target game workdir
- run game tests from the target game workdir
- commit and push only the target game repository
- mark worker status `done / rework / cancelled`
- open the next game stage issue with `--resume-workdir /home/openclaw/claudecode-manager`
- start the next ClaudeCode worker using the target game's fixed Claude session
- use `/home/openclaw/babel-runtime` as the default Go issue bridge implementation through `scripts/claudecode_issue_bridge.sh`

Manager must not:

- implement game source changes directly unless repairing manager-induced breakage
- add manager/runtime code to a game repository
- use `BabelOnline-GoCpp`, `Babel`, or `BabelMicrogames` as a per-game worker queue
- use `/home/openclaw/claudecode-manager/.codex-runtime/claudecode_workers.json` as dispatch truth
- let issue watcher resume Codex inside a game workdir

Dispatch truth is always the target game workdir:

- `/home/openclaw/babel-microgames/<game>/.codex-runtime/claudecode_workers.json`

The manager repository may keep an aggregate state file, but it must not own per-game worker packets or worker status.

Manager scripts may keep ClaudeCode-specific orchestration in this repository, but `worker-*`, `open-stage`, `manager-handoff`, watcher events, and `BABEL_ISSUE_BRIDGE_EVENT_HOOK` must flow through the `s` Go bridge by default. Do not re-expand the local copied `cmd/babel-issue-bridge` into an active second implementation.

Its GitHub issue namespace and source repositories must be separate from the long-lived `s` / `m` repositories.

Use `BabelMicrogames` for manager-level notes and repository ownership. Use `BabelMicrogame-*` repositories for per-game source, issues, worker queues, and GitHub Pages.

Game worker repositories must use this prefix:

- `dengxiaocheng/BabelMicrogame-*`

ClaudeCode manager scripts must not open worker issues in:

- `dengxiaocheng/BabelOnline-GoCpp`
- `dengxiaocheng/Babel`

Those repositories are reserved for the existing runtime/manual session lanes.

This repository was forked from the new greenfield runtime for Babel.

It should:

- inherit validated product capabilities
- not inherit legacy implementation debt
- keep deterministic state outside the LLM
- support future Godot synchronization
- support later C++ extraction only after Go-first validation

## Core Working Principle

The repository should evolve under this rule:

- inherit features
- rebuild implementation
- keep boundaries explicit
- keep docs and code synchronized

## Tooling Language Rule

Repository implementation and repository-level tooling should default to:

- `Go`
- `C++`

Do not introduce new Python components unless the user explicitly approves a temporary exception.

Existing Python files should be treated as migration debt and be folded back into Go over time rather than expanded further.

## Script Hygiene Rule

Repository shell scripts under `scripts/` should stay terse.

That means:

- do not add explanatory comment blocks inside scripts
- do not embed chat-style guidance or long instructional prose in script output
- keep inline output limited to the minimum commands or state needed to operate the script
- if usage needs to be documented at length, put that explanation in repository docs instead of the script body

## Termux Delivery Rule

For Termux-facing operational usage, do not rely on chat-only command snippets as the primary delivery vehicle.

Default rule:

- if the user is expected to run the command more than once, commit it under `scripts/`
- if the command is easy to break when copied from chat on mobile, commit it under `scripts/`
- if a launcher or rewrite flow changes, update the repository script first and then tell the user to execute that script
- prefer replying with a stable script path plus a single invocation command, instead of pasting a large inline shell block

When the user explicitly says their current Termux environment cannot rely on network fetches or direct server-side paths, add one more mandatory rule:

- still commit the canonical script under `scripts/` first
- then also provide a complete hand-copyable script body in chat
- do not answer with only a remote path, raw GitHub URL, or server-side execution command

In other words:

- canonical source of truth stays in `scripts/`
- but the user-facing delivery for offline/mobile Termux must include a full manual-copy version

This rule exists because Termux/mobile copy behavior is fragile and chat formatting is not a reliable transport for operational shell content.

## Windows Local Script Rule

Windows local runtime scripts should follow the same repository-first discipline as Termux, but with their own archive location.

Default rule:

- reusable Windows local scripts must be committed under `scripts/windows/`
- operational explanation for those scripts must be reflected under `docs/operations/`
- do not leave Windows local launchers or recovery scripts only in chat
- do not scatter Windows local run scripts in the repository root

If the user needs a hand-copyable Windows version because local copy/paste is the practical path:

- still commit the canonical version under `scripts/windows/` first
- then provide a complete manual-copy version in chat

Canonical documentation for this lane lives in:

- `docs/operations/WINDOWS_LOCAL.md`

## Canonical Documentation Rule

The repository has three documentation zones:

- `docs/`
  canonical documentation only

- `docs/planning/`
  planning references and earlier design inputs

- `docs/incoming/`
  staging area for new design and requirement drops

- `plan/`
  active plan files created on user request for the current execution cycle

Do not place new design markdown files in the repository root.

Canonical docs in `docs/` should default to Simplified Chinese prose.

English is still appropriate for:

- code identifiers
- interface/type names
- protocol keywords
- file paths and command examples

Avoid switching canonical docs back to English-first prose unless there is a clear repository-level reason.

## Plan File Rule

When the user asks for a new plan file, create it under:

- `plan/`

Use this directory for:

- active execution plans
- short-to-medium operational plans
- migration plans
- refactor plans that need file-backed iteration

Do not treat `plan/` as a replacement for canonical docs:

- once a plan becomes stable design or operational policy, reflect it back into `docs/`
- keep `docs/planning/` for earlier planning references and historical planning inputs
- keep `plan/` for current working plans created for the user

## Incoming Design Rule

When new design or requirement documents are added:

1. they should enter through `docs/incoming/`
2. they should use `docs/incoming/TEMPLATE.md` when practical
3. they should be logged in `docs/REQUIREMENT_CHANGELOG.md`
4. they should be reviewed against `docs/REQUIREMENT_SYNC_CHECKLIST.md`
5. canonical docs should be updated before or alongside implementation work

Additional ownership constraint for `docs/incoming/`:

- only the user may add new files into `docs/incoming/`
- Codex must not proactively create new incoming files there
- Codex must not proactively process newly noticed incoming files on its own
- Codex may only merge an incoming document into existing canonical docs after the user explicitly asks to process that document
- after merging, the original incoming file should be archived rather than left as an active loose drop

## Required Canonical Reflection

Any meaningful design change must be checked against:

- `docs/FEATURE_INHERITANCE.md`
- `docs/SYSTEM_TARGET_ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `docs/SUBSYSTEM_BOUNDARIES.md`
- `docs/INTERFACES.md`
- `docs/STORAGE_SCHEMA.md`
- `docs/TESTING.md`

If system direction changes, at least one canonical doc should change.

## Changelog Rule

Important design or requirement updates must not live only in prose or chat history.

They should be tracked in:

- `docs/REQUIREMENT_CHANGELOG.md`

Status should be explicit, for example:

- `incoming`
- `reviewed`
- `accepted`
- `rejected`
- `archived`
- `implemented`

## Babel / C++ Alignment Rule

Stable world rules, deterministic constraints, and future C++ ownership boundaries are anchored to the Babel-side core direction.

This repository may iterate rapidly on experience and orchestration, but it must not casually redefine:

- core world logic
- deterministic simulation assumptions
- canonical state semantics
- future C++ ownership boundaries

Those changes are architecture-significant and must be documented explicitly.

## Cross-Session Collaboration Rule

This `online` session may coordinate with the separate Babel / C++ session, but it must not assume hidden shared model context.

Cross-session alignment should use explicit, file-backed operational coordination such as:

- stage issues
- structured handoff comments
- node-local collaboration MCP state
- documented Go/C++ boundary contracts

The collaboration MCP is operational coordination only.

It may track:

- current boundary contract
- session heartbeats
- claimed scopes
- handoff records
- progress checkpoints

It must not become:

- canonical runtime state
- a replacement for repository documentation
- a hidden channel for silently changing C++ ownership boundaries

When the Babel / C++ session is acting as the managed implementation session for C++ work:

- this `online` session remains the default coordinator
- the Babel session should own the claimed C++ scopes after explicit handoff
- coordination should be recoverable from files and docs, not from chat memory alone

## Godot Synchronization Rule

This repository is a text-first gameplay validation and orchestration system.

It exists in part to validate rapidly changing realtime scene behavior before it is synchronized into Godot-facing requirements.

That means:

- stable world logic aligns with Babel/C++
- rapidly changing scene/pacing/interaction ideas may be explored here
- validated realtime gameplay should eventually be reflected into Godot-facing requirement assets

Do not treat experimental text behavior as finished product behavior until it is validated.

## LLM Rule

LLMs may:

- narrate
- summarize
- propose
- assist interpretation
- support experiments

LLMs must not:

- become the canonical state authority
- directly own deterministic settlement
- silently replace documented rules

## Runtime Authority Rule

The Go runtime is the orchestration authority.

It owns:

- lifecycle
- persistence ordering
- recovery
- event/checkpoint flow
- agent scheduling

Neither Codex nor Claude Code should become the hidden system supervisor.

## Agent Working Memory Rule

Long-lived agent memory should be file-backed and operational.

It is not canonical state.

Canonical truth remains in runtime-controlled persistence.

## Testing Rule

New subsystem work should include tests when feasible.

Priority test patterns:

- deterministic state tests
- replay tests
- restart simulation tests
- integration tests for new gateway/control-plane behavior

Do not add major behavior that cannot be traced or replayed.

## Boundary Rule

Follow `docs/SUBSYSTEM_BOUNDARIES.md`.

If a change blurs subsystem ownership, stop and clarify the boundary before continuing.

## Roadmap Rule

Follow `docs/ROADMAP.md` for phase ordering.

The immediate implementation queue is:

1. mode router
2. free chat orchestrator
3. project consult orchestrator
4. transport-facing response handling
5. file-backed agent working memory
6. requirement-management foundation

## Documentation Hygiene Rule

Before finishing significant work:

- update docs if system understanding changed
- ensure the changelog/checklist flow was followed for requirement changes
- keep `docs/INDEX.md` as a reliable entrypoint
- process and operational flow changes should also be reflected in docs, so later troubleshooting does not depend on chat history

When reading docs for a task:

- start from `docs/INDEX.md`
- select the smallest relevant document set for that task
- do not re-read the whole documentation set on every turn by default
- only expand the read set when the current documents are insufficient or conflicting

Before committing implementation or operations changes:

- use `docs/governance/DOC_MANIFEST.json` as the machine-readable sync matrix source
- ensure the current diff would pass `go run ./cmd/babel-dev check-docs-sync-guard`
- prefer keeping `.githooks/pre-commit` installed through `go run ./cmd/babel-dev install-hooks`

## Stage Commit Rule

Before stopping at the end of a small implementation stage and asking the user what to do next:

- commit the current stage
- push it to the active remote branch
- only then ask for the next step

Do not leave a finished stage only in local unpushed workspace state before asking for further direction.

## Stage Issue Rule

After a small implementation stage has been committed and pushed, and before waiting for further user direction:

- create a GitHub issue for that stage
- assign that issue to the user when possible
- put the stage report and the next-step decision request into the issue body
- mirror the same next-step decision request in the terminal close-out message, with the issue link
- assume the user may reply by commenting on that issue and then closing it

The issue is the preferred out-of-band continuation point when the user is not replying in the active terminal session.

If the user replies in the active terminal session while the current stage issue is still open:

- only treat that reply as a stage-handoff reply when the system is explicitly in a waiting state
- close that stage issue
- note that the active terminal has already resumed the work
- when adding a closing comment, prefer using the user's terminal reply verbatim by default
- do not trigger or wait for a watcher-driven resume for that same reply

If the user manually resumes the thread through the local terminal path (for example Termux `s`):

- treat that manual terminal as the new active waiting point
- close the previously open stage issue promptly
- do not leave both the GitHub issue and the live terminal session in a waiting state at the same time

## Issue Watcher Rule

The local issue watcher is operational memory and automation only.

It may:

- watch the latest stage issue
- detect a user comment plus issue closure
- resume the current Codex thread with that comment as the next instruction

It must not:

- become canonical project state
- replace runtime state or recovery
- silently mutate repository files without going through the resumed Codex thread

## Multi-Node Development Rule

The same repository may be worked on from another interactive node, for example a user-controlled Windows workstation.

In that case:

- the Windows or other interactive node may do live development without reproducing the full stage-issue ceremony locally
- the server-side Codex thread should remain in watcher-only or idle state until the handoff returns to the server node
- the current stage issue becomes the cross-node handoff channel rather than a requirement for the external node's local editing loop
- after the external node commits and pushes, the user should close the current stage issue with a top-level comment that tells the server node to pull or sync the new code and continue
- the reserved short handoff keyword `拉取` should be treated as a valid default sync instruction
- the resumed server-side thread must explicitly fetch and update its local worktree before making further changes
- after syncing, the resumed server-side thread must read the pulled code changes, continue the task, and later re-enter the normal stage issue flow
- do not keep two active implementation nodes modifying the same branch concurrently without an explicit handoff point
- once a Go/C++ boundary contract and non-overlapping claimed scopes are explicit, `online` and `babel-cpp` should proceed in parallel by default rather than waiting on each other
- handoff publication should be treated as a dependency declaration, not as an automatic stop-the-world barrier for the sender's own non-overlapping lane

## Cross-Repo Focus Rule

This Codex session is the default working session for:

- `/home/openclaw/babel-runtime`

It is not the default implementation session for:

- `/home/openclaw/Babel`

That means:

- the Termux shortcut `m` may launch or resume the Babel-side dedicated session
- the existence of `m`, Babel watcher state, or Babel stage issues does not authorize this session to continue editing the Babel repository by default
- unless the user explicitly tells this session to work in `/home/openclaw/Babel`, this session should stay focused on the `babel-runtime` repository
- Babel-side implementation, C++ refactors, and Babel stage-issue wakeups should be handled by the Babel-dedicated session

## Failure Modes To Avoid

Avoid these patterns:

- hidden state authority inside agents
- transport logic mixed into runtime core
- recovery logic diverging from normal execution
- validated gameplay living only in conversations
- root-level markdown sprawl returning
