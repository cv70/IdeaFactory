# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```bash
cd backend
go build ./...          # compile check
go test ./...           # run all tests
go test ./domain/exploration/...   # run a specific package
go run main.go          # start server on :8888
```
Config is loaded from `IDEA_FACTORY_CONFIG_PATH` env var (default `config/idea_factory_config.yaml`). The actual config file lives outside the repo — do not read or modify it.

### Frontend (TypeScript + React)
```bash
cd frontend
npm run dev             # dev server on :5173 (proxies /api → localhost:8888, including WebSocket)
npm run build           # tsc type-check + vite build
npm run lint            # eslint
npm test                # vitest (watch mode)
npx vitest run          # single-pass test run
npx vitest run src/lib/workbench.test.ts   # single file
```

## Architecture

### Overview
IdeaFactory is an **autonomous exploration OS**. Users provide a topic + goal + constraints; the backend runs an LLM-driven agent cycle that builds a growing **direction map** (a graph of opportunities, hypotheses, questions, ideas, evidence). The frontend renders this graph live via WebSocket.

### Backend (`backend/`)

**Entry point**: `main.go` registers two domains on a Gin router under `/api/v1`.

**`domain/exploration/`** — the core. Key responsibilities split across files:
- `domain.go` — `ExplorationDomain` struct; per-workspace in-memory state (`RuntimeWorkspaceState`) behind a mutex via `withWorkspaceState`
- `handler_workspace.go`, `handler_run.go`, `handler_intervention.go` — HTTP handlers for V1 API
- `runtime_agent.go` — orchestrates the run→plan→task→result cycle; calls `planner` to build `ExecutionPlan`/`PlanStep`, dispatches to `agents/`
- `planner_llm.go` — `LLMPlanner` wraps 4 agents (General, Research, Graph, Artifact)
- `mutations.go` — `MutationEvent` stream: every state change emits mutations persisted to DB and pushed to WebSocket subscribers
- `realtime.go` — WebSocket hub; broadcasts snapshot and mutation events to subscribed clients
- `persistence.go` — dual-path: JSON snapshot to `WorkspaceRuntimeState` (primary), plus projection tables (secondary). Note: `InterventionEvent` and `MutationLog` use `gorm.Model` (uint PK); domain string IDs are stored in `TargetID`/`Payload` fields respectively.
- `projection_builder.go` — converts internal state to the V1 `projection` response (nodes + edges map)
- `exploration.go` / `deterministic.go` — in-memory fallback workspace logic (used when no DB)

**`agents/`** — Eino ADK agents. `ExplorationAgent` (deep/resumable) wraps ResearchAgent, GraphAgent, ArtifactAgent, GeneralAgent as sub-agents. `NewExplorationAgent` is the main agent used for autonomous runs.

**`domain/idea/`** — older standalone idea-generation endpoints (`/api/v1/ideas/`); not integrated with the exploration runtime.

**`datasource/dbdao/`** — GORM models. All tables use embedded `gorm.Model`. AutoMigrate runs at startup.

**`infra/`** — wires DB (SQLite via GORM) and LLM model (Qwen/OpenAI-compatible via Eino).

### Frontend (`frontend/src/`)

**State lives in `App.tsx`**: `ExplorationSession` is the central state. `explorationApi.ts` handles all requests with a priority waterfall: V1 REST → WebSocket → legacy REST → in-memory mock.

**Key data flow**:
1. User submits topic → `createExploration()` → POST `/api/v1/workspaces` + POST `/api/v1/workspaces/:id/runs`
2. `subscribeExploration()` opens a WebSocket (`explorationSocket.ts`) to `/api/v1/exploration/ws`
3. Server pushes `snapshot` (full state) and `mutation` (incremental) events
4. `mutations.ts` `applyExplorationMutations()` patches session state in place
5. `buildWorkbenchView()` (in `workbench.ts`) derives the sidebar `WorkbenchView` from the session

**`GraphView.tsx`**: renders the graph using React Flow + D3-Force. D3 runs a physics simulation; on each tick it calls `setRfNodes` to update node positions. `SimNode` tracks D3 positions; `RFNode<RFNodeData, 'ideaNode'>` is the React Flow node. `FloatingEdge` computes edge paths between node circle boundaries.

**`types/exploration.ts`**: canonical frontend types (`Node`, `Edge`, `ExplorationSession`, `WorkbenchView`). Node types: `topic | opportunity | question | hypothesis | idea | direction | evidence | claim | artifact | ...`

**`lib/explorationApi.ts`**: normalizes V1 backend responses (`workspace` + `projection`) into `ExplorationSession` via `toPayloadFromV1`. The V1 API splits data across two endpoints: `GET /workspaces/:id` (metadata) and `GET /workspaces/:id/projection` (graph map).
