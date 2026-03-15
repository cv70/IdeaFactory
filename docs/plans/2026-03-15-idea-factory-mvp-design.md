# Idea Factory MVP Design

## Context

This design translates the target-state specs in `docs/superpowers/specs/` into a deliverable MVP that can run end-to-end in the current repository. The repository already contains:

- a Go backend with an `exploration` domain and early target-state API handlers
- a React workbench prototype in `frontend/`
- target-state product, technical, system, and OpenAPI specifications

The MVP does not attempt to fully implement the target-state autonomous runtime. It implements the smallest coherent slice that proves the product loop.

## Goal

Deliver a runnable v1 MVP that supports:

- creating a workspace
- starting a run
- building and reading a direction map projection
- submitting an intervention
- observing run, projection, and intervention lifecycle changes in the frontend

The system must work without an LLM, while leaving clear insertion points for future agent-driven planning and research.

## Scope

### In Scope

- Target-state v1 HTTP endpoints for workspace, run, projection, intervention, and trace summary/events
- A deterministic runtime that generates plans, tasks, graph mutations, and projections
- An in-memory persistence model for MVP domain state
- Frontend integration against the v1 API
- A map-first workbench experience showing direction branches, recent changes, focus, and intervention feedback
- Automated tests for core backend runtime logic and frontend API/view-model adaptation

### Out of Scope

- Full asynchronous job execution infrastructure
- Full database-backed consistency implementation
- Rich graph editing
- Real-time collaboration
- Complete artifact generation workflow
- Production-grade LLM orchestration across all task types

## Architecture

### Runtime Strategy

The MVP runtime uses a deterministic primary path with optional LLM enhancement:

- deterministic planner and executor are the default and required path
- planner and research steps expose adapter points for future LLM-backed behavior
- API contracts and domain state do not change based on whether an LLM is enabled

This keeps the MVP testable and locally runnable while preserving the target-state structure.

### Backend Modules

The backend remains inside `backend/domain/exploration` and is refactored around these responsibilities:

- `workspace service`: workspace CRUD and metadata state
- `run coordinator`: create runs, prevent duplicate concurrent runs, drive status transitions
- `planner`: create plan versions and ordered steps from workspace state, graph state, and intervention signals
- `task executor`: execute minimal task kinds such as direction discovery, evidence generation, and projection synthesis
- `integration/projection builder`: convert task results into graph mutations, projection snapshots, and trace events
- `intervention processor`: persist intervention lifecycle and trigger replanning

### Frontend Modules

The frontend keeps the existing workbench structure but changes the main data source to the v1 backend:

- launch flow creates a workspace and triggers the initial run
- workbench reads `workspace + projection` as the source of truth
- existing exploration-oriented components are adapted to direction-map concepts
- intervention submission and status display become first-class interactions

## Domain Model

### MVP State Coverage

The MVP implements a subset of the target-state statuses:

- `Workspace`: `draft -> active -> archived`
- `Run`: `queued -> planning -> dispatching -> integrating -> projected -> completed | failed`
- `ExecutionPlan`: `draft -> active -> completed | superseded | failed`
- `PlanStep`: `todo -> doing -> done | invalidated`
- `AgentTask`: `queued -> running -> succeeded | failed`
- `Intervention`: `received -> absorbed -> replanned -> reflected`

### Graph Coverage

The MVP graph supports the essential ontology needed for the map-first UI:

- nodes: `Direction`, `Evidence`, `Unknown`, `Decision`
- edges: `supports`, `contradicts`, `branches_from`, `raises`, `justifies`

`Artifact` nodes and richer relationships stay out of the first cut.

## Runtime Flow

### Initial Run

1. Create workspace.
2. Start run.
3. Coordinator creates run and active plan.
4. Planner generates 2-3 initial direction branches from topic, goal, and constraints.
5. Executor produces structured evidence, unknowns, and decisions for those branches.
6. Integration writes mutation events and builds projection.
7. Run advances to `projected` and `completed`.

### Intervention Replanning

1. User submits an intervention.
2. Intervention is persisted as `received`.
3. Runtime marks it `absorbed`.
4. Current plan is superseded or remaining steps invalidated as needed.
5. Planner emits a new plan version with a changed focus.
6. Projection updates recent changes and intervention effects.
7. Intervention advances through `replanned` to `reflected`.

## Deterministic Planning Rules

The deterministic planner follows a small rule set:

- on the first run, synthesize 2-3 candidate directions from workspace topic and goal
- assign initial maturity as `emerging`
- generate evidence summaries and open questions for each direction
- choose a focus branch using simple heuristics based on topic keywords, constraints, and prior intervention bias
- on later runs, deepen promising directions and fold low-signal branches only when explicitly suppressed
- when an intervention is present, bias step generation toward the requested direction change and mark invalidated steps

## API Contract Strategy

The target-state v1 routes in `backend/domain/exploration/routes.go` become the primary supported contract for the frontend MVP. Legacy exploration routes may remain temporarily for compatibility, but the frontend main path uses:

- `POST /api/v1/workspaces`
- `GET /api/v1/workspaces/:workspaceID`
- `POST /api/v1/workspaces/:workspaceID/runs`
- `GET /api/v1/workspaces/:workspaceID/runs/:runID`
- `GET /api/v1/workspaces/:workspaceID/projection`
- `POST /api/v1/workspaces/:workspaceID/interventions`
- `GET /api/v1/workspaces/:workspaceID/interventions/:interventionID`
- `GET /api/v1/workspaces/:workspaceID/trace/summary`
- `GET /api/v1/workspaces/:workspaceID/trace/events`

For MVP, run execution may complete within the request lifecycle while still returning the documented `202 Accepted` semantics. This preserves the contract while avoiding job-system complexity.

## Frontend Experience

The MVP frontend has four visible surfaces:

- launch page for workspace creation
- map-first main view for directions and branch structure
- sidebar summary for focus, recent changes, and supporting evidence
- intervention panel showing submission status and resulting directional shift

The UI should stop treating the main object as a transient exploration session and instead present a long-lived workspace with a living direction map.

## Error Handling

- invalid workspace or run IDs return `404`
- invalid payloads return `400`
- runtime failures mark the run as `failed` and preserve traceability
- projection failures do not destroy prior graph state
- frontend shows degraded but readable status when projection or run fetches fail

## Testing

### Backend

- unit tests for deterministic planner behavior
- unit tests for run status transitions
- unit tests for intervention lifecycle progression
- API tests for v1 workspace, run, projection, and intervention endpoints

### Frontend

- tests for v1 API adaptation in `explorationApi`
- tests for projection-to-workbench mapping
- component tests for workspace launch and intervention-driven refresh states

## Delivery Strategy

Implementation should proceed as a vertical slice:

1. solidify backend domain state and deterministic runtime
2. make v1 API responses reliable and test-covered
3. adapt frontend data access to v1 APIs
4. wire the workbench to real workspace and projection state
5. add intervention lifecycle display and end-to-end verification

## Risks

- current backend exploration code may mix older prototype concepts with target-state concepts, so refactoring boundaries need care
- frontend naming and types are still centered on opportunity/idea exploration and may cause adaptation friction
- if optional LLM hooks leak into required execution paths, local determinism and testability will regress

## Decision

The MVP will be implemented as a deterministic end-to-end system with optional LLM enhancement points, using the existing backend and frontend foundations in this repository.
