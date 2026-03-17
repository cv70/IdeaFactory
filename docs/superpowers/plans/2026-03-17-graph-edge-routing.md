# Graph Edge Routing Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace fixed-handle smoothstep edges with floating circle-boundary edges, and tune D3 forces to spread nodes more.

**Architecture:** All changes are in one file (`GraphView.tsx`). A pure `circleBoundaryPoints` helper is extracted for testability. `FloatingEdge` is a custom React Flow edge component that calls the helper to compute its own endpoint positions, ignoring the handle-derived props. D3 force parameters are updated in the simulation init effect.

**Tech Stack:** React, TypeScript, `@xyflow/react ^12.5.0`, `d3-force`, Vitest

**Spec:** `docs/superpowers/specs/2026-03-17-graph-edge-routing-design.md`

---

## Chunk 1: `buildRFEdge` type field + `circleBoundaryPoints` utility

### Task 1: Update `buildRFEdge` to return `type: 'floating'`

**Files:**
- Modify: `frontend/src/components/GraphView.tsx` (function `buildRFEdge`, line ~75)
- Test: `frontend/src/components/GraphView.test.ts`

- [ ] **Step 1: Write the failing test**

In `GraphView.test.ts`, inside the existing `describe('buildRFEdge', ...)` block, add an assertion for the `type` field:

```ts
describe('buildRFEdge', () => {
  it('builds a React Flow edge from ExplorationEdge', () => {
    const edge: ExplorationEdge = { id: 'e1', from: 'n1', to: 'n2', type: 'leads_to' }
    const rfEdge = buildRFEdge(edge)
    expect(rfEdge.id).toBe('e1')
    expect(rfEdge.source).toBe('n1')
    expect(rfEdge.target).toBe('n2')
    expect(rfEdge.type).toBe('floating')   // ← new assertion
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: FAIL — `expected undefined to be 'floating'`

- [ ] **Step 3: Add `type: 'floating'` to `buildRFEdge`**

In `GraphView.tsx`, find:

```ts
export function buildRFEdge(edge: ExplorationEdge): RFEdge {
  return { id: edge.id, source: edge.from, target: edge.to }
}
```

Replace with:

```ts
export function buildRFEdge(edge: ExplorationEdge): RFEdge {
  return { id: edge.id, source: edge.from, target: edge.to, type: 'floating' }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/GraphView.tsx frontend/src/components/GraphView.test.ts
git commit -m "feat: buildRFEdge returns type floating"
```

---

### Task 2: Export and test `circleBoundaryPoints` utility

**Files:**
- Modify: `frontend/src/components/GraphView.tsx` (add exported function after `nodeRadius`)
- Test: `frontend/src/components/GraphView.test.ts`

- [ ] **Step 1: Write failing tests**

Add a new `describe` block in `GraphView.test.ts` (import `circleBoundaryPoints` alongside the existing imports):

```ts
import { nodeConfig, nodeRadius, buildRFNode, buildRFEdge, circleBoundaryPoints } from './GraphView'
```

```ts
describe('circleBoundaryPoints', () => {
  it('returns null when source and target centers are the same', () => {
    expect(circleBoundaryPoints(0, 0, 10, 0, 0, 10)).toBeNull()
  })

  it('computes boundary points for a horizontal edge', () => {
    // source at (0,0) r=10, target at (100,0) r=10
    // unit vector: (1, 0)
    // source boundary: (10, 0), target boundary: (90, 0)
    const pts = circleBoundaryPoints(0, 0, 10, 100, 0, 10)
    expect(pts).not.toBeNull()
    expect(pts!.sx).toBeCloseTo(10)
    expect(pts!.sy).toBeCloseTo(0)
    expect(pts!.tx).toBeCloseTo(90)
    expect(pts!.ty).toBeCloseTo(0)
  })

  it('computes boundary points for a vertical edge', () => {
    // source at (0,0) r=10, target at (0,100) r=10
    // unit vector: (0, 1)
    // source boundary: (0, 10), target boundary: (0, 90)
    const pts = circleBoundaryPoints(0, 0, 10, 0, 100, 10)
    expect(pts).not.toBeNull()
    expect(pts!.sx).toBeCloseTo(0)
    expect(pts!.sy).toBeCloseTo(10)
    expect(pts!.tx).toBeCloseTo(0)
    expect(pts!.ty).toBeCloseTo(90)
  })

  it('computes boundary points for a diagonal edge', () => {
    // source at (0,0) r=10, target at (30,40) r=10
    // dist=50, ux=0.6, uy=0.8
    // source boundary: (6, 8), target boundary: (24, 32)
    const pts = circleBoundaryPoints(0, 0, 10, 30, 40, 10)
    expect(pts).not.toBeNull()
    expect(pts!.sx).toBeCloseTo(6)
    expect(pts!.sy).toBeCloseTo(8)
    expect(pts!.tx).toBeCloseTo(24)
    expect(pts!.ty).toBeCloseTo(32)
  })

  it('handles asymmetric radii correctly', () => {
    // source at (0,0) r=10, target at (100,0) r=20
    // unit vector: (1, 0)
    // source boundary: (10, 0), target boundary: (80, 0)
    const pts = circleBoundaryPoints(0, 0, 10, 100, 0, 20)
    expect(pts).not.toBeNull()
    expect(pts!.sx).toBeCloseTo(10)
    expect(pts!.sy).toBeCloseTo(0)
    expect(pts!.tx).toBeCloseTo(80)
    expect(pts!.ty).toBeCloseTo(0)
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: FAIL — `circleBoundaryPoints is not a function` (or similar import error)

- [ ] **Step 3: Implement `circleBoundaryPoints`**

In `GraphView.tsx`, add this exported function immediately after the `nodeRadius` function (in the "Visual encoding" section, before the RF node/edge builders):

```ts
type BoundaryPoints = { sx: number; sy: number; tx: number; ty: number }

export function circleBoundaryPoints(
  scx: number, scy: number, sr: number,
  tcx: number, tcy: number, tr: number,
): BoundaryPoints | null {
  const dx = tcx - scx
  const dy = tcy - scy
  const dist = Math.sqrt(dx * dx + dy * dy)
  if (dist === 0) return null
  const ux = dx / dist
  const uy = dy / dist
  return {
    sx: scx + ux * sr,
    sy: scy + uy * sr,
    tx: tcx - ux * tr,
    ty: tcy - uy * tr,
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/GraphView.tsx frontend/src/components/GraphView.test.ts
git commit -m "feat: add circleBoundaryPoints utility with tests"
```

---

## Chunk 2: FloatingEdge component + wiring + D3 tuning

### Task 3: Implement `FloatingEdge` and wire into `GraphCanvas`

**Files:**
- Modify: `frontend/src/components/GraphView.tsx`
  - Add imports from `@xyflow/react`
  - Add `FloatingEdge` component (after `IdeaNode`)
  - Add `edgeTypes` constant
  - Update `GraphCanvas` props and render

- [ ] **Step 1: Add new imports**

In `GraphView.tsx`, find the existing `@xyflow/react` import block:

```ts
import {
  ReactFlow, ReactFlowProvider,
  useNodesState, useEdgesState, useReactFlow,
  Background, Controls, MiniMap,
  Handle, Position,
  type NodeProps,
} from '@xyflow/react'
import type { Node as RFNode, Edge as RFEdge } from '@xyflow/react'
```

Replace with:

```ts
import {
  ReactFlow, ReactFlowProvider,
  useNodesState, useEdgesState, useReactFlow,
  useInternalNode,
  getBezierPath, BaseEdge,
  Background, Controls, MiniMap,
  Handle, Position,
  type NodeProps, type EdgeProps,
} from '@xyflow/react'
import type { Node as RFNode, Edge as RFEdge } from '@xyflow/react'
```

- [ ] **Step 2: Add `FloatingEdge` component**

In `GraphView.tsx`, after the closing brace of `IdeaNode` (around line 100) and before `const nodeTypes = ...`, add:

```ts
function FloatingEdge({ id, source, target, markerEnd, style }: EdgeProps) {
  const sourceNode = useInternalNode(source)
  const targetNode = useInternalNode(target)

  if (!sourceNode || !targetNode) return null
  if (!sourceNode.measured?.width || !targetNode.measured?.width) return null

  const sr = nodeRadius(sourceNode.data?.type as string)
  const tr = nodeRadius(targetNode.data?.type as string)
  const scx = sourceNode.internals.positionAbsolute.x + sourceNode.measured.width / 2
  const scy = sourceNode.internals.positionAbsolute.y + sr
  const tcx = targetNode.internals.positionAbsolute.x + targetNode.measured.width / 2
  const tcy = targetNode.internals.positionAbsolute.y + tr

  const pts = circleBoundaryPoints(scx, scy, sr, tcx, tcy, tr)
  if (!pts) return null

  const [edgePath] = getBezierPath({
    sourceX: pts.sx,
    sourceY: pts.sy,
    targetX: pts.tx,
    targetY: pts.ty,
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
    curvature: 0.15,
  })

  return <BaseEdge id={id} path={edgePath} markerEnd={markerEnd} style={style} />
}
```

- [ ] **Step 3: Register `edgeTypes`**

Find:

```ts
const nodeTypes = { ideaNode: IdeaNode }
```

Replace with:

```ts
const nodeTypes = { ideaNode: IdeaNode }
const edgeTypes = { floating: FloatingEdge }
```

- [ ] **Step 4: Update `GraphCanvas` to use `edgeTypes` and `defaultEdgeOptions`**

In `GraphCanvas`, find:

```ts
      <ReactFlow
        nodes={rfNodes}
        edges={rfEdges}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={{
          type: 'smoothstep',
          style: { stroke: 'rgba(23,42,92,0.15)', strokeWidth: 1.5 },
        }}
```

Replace with:

```ts
      <ReactFlow
        nodes={rfNodes}
        edges={rfEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        defaultEdgeOptions={{
          type: 'floating',
          style: { stroke: 'rgba(23,42,92,0.15)', strokeWidth: 1.5 },
        }}
```

- [ ] **Step 5: Run tests**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: all tests PASS (no new tests needed — `FloatingEdge` is a React component that requires a render environment; its logic is covered by `circleBoundaryPoints` tests)

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/GraphView.tsx
git commit -m "feat: add FloatingEdge component, wire into GraphCanvas"
```

---

### Task 4: Tune D3 force parameters

**Files:**
- Modify: `frontend/src/components/GraphView.tsx` (D3 simulation init effect, ~line 286)

- [ ] **Step 1: Update the three force parameters**

In `GraphView.tsx`, find the D3 simulation init block:

```ts
    const sim = d3
      .forceSimulation<SimNode>()
      .force(
        'link',
        d3.forceLink<SimNode, SimLink>().id((d) => d.id).distance(130).strength(0.4),
      )
      .force('charge', d3.forceManyBody<SimNode>().strength((d) => -(nodeRadius(d.type as string) * 18)))
      .force('center', d3.forceCenter(0, 0).strength(0.05))
      .force('collide', d3.forceCollide<SimNode>((d) => nodeRadius(d.type as string) + 12).strength(0.8))
```

Replace with:

```ts
    const sim = d3
      .forceSimulation<SimNode>()
      .force(
        'link',
        d3.forceLink<SimNode, SimLink>().id((d) => d.id).distance(160).strength(0.4),
      )
      .force('charge', d3.forceManyBody<SimNode>().strength((d) => -(nodeRadius(d.type as string) * 28)))
      .force('center', d3.forceCenter(0, 0).strength(0.05))
      .force('collide', d3.forceCollide<SimNode>((d) => nodeRadius(d.type as string) + 18).strength(0.8))
```

Changes: `distance(130 → 160)`, `* 18 → * 28`, `+ 12 → + 18`.

- [ ] **Step 2: Run tests**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts
```

Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/GraphView.tsx
git commit -m "feat: tune D3 forces — more repulsion and spacing"
```
