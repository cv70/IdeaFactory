# Graph Edge Routing — Design Spec

**Date:** 2026-03-17
**Status:** Approved
**Scope:** `frontend/src/components/GraphView.tsx` only

---

## Problem

The current graph renders edges using React Flow's `smoothstep` type with handles fixed at `Position.Left` (target) and `Position.Right` (source). Because D3-force distributes nodes radially in all directions, edges must route through the fixed handle sides even when a more direct path exists — producing visible tangles and unnecessary curves.

Additionally, the D3 repulsion force is modest, allowing nodes to cluster at higher node counts and compounding the visual clutter.

---

## Goal

Reduce edge tangles and crossings by:
1. Replacing fixed-handle smoothstep edges with floating edges that always connect via the shortest path between node circle boundaries.
2. Modestly increasing D3 repulsion so nodes occupy more canvas space.

---

## Design

### 1. FloatingEdge Component

A new custom edge component registered as `edgeTypes.floating`.

**Algorithm:**
1. Retrieve source and target nodes via `useStore(s => s.nodeLookup.get(id))`.
2. Compute each node's circle centre:
   - `cx = internals.positionAbsolute.x + measured.width / 2`
   - `cy = internals.positionAbsolute.y + nodeRadius(data.type)`
3. Compute the unit vector from source centre → target centre.
4. Offset each centre outward by its circle radius along that vector to get the true boundary exit/entry points.
5. Render via `getBezierPath` with a small curvature offset for visual polish.
6. Guard against zero-distance edges (same-position nodes) with an early `null` return.

**Registration:**
- `edgeTypes = { floating: FloatingEdge }` passed to `<ReactFlow>`.
- `buildRFEdge` returns `{ ..., type: 'floating' }`.
- `defaultEdgeOptions.type` set to `'floating'`.

The existing `Handle` elements in `IdeaNode` are retained (invisible) for React Flow internal compatibility; `FloatingEdge` ignores the handle-derived `sourceX/Y` / `targetX/Y` props entirely.

### 2. D3 Force Parameter Changes

| Force | Parameter | Old | New | Rationale |
|-------|-----------|-----|-----|-----------|
| `forceLink` | `distance` | 130 | 160 | More space between linked nodes |
| `forceManyBody` | `strength` | `-(r × 18)` | `-(r × 28)` | Stronger repulsion → less clustering |
| `forceCollide` | `radius` | `r + 12` | `r + 18` | Prevents circle overlap |

All other simulation parameters unchanged (`center` strength 0.05, `alphaDecay` 0.02).

---

## Constraints

- No new files — all changes within `GraphView.tsx`.
- No changes to node visuals, colours, or CSS.
- No changes to edge colours or opacity.
- No changes to the overall layout algorithm (D3-force remains the sole layout engine).

---

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/GraphView.tsx` | Add `FloatingEdge`; update `buildRFEdge`, `defaultEdgeOptions`, `edgeTypes`; tune D3 force params |

---

## Out of Scope

- Hierarchical / dagre layout
- Per-edge-type visual differentiation (colour, dash pattern)
- Node layout restructuring
