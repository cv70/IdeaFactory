# Graph View Design
Date: 2026-03-16
Status: Approved

## Problem

The current `WorkbenchColumns` component shows nodes and edges as static lists split across three columns. Users cannot see the actual graph structure — how nodes connect, which branches are expanding, and how the space is growing. The goal is to replace this with a living, force-directed graph visualization that grows in real time as the agent produces new nodes.

## Solution

Replace `WorkbenchColumns` with a new `GraphView` component powered by D3 force simulation. The graph renders in SVG inside a resizable container. D3 manages node positions; React manages the SVG elements. The sidebar, launch panel, and workspace manager are unchanged.

## Architecture

### Components

**`GraphView.tsx`** (new)
- Props: `session: ExplorationSession`, `onSelectNode: (node: Node | null) => void`, `selectedNodeId: string | null`
- Owns: D3 force simulation (via `useRef`), node position state, zoom/pan transform state, hovered/selected node state
- Renders: SVG with a zoom group containing edge lines, node circles, and node labels; a floating `NodeDetailCard` overlay when a node is selected

**`NodeDetailCard`** (inline sub-component inside GraphView)
- Shows selected node type badge, title, summary, and context-sensitive action buttons (e.g. "Expand branch" for direction nodes)

**`App.tsx`** (modified)
- Remove `WorkbenchColumns` import and usage
- Add `selectedNodeId` state (replaces `selectedOpportunityId`)
- Pass `onSelectNode` callback to `GraphView`; callback maps selected node to the existing `handleSelectOpportunity` / `handleExpandOpportunity` logic

### D3 Simulation Setup

```
forceSimulation
  └── forceLink   (distance: 120, strength: 0.3) — edge attraction
  └── forceManyBody (strength: -280) — node repulsion
  └── forceCenter  (cx, cy, strength: 0.08) — pulls graph toward SVG center
  └── forceCollide  (radius: nodeRadius + 10) — prevents overlap
  alphaDecay: 0.025
```

The topic node has `fx = cx, fy = cy` to pin it to center. All other nodes float freely.

### Data Preparation

1. If no `type === 'topic'` node exists in `session.nodes`, create a synthetic one with `id = '__topic__' + session.id`, `title = session.topic`.
2. For each `direction` or `opportunity` node that has no incoming edge, synthesize an edge from the topic node to it.
3. Pass all nodes and edges to the simulation.

### Node-to-Simulation Sync

On each render triggered by changes to `session.nodes` / `session.edges`:
- New nodes are inserted into the simulation's node map with initial position near their parent node (or near center if no parent).
- Existing nodes have their data updated but positions preserved (D3 mutates x/y in place).
- Removed nodes are deleted from the map.
- `simulation.alpha(0.4).restart()` is called only when new nodes are added.

### Zoom & Pan

- D3 zoom behavior attached to the SVG element via `useEffect`.
- On zoom event, React state `transform: {x, y, k}` is updated.
- The main `<g>` group applies `translate(x,y) scale(k)`.

### Node Dragging

- D3 drag behavior attached to each node circle.
- On drag start: `node.fx = node.x; node.fy = node.y` (fix position).
- On drag: update `fx`, `fy`.
- On drag end: release fix (`fx = null, fy = null`), let simulation resume.

### Visual Encoding

| Node type | Fill | Stroke | Radius |
|-----------|------|--------|--------|
| topic | `#0f1f4a` | `#2851b0` | 40 |
| direction | `#c56700` | `#f59e0b` | 28 |
| opportunity | `#b45309` | `#f59e0b` | 26 |
| question | `#1e40af` | `#60a5fa` | 20 |
| hypothesis | `#5b21b6` | `#a78bfa` | 20 |
| idea | `#047857` | `#34d399` | 22 |
| evidence | `#4b5563` | `#9ca3af` | 14 |
| artifact | `#b45309` | `#fbbf24` | 20 |
| claim | `#0e7490` | `#22d3ee` | 18 |
| tension | `#9f1239` | `#fb7185` | 18 |
| decision | `#166534` | `#4ade80` | 18 |
| unknown | `#6b7280` | `#d1d5db` | 16 |

Selected node: white ring `stroke: #fff, stroke-width: 3, filter: drop-shadow`.
Hovered node: ring `stroke: rgba(255,255,255,0.6)`.

Edge lines: `stroke: rgba(23,42,92,0.15)`, `stroke-width: 1.5`. Selected node's edges highlighted `rgba(40,81,176,0.5)`.

### Entry Animation

New nodes and their labels get CSS class `.node-enter` with `@keyframes nodeEnter { from { opacity:0; transform: scale(0) } to { opacity:1; transform: scale(1) } }`, duration 400ms ease-out. Applied via a `Set<string>` of recently-added node IDs cleared after the animation duration.

### Layout Container

`GraphView` sits inside a `<div className="graphContainer">` that is `100%` wide and `min-height: 520px` (responsive). A `ResizeObserver` keeps the center force updated as the container resizes.

## Node Detail Card

Shown as an absolutely-positioned card in the bottom-left corner of the graph container when a node is selected:
- Type badge (colored pill matching node color)
- Title (h3)
- Summary (p, max 3 lines)
- For `direction`/`opportunity` nodes: "Expand branch" button → calls `onExpandOpportunity`
- "×" dismiss button

## Dependencies

Add to `frontend/package.json`:
- `d3` (v7, includes force/zoom/drag/selection)
- `@types/d3` (devDependency)

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/GraphView.tsx` | New — full graph component |
| `frontend/src/App.tsx` | Replace WorkbenchColumns with GraphView |
| `frontend/src/App.css` | Add graph container and node-enter animation styles |
| `frontend/src/lib/i18n.ts` | Add graph i18n keys |
| `frontend/package.json` | Add d3 dependency |

## Out of Scope

- Server-side layout computation
- Graph export (PNG/SVG)
- Edge labels
- Node editing in the graph
