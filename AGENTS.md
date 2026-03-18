# Repository Guidelines

## Project Structure & Module Organization
This repository is split into `frontend/` and `backend/`. Use `frontend/src/` for React UI code, `frontend/public/` for static assets, and keep tests next to the code they cover such as `frontend/src/components/WorkspaceHeader.test.tsx`. Backend application code lives under `backend/`, with domain logic in `backend/domain/`, persistence in `backend/datasource/`, tool/runtime integrations in `backend/agentools/` and `backend/sdk/`, and entrypoints in `backend/cmd/` or `backend/main.go`. Product and architecture references live in `docs/superpowers/specs/`.

## Build, Test, and Development Commands
Run the frontend locally with `cd frontend && npm run dev`. Build the frontend with `cd frontend && npm run build`. Lint frontend code with `cd frontend && npm run lint`. Run frontend tests with `cd frontend && npm test`. Run backend tests with `cd backend && go test ./...`. Prefer these repo-local commands before introducing new scripts.

## Coding Style & Naming Conventions
Frontend uses TypeScript, React, and ESLint. Follow the existing style: 2-space indentation, single quotes, semicolons omitted, and small typed props objects. Name React components and types in PascalCase, hooks and helpers in camelCase, and keep filenames aligned with exports, for example `GraphView.tsx` and `explorationApi.ts`. Backend code should remain `gofmt`-formatted, with exported Go identifiers in PascalCase and package-local helpers in camelCase.

## Testing Guidelines
Frontend tests use Vitest with Testing Library and should live beside the implementation as `*.test.ts` or `*.test.tsx`. Backend tests use the standard Go test runner and should be named `*_test.go`. Add or update tests whenever behavior changes, especially around API handlers, runtime orchestration, and workbench mutations.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commits, for example `feat(frontend): integrate workspace pause/resume patch API` and `feat: add PATCH /workspaces/:id pause/resume endpoint`. Keep commits scoped and descriptive. Pull requests should explain the behavior change, list verification commands run, link the relevant issue or design doc, and include screenshots or GIFs for visible frontend changes.

## Architecture & Docs
Before larger changes, read `README.md` and the relevant spec in `docs/superpowers/specs/`. Treat those docs as the source of truth for product intent, API shape, and system boundaries.
