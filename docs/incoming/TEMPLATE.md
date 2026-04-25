# Incoming Requirement Template

Use this template for new design or requirement documents dropped into `docs/incoming/`.

```md
# <Title>

## Metadata

- status: incoming
- date: YYYY-MM-DD
- owner:
- source:
- scope:
  - runtime
  - gateway
  - control-plane
  - llm
  - store-recovery
  - godot
  - cplusplus
  - requirement-system
- priority:
  - low
  - medium
  - high
  - critical
- related_docs:
  - ...
- sync_targets:
  - docs/FEATURE_INHERITANCE.md
  - docs/SYSTEM_TARGET_ARCHITECTURE.md
  - docs/ROADMAP.md
  - docs/SUBSYSTEM_BOUNDARIES.md
  - docs/INTERFACES.md

## Summary

One short paragraph describing the change.

## Problem

What issue, gap, or opportunity does this document address?

## Proposal

Describe the new design, requirement, or change.

## Stable Constraints

Which parts must stay aligned with Babel/C++, world rules, or other hard constraints?

## Flexible/Experimental Area

Which parts are still intended to change quickly and should remain easy to iterate?

## Impact

Describe which subsystems are affected.

## Expected Canonical Updates

List which canonical docs should likely be updated if this proposal is accepted.

## Open Questions

List unresolved questions.

## Acceptance Signal

Describe what would make this proposal accepted, rejected, or turned into a staged experiment.
```
