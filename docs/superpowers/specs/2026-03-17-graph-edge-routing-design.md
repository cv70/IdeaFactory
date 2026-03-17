# Graph Edge Routing — Design Spec

**Date:** 2026-03-17
**Status:** Approved
**Scope:** `frontend/src/components/GraphView.tsx` only
**React Flow version:** `@xyflow/react ^12.5.0`

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

A new custom edge component registered as `edgeTypes.floating`. It ignores the handle-derived `sourceX/Y` / `targetX/Y` props and computes its own endpoints.

**Node geometry note:** `IdeaNode` is a `flex-direction: column` container with `align-items: center`. The label has `max-width: 120px` and can be wider than the circle, so `measured.width / 2` and `nodeRadius(data.type)` are *not* always equal.
- The circle occupies the top of the node; its top edge aligns with the node's top edge.
- **circle centre x** = `positionAbsolute.x + measured.width / 2` — node bounding box is centred on the circle horizontally due to `align-items: center`, so this gives the circle centre regardless of label width.
- **circle centre y** = `positionAbsolute.y + nodeRadius(data.type)` — the circle sits flush at the top, so one radius down from the top edge reaches the circle centre.

**Algorithm:**
1. Call `useInternalNode(source)` and `useInternalNode(target)` — where `source` and `target` are the custom edge component's node-id props (not the edge `id` prop). Returns `InternalNode` with `internals.positionAbsolute` and `measured.width`.
2. Guard: if either node is missing, or if `measured.width` is not yet populated (first render frame, before React Flow has measured the node), return `null`. (`cy` uses `nodeRadius(data.type)` — a type-derived constant — so `measured.height` is not needed and is not checked.)
3. Compute each node's circle centre using the geometry above.
4. Compute `dx = targetCx - sourceCx`, `dy = targetCy - sourceCy`, `dist = √(dx²+dy²)`. Guard: if `dist === 0`, return `null`.
5. Compute unit vector `ux = dx/dist`, `uy = dy/dist`.
6. Source boundary point: `(sourceCx + ux × sourceRadius, sourceCy + uy × sourceRadius)` — moves source centre *toward* target.
7. Target boundary point: `(targetCx - ux × targetRadius, targetCy - uy × targetRadius)` — moves target centre *toward* source (opposite direction).
8. Call `getBezierPath({ sourceX: <step 6 x>, sourceY: <step 6 y>, targetX: <step 7 x>, targetY: <step 7 y>, sourcePosition: Position.Right, targetPosition: Position.Left, curvature: 0.15 })`. The boundary-point values from steps 6–7 replace the handle-derived props; `Position.Right/Left` prevents unexpected control-point bowing on near-vertical paths (React Flow defaults to `Position.Bottom` when omitted).

**Registration:**
- `edgeTypes = { floating: FloatingEdge }` passed to `<ReactFlow>`.
- `buildRFEdge` returns `{ ..., type: 'floating' }` (authoritative).
- `defaultEdgeOptions.type` also set to `'floating'` as a belt-and-suspenders fallback.

The existing invisible `Handle` elements in `IdeaNode` are retained for React Flow internal compatibility.

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
