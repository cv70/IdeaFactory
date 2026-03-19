# Idea Factory — Refactor & Runtime Semantics Design (v3)

**Date:** 2026-03-15
**Status:** Partially superseded on 2026-03-19
**Scope:** Historical refactor record for the early exploration runtime

---

## Superseded Notice

This document captured an earlier refactor stage where the backend was being reshaped around:

- `Planner` as the runtime seam
- deterministic graph generation as the baseline behavior
- explicit `ExecutionPlan` / `PlanStep` state as the execution backbone
- synchronous initial graph seeding during workspace creation

Parts of that structural cleanup are still historically useful, but its runtime semantics are no longer the target state.

The current target-state design has moved to:

- `MainAgent`-driven graph growth
- append-style graph tools as the write path
- thin backend runtime responsibilities
- no backend-owned phase strategy for deciding which nodes or edges to grow next

## What Remains Useful from This Document

The following themes still align with the current direction:

- splitting oversized handler/runtime files into clearer boundaries
- keeping runtime state cohesive per workspace
- preserving mutation broadcasting as a first-class runtime behavior
- keeping graph, projection, and runtime state explicitly modeled

These structural ideas may still inform implementation work when they do not reintroduce planner-centric graph policy.

## What Is No Longer the Target State

The following assumptions in the original document should be considered obsolete:

- backend-side `Planner` is the primary runtime decision seam
- `DeterministicPlanner` owns graph growth semantics
- `ExecutionPlan` and `PlanStep` are required to drive each round of graph growth
- workspace creation must synchronously seed initial direction nodes
- program-side rules determine which node type is generated next
- balance changes are translated into backend graph-generation branches

Under the current direction, these responsibilities move to `MainAgent`, with the backend remaining responsible for orchestration and graph write safety only.

## Current Source of Truth

Use these documents instead for any new implementation work:

- [idea-factory-technical-design.md](./idea-factory-technical-design.md)
- [idea-factory-system-architecture.md](./idea-factory-system-architecture.md)

The currently intended shape is:

- `MainAgent` reads workspace / graph / recent mutations
- `MainAgent` decides graph expansion and optimization
- graph writes go through append-style tool contracts such as `append_graph_batch`
- the backend performs minimal validation, persistence, event generation, and recovery

## Historical Role

This file should now be read only as a record of the codebase's intermediate refactor direction.

It is not the normative spec for the current runtime model.
