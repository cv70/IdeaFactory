# Idea Factory Development Guidelines

This document outlines the architectural principles and development practices for Idea Factory, based on insights from Claude Code, Codex, and Hermes Agent.

## 🏗️ Three-Layer Architecture

Idea Factory follows a three-layer architecture inspired by the reference systems:

### L1: Control Face (治理面) - Inspired by Claude Code
Provides explicit user controls for governing the autonomous system.

**Key Principles:**
- Explicit control surface for user intervention
- Mode switching and state visualization
- Explicit entry points for review, resume, artifact generation
- Predictable user-system interaction patterns

**Development Practices:**
- All high-impact user actions must be available through explicit control mechanisms
- Control actions should be structured (not free-form text) where possible
- Provide clear feedback on control action absorption and effects
- Maintain session history and enable recovery from checkpoints

### L2: Execution Protocol Layer (执行协议层) - Inspired by Codex
Defines hard boundaries for safe, traceable execution.

**Key Principles:**
- Clear `run` → `turn` → `checkpoint` protocol
- Tool usage governed by explicit risk policies
- All graph mutations must be explicitly recorded and traceable
- Workspace isolation and sandboxing for unsafe operations

**Development Practices:**
- Every tool execution must go through approval/risk policy checks
- Maintain immutable execution traces for all runs
- Implement proper sandboxing for code execution and file operations
- Ensure all state changes are reversible or recoverable

### L3: Long-Term Capability Layer (长期能力层) - Inspired by Hermes Agent
Enables persistent learning and capability accumulation.

**Key Principles:**
- Persistent workspace memory across runs
- User preference memory for steering behavior
- Skill system for reusable exploration patterns
- Cross-run context continuation

**Development Practices:**
- Implement persistent storage for workspace and user memories
- Create skill discovery and automatic loading mechanisms
- Support subagent delegation for complex tasks
- Enable cross-session learning without prompt pollution

## 📁 Project Structure & Module Organization

This repository is split into `frontend/` and `backend/`. Use `frontend/src/` for React UI code, `frontend/public/` for static assets, and keep tests next to the code they cover such as `frontend/src/components/WorkspaceHeader.test.tsx`. Backend application code lives under `backend/`, with domain logic in `backend/domain/`, persistence in `backend/datasource/`, tool/runtime integrations in `backend/agentools/` and `backend/sdk/`, and entrypoints in `backend/cmd/` or `backend/main.go`. Product and architecture references live in `docs/superpowers/specs/`.

## 🔧 Build, Test, and Development Commands

Run the frontend locally with `cd frontend && npm run dev`. Build the frontend with `cd frontend && npm run build`. Lint frontend code with `cd frontend && npm run lint`. Run frontend tests with `cd frontend && npm test`. Run backend tests with `cd backend && go test ./...`. Prefer these repo-local commands before introducing new scripts.

## 🎨 Coding Style & Naming Conventions

Frontend uses TypeScript, React, and ESLint. Follow the existing style: 2-space indentation, single quotes, semicolons omitted, and small typed props objects. Name React components and types in PascalCase, hooks and helpers in camelCase, and keep filenames aligned with exports, for example `GraphView.tsx` and `explorationApi.ts`. Backend code should remain `gofmt`-formatted, with exported Go identifiers in PascalCase and package-local helpers in camelCase.

## 🧪 Testing Guidelines

Frontend tests use Vitest with Testing Library and should live beside the implementation as `*.test.ts` or `*.test.tsx`. Backend tests use the standard Go test runner and should be named `*_test.go`. Add or update tests whenever behavior changes, especially around API handlers, runtime orchestration, and workbench mutations.

## 📝 Commit & Pull Request Guidelines

Recent history follows Conventional Commits, for example `feat(frontend): integrate workspace pause/resume patch API` and `feat: add PATCH /workspaces/:id pause/resume endpoint`. Keep commits scoped and descriptive. Pull requests should explain the behavior change, list verification commands run, link the relevant issue or design doc, and include screenshots or GIFs for visible frontend changes.

## 🏛️ Architecture & Docs

Before larger changes, read `README.md` and the relevant spec in `docs/superpowers/specs/`. Treat those docs as the source of truth for product intent, API shape, and system boundaries.

## 🔍 Layer-Specific Development Guidelines

### Control Face Development
When implementing control surface features:
1. Reference Claude Code's command system for explicit user controls
2. Ensure all control actions provide clear feedback on absorption and effects
3. Implement mode switching (explore/pause/resume/etc.) as explicit state transitions
4. Provide visualization of current system state and recent changes

### Execution Protocol Development
When implementing execution logic:
1. Follow Codex's task-turn-protocol model for traceable execution
2. Implement tool risk policies that govern all external actions
3. Ensure all graph mutations are immutable and traceable
4. Implement proper checkpointing and recovery mechanisms
5. sandbox all potentially unsafe operations

### Long-Term Capability Development
When implementing persistent capabilities:
1. Reference Hermes Agent's memory and skill systems
2. Implement persistent workspace memory that survives system restarts
3. Create skill discovery mechanisms for reusable exploration patterns
4. Enable user preference memory to steer system behavior over time
5. Support subagent delegation for complex parallel workloads

## 🔄 Implementation Approach

Development should follow a layered validation approach:
1. **Phase 1**: Establish control face and execution protocol basics
2. **Phase 2**: Validate long-term capability mechanisms
3. **Phase 3**: Open platform capabilities for extension

Each phase should maintain clear separation of concerns between layers while ensuring tight integration at the interfaces.