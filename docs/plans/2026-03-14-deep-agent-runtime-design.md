# Idea Factory Deep-Agent Runtime Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align product and backend evolution around a `deep-agent orchestration runtime` with a light frontend and a strong self-balancing internal kernel.

**Architecture:** Keep the product contract centered on `workspace`, `方向地图`, `intervention`, and `artifact`, but make the implementation model explicit: a `MainAgent` generates and revises internal plans, dispatches tasks to isolated `SubAgents`, and writes durable graph/projection state while an internal balance engine continuously adjusts exploration rhythm. The existing `backend/domain/exploration` prototype is treated as the migration baseline, not discarded.

**Tech Stack:** Go, Gin, GORM, Eino ADK / Deep, WebSocket streaming, Markdown specs

---

## Summary

The current docs should now align around the same runtime shape:

- Product docs remain user-facing and keep the map as the primary default interface.
- Technical docs treat `MainAgent + SubAgents + TaskTool + wrapped tools + workdir/context + balance engine` as the default execution kernel.
- Backend implementation should migrate from the current in-memory auto-expansion prototype toward an explicit internal `run -> plan -> task -> result -> graph/projection` pipeline.

## Chosen Approach

### Product-facing approach

- Preserve the existing user mental model: users govern a `workspace`, not a multi-agent framework.
- Keep the frontend light: users mostly see the map and minimal state summaries.
- Keep `run` as a secondary explanatory object rather than a primary navigation object.

### Backend-facing approach

- Reuse the proven shape from `backend/deep` rather than inventing a new runtime model.
- Introduce durable models for `ExecutionPlan`, `PlanStep`, `AgentTask`, `BalanceState`, and task result summaries.
- Convert `intervention` from a direct mutation trigger into a replanning trigger.
- Treat graph and projection as normalized outputs of agent execution, not as direct side effects of ad-hoc helper functions.
- Replace hard elimination with suppression, folding, and later reactivation of weak directions.

### Migration approach

- Evolve the existing `backend/domain/exploration` package in place.
- Keep current HTTP and WebSocket entry points working during migration.
- Add tests around plan state and replanning before replacing the current timer-driven expansion loop.

## Product Doc Sync

The product doc should make these behaviors explicit without leaking implementation jargon:

- The system maintains an internal plan and balance loop rather than acting as an opaque loop.
- Frontend status should stay minimal and explain what changed recently and why.
- `intervention` feedback should explain how future exploration focus has shifted.

## Backend Target Shape

The exploration backend should converge toward:

- `workspace` as product contract
- `run` as execution instance
- `execution_plan` / `plan_step` as explicit control state
- `agent_task` as delegated sub-agent work
- `balance_state` as internal rhythm control
- `agent_task_result` as normalized structured output
- graph/projection persistence as the durable result layer
- websocket / streaming as the delivery path for projection and lightweight state updates

## Acceptance Criteria

- Docs no longer conflict on what “runtime” means.
- Product docs describe lightweight, map-first interaction and internal self-balancing behavior.
- A migration plan exists that maps current `backend/domain/exploration` files to the target deep-agent runtime.
