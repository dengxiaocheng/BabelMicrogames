# Runtime Redesign Reset

## Metadata

- status: accepted
- date: 2026-04-22
- owner: repository owner + codex
- source: direct redesign request after implementation review
- scope:
  - runtime
  - gateway
  - llm
  - store-recovery
  - godot
  - requirement-system
- priority:
  - critical
- related_docs:
  - docs/ARCHITECTURE.md
  - docs/FEATURE_INHERITANCE.md
  - docs/SYSTEM_TARGET_ARCHITECTURE.md
  - docs/ROADMAP.md
  - docs/SUBSYSTEM_BOUNDARIES.md
  - docs/INTERFACES.md
  - docs/STORAGE_SCHEMA.md
  - docs/TESTING.md
- sync_targets:
  - docs/FEATURE_INHERITANCE.md
  - docs/SYSTEM_TARGET_ARCHITECTURE.md
  - docs/ROADMAP.md
  - docs/SUBSYSTEM_BOUNDARIES.md
  - docs/INTERFACES.md
  - docs/STORAGE_SCHEMA.md
  - docs/TESTING.md

## Summary

The repository should stop treating the current implementation skeleton as the target architecture. The new design resets the system around one authoritative runtime kernel, pluggable mode modules, explicit agent task supervision, projection and delivery separation, and an execution-centric persistence model.

## Problem

The current repository captures many correct constraints, but it does not yet express a strong operating model for how real product behavior should execute end to end.

Current problems:

- the package split reflects scaffolding rather than durable system boundaries
- solo and multiplayer are modeled too early as separate runtime centers
- LLM work is described mostly as rendering, which does not cover free chat or consultation
- persistence is not yet shaped around executions, retries, leases, and recoverable stage boundaries
- recovery, projection, delivery, and agent work are not yet modeled as one coherent execution lifecycle
- requirement capture exists as intent, but not yet as a first-class runtime-adjacent asset system

## Proposal

Reset the target architecture around these units:

- `kernel`
  The only online execution authority. It accepts normalized envelopes, enforces idempotency and leases, loads runtime state, invokes the correct mode module, persists the result, and resumes stalled work.

- `mode router`
  Resolves which mode module owns the current input based on runtime metadata, route policy, and transport-independent intent.

- `mode modules`
  Implement user-visible behavior such as free chat, project consult, solo scene, and room scene. They translate inbound intent into a mode command, decide required deterministic work, request agent tasks, and define projection policy.

- `deterministic core`
  Applies authoritative state transitions. It owns time, map, relationship, inventory, and other ruleset-backed simulation changes.

- `agent supervisor`
  Runs agent tasks as advisory, narrative, or interpretive work. Agents only emit artifacts; they do not directly mutate canonical runtime state.

- `projection`
  Builds player-visible text, operator-facing summaries, and later Godot-facing structured scene payloads from canonical state plus approved artifacts.

- `delivery`
  Owns outbound retryable transport jobs. Delivery is recoverable but not authoritative for canonical state.

- `repository`
  Stores runtime instances, versioned snapshots, execution records, events, tasks, artifacts, projections, and delivery jobs.

- `requirement registry`
  Stores validated rulesets, prompt packs, constraints, and later gameplay requirement assets as versioned data rather than informal notes.

## Stable Constraints

The redesign must preserve these hard constraints:

- Go runtime remains the orchestration authority
- deterministic simulation remains outside the LLM
- LLM output is assistive, not canonical truth
- Godot is a future client, not a state authority
- C++ remains a later optimization boundary, not the current runtime center
- recovery and normal execution must reuse the same stage model

## Flexible/Experimental Area

These areas should remain easy to change:

- mode heuristics and routing policy
- prompt pack composition
- agent task breakdown
- operational memory artifact shapes
- player-visible projection formatting
- Godot projection payload details before those requirements stabilize

## Impact

Affected subsystems:

- runtime core
- transport adapters
- LLM orchestration
- persistence and recovery
- testing strategy
- requirement management
- future Godot synchronization

## Expected Canonical Updates

- `docs/ARCHITECTURE.md`
- `docs/FEATURE_INHERITANCE.md`
- `docs/SYSTEM_TARGET_ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `docs/SUBSYSTEM_BOUNDARIES.md`
- `docs/INTERFACES.md`
- `docs/STORAGE_SCHEMA.md`
- `docs/TESTING.md`

## Open Questions

- how much of free chat should remain stateless versus runtime-instance-backed
- whether consult mode should share the same runtime instance model as roleplay modes
- which projections should be persisted by default versus regenerated on demand
- which requirement assets must be runtime-loaded versus build-time compiled

## Acceptance Signal

This redesign should be considered accepted when:

- canonical docs consistently describe the kernel/mode/task model
- implementation planning no longer treats the current early package split as fixed
- storage and testing expectations are aligned with recoverable executions
- future code work can proceed from this design without relying on legacy or scaffold assumptions
