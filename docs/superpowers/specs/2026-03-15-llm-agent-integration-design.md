# LLM/Agent Integration Design

**Date:** 2026-03-15
**Status:** Superseded on 2026-03-19
**Scope:** `backend/domain/exploration/` + `backend/agents/`

---

## Superseded Notice

This document described a transitional design where graph growth remained governed by backend-side planner logic (`LLMPlanner` + deterministic fallback), while agents were only responsible for generating content for specific node types.

That is no longer the intended target state.

The current target-state design moves graph growth authority to `MainAgent` itself:

- `MainAgent` reads the current workspace and graph state directly
- `MainAgent` decides what to add to the graph and when to do it
- graph writes happen through a controlled append-style graph tool
- the backend no longer acts as a phase planner for graph growth
- deterministic planner fallback is not part of the intended target state

Program-side responsibility is reduced to:

- run scheduling and lifecycle management
- minimal structural validation
- atomic persistence
- mutation/event broadcasting
- pause/resume and restart recovery

## Why This Design Was Superseded

The older design still kept graph policy in code:

- node generation priority stayed in `GenerateNodesForCycle`
- graph topology was constrained by planner-side phase ordering
- agent errors could fall back to deterministic program-side graph growth
- workspace creation still depended on synchronous graph seeding

That conflicted with the current implementation direction: graph expansion and optimization should be decided by the Agent, not by backend planner branches.

## Current Source of Truth

Use these documents instead:

- [idea-factory-technical-design.md](./idea-factory-technical-design.md)
- [idea-factory-system-architecture.md](./idea-factory-system-architecture.md)

In particular, the current source of truth is:

- `run -> agent session -> graph mutation -> projection`
- `MainAgent` is the sole graph growth decision-maker
- graph mutation flows through append-style tool contracts such as `append_graph_batch`
- the backend is a thin runtime shell, not a graph planner

## Historical Value

This file is kept only as historical context for the intermediate step between:

1. fully deterministic backend graph generation, and
2. fully agent-driven graph growth.

It should not be used as the basis for new implementation work.
