# Graph View Design
Date: 2026-03-16
Status: Approved

## Problem

The current `WorkbenchColumns` shows nodes/edges as static lists. Replace it with a living, force-directed graph that grows in real time as the agent produces new nodes.

## Solution: React Flow (rendering) + D3-Force (layout calculation)

**React Flow** (`@xyflow/react`) owns: SVG/DOM rendering, zoom, pan, drag, node/edge components, selection events.
**D3-Force** (`d3-force`) owns: position calculation via physics simulation — repulsion, link springs, collision, centering.

This combo avoids every problem from the previous Dagre-based spec:
- D3-Force produces smooth organic movement inherently — no "Pro" position-animation feature needed
- No coordinate-system offset mismatch (D3-Force gives pixel positions directly)
- No dagre `sourcePosition`/`targetPosition` directionality issues
- Node drag: use React Flow's `onNodeDrag*` callbacks to pin/release `fx`/`fy` in the D3 simulation

---

## Component Structure

### Provider split (required by React Flow)

React Flow's `useReactFlow()` hook must be called inside a child of `<ReactFlow>`. Split into two components:

```
GraphView            ← owns D3 simulation, derives allNodes/allEdges, renders <ReactFlowProvider>
  └── GraphCanvas    ← inside <ReactFlow>, calls useReactFlow() for fitView, renders NodeDetailCard
```

### `GraphView.tsx` (new file)
Props:
```typescript
import type { Node as ExplorationNode } from '../types/exploration'  // aliased to avoid collision with RF Node
import type { ExplorationSession } from '../types/exploration'

type GraphViewProps = {
  session: ExplorationSession
  selectedNodeId: string | null
  onSelectNode: (node: ExplorationNode | null) => void
  onExpandOpportunity: (node: ExplorationNode) => void
}
```

Owns:
- `simRef: useRef<d3.Simulation<SimNode, SimLink>>` — D3 simulation instance, created once at mount
- `simNodeMap: useRef<Map<string, SimNode>>` — stable map of id → SimNode (preserves D3 positions across re-renders)
- `draggingId: useRef<string | null>` — node being dragged (for fx/fy pinning)
- `[rfNodes, setRfNodes, onNodesChange]: useNodesState<RFNodeData>` — React Flow nodes (third element required for drag/measure callbacks)
- `[rfEdges, setRfEdges, onEdgesChange]: useEdgesState` — React Flow edges

### Type aliases (top of GraphView.tsx)

```typescript
import type { Node as RFNode, Edge as RFEdge } from '@xyflow/react'
import type { Node as ExplorationNode, Edge as ExplorationEdge } from '../types/exploration'
```

Use `ExplorationNode`/`ExplorationEdge` throughout for domain types, `RFNode`/`RFEdge` for React Flow types.

---

## D3-Force Simulation

### SimNode and SimLink types

```typescript
type SimNode = d3.SimulationNodeDatum & {
  id: string
  type: string           // NodeType from exploration.ts
  title: string
  [key: string]: unknown // remaining ExplorationNode fields
}

type SimLink = d3.SimulationLinkDatum<SimNode>
// d3-force mutates source/target from string → SimNode after linkForce.links() is called.
// Always pass fresh { source: string, target: string } objects on each update (never reuse).
```

### Initialization (once at mount)

```typescript
useEffect(() => {
  const sim = d3.forceSimulation<SimNode>()
    .force('link',    d3.forceLink<SimNode, SimLink>().id(d => d.id).distance(130).strength(0.4))
    .force('charge',  d3.forceManyBody<SimNode>().strength(d => -(nodeRadius(d.type) * 18)))
    .force('center',  d3.forceCenter(0, 0).strength(0.05))
    .force('collide', d3.forceCollide<SimNode>(d => nodeRadius(d.type) + 12).strength(0.8))
    .alphaDecay(0.02)
    .on('tick', onTick)
  simRef.current = sim
  return () => sim.stop()
}, [])
```

Center is at `(0, 0)` — React Flow's coordinate space has no required origin. Topic node is pinned: `node.fx = 0; node.fy = 0`.

### `onTick` callback

Runs on every simulation tick (~60fps while active, decaying over ~230 ticks / ~4 seconds):

```typescript
function onTick() {
  setRfNodes(prev => prev.map(rfNode => {
    const simNode = simNodeMap.current.get(rfNode.id)
    if (!simNode || simNode.x == null) return rfNode
    return { ...rfNode, position: { x: simNode.x, y: simNode.y } }
  }))
}
```

**Performance note:** This runs at up to 60fps for 3–4 seconds per new batch of nodes. Acceptable for graphs up to ~30 nodes. Beyond that, consider throttling with `requestAnimationFrame` gating or applying positions directly to DOM refs.

### Simulation sync (when session.nodes / session.edges change)

```typescript
useEffect(() => {
  const sim = simRef.current
  if (!sim) return

  const map = simNodeMap.current
  let hasNew = false

  // 1. Add new nodes, update existing
  for (const node of allNodes) {  // allNodes = [syntheticTopic?, ...session.nodes]
    if (!map.has(node.id)) {
      const parentEdge = allEdges.find(e => e.to === node.id)
      const parent = parentEdge ? map.get(parentEdge.from) : null
      map.set(node.id, {
        ...node,
        x: parent?.x != null ? parent.x + (Math.random() - 0.5) * 30 : (Math.random() - 0.5) * 60,
        y: parent?.y != null ? parent.y + (Math.random() - 0.5) * 30 : (Math.random() - 0.5) * 60,
      })
      hasNew = true
    } else {
      Object.assign(map.get(node.id)!, node)  // update data, preserve x/y/vx/vy
    }
  }

  // 2. Remove stale nodes
  const nodeIds = new Set(allNodes.map(n => n.id))
  for (const id of map.keys()) if (!nodeIds.has(id)) map.delete(id)

  // 3. Update RF nodes (add entering nodes as hidden, reveal atomically after layout settles)
  const enteringIds = new Set(allNodes.filter(n => !rfNodes.find(r => r.id === n.id)).map(n => n.id))
  setRfNodes(allNodes.map(n => buildRFNode(n, enteringIds.has(n.id))))
  setRfEdges(allEdges.map(buildRFEdge))

  // 4. Update D3 simulation
  sim.nodes([...map.values()])
  const linkForce = sim.force<d3.ForceLink<SimNode, SimLink>>('link')
  linkForce?.links(allEdges.filter(e => nodeIds.has(e.from) && nodeIds.has(e.to))
    .map(e => ({ source: e.from, target: e.to })))

  // 5. Restart simulation on new nodes
  if (hasNew) sim.alpha(0.5).restart()

  // 6. Reveal entering nodes — single effect cleanup prevents stale timers on unmount
  if (enteringIds.size > 0) {
    const t1 = setTimeout(() => {
      setRfNodes(prev => prev.map(n =>
        enteringIds.has(n.id) ? { ...n, hidden: false, className: 'nodeEnter' } : n
      ))
    }, 16)   // one frame — positions are committed but node not yet visible
    const t2 = setTimeout(() => {
      setRfNodes(prev => prev.map(n =>
        enteringIds.has(n.id) ? { ...n, className: '' } : n
      ))
    }, 520)  // 16 + 500 (animation duration)
    return () => { clearTimeout(t1); clearTimeout(t2) }
  }
}, [allNodes, allEdges])
```

### Node dragging (React Flow callbacks → D3)

```typescript
onNodeDragStart={(_, node) => {
  const simNode = simNodeMap.current.get(node.id)
  if (simNode) { simNode.fx = node.position.x; simNode.fy = node.position.y }
  simRef.current?.alphaTarget(0.3).restart()
}}
onNodeDrag={(_, node) => {
  const simNode = simNodeMap.current.get(node.id)
  if (simNode) { simNode.fx = node.position.x; simNode.fy = node.position.y }
}}
onNodeDragStop={(_, node) => {
  const simNode = simNodeMap.current.get(node.id)
  if (simNode) { simNode.fx = null; simNode.fy = null }
  simRef.current?.alphaTarget(0)
}}
```

---

## Data Preparation (`useMemo`)

```typescript
const { allNodes, allEdges } = useMemo(() => {
  // 1. Anchor: find real topic node or create synthetic
  const realTopic = session.nodes.find(n => n.type === 'topic')
  const anchorId = realTopic?.id ?? `__topic__${session.id}`
  const syntheticTopic: ExplorationNode | null = realTopic ? null : {
    id: anchorId, sessionId: session.id, type: 'topic',
    title: session.topic, summary: session.outputGoal,
    status: 'active', score: 1, depth: 0, metadata: {}, evidenceSummary: '',
  }

  // 2. Synthetic edges: connect unlinked direction/opportunity nodes to anchor
  const incomingIds = new Set(session.edges.map(e => e.to))
  const syntheticEdges: ExplorationEdge[] = session.nodes
    .filter(n => (n.type === 'direction' || n.type === 'opportunity') && !incomingIds.has(n.id))
    .map(n => ({ id: `__e__${anchorId}__${n.id}`, from: anchorId, to: n.id, type: 'leads_to' }))

  return {
    allNodes: syntheticTopic ? [syntheticTopic, ...session.nodes] : session.nodes,
    allEdges: [...session.edges, ...syntheticEdges],
  }
}, [session.nodes, session.edges, session.id, session.topic, session.outputGoal])
```

---

## React Flow Configuration

```tsx
<ReactFlow
  nodes={rfNodes}
  edges={rfEdges}
  nodeTypes={{ ideaNode: IdeaNode }}
  edgeTypes={{}}               // use default smoothstep edges
  defaultEdgeOptions={{ type: 'smoothstep', style: { stroke: 'rgba(23,42,92,0.15)', strokeWidth: 1.5 } }}
  onNodesChange={onNodesChange}   // required: enables drag, selection, and node measurement
  onEdgesChange={onEdgesChange}
  nodesDraggable={true}
  nodesConnectable={false}     // user cannot add edges
  elementsSelectable={true}
  panOnDrag={true}
  zoomOnScroll={true}
  fitView
  fitViewOptions={{ padding: 0.15, duration: 400 }}
  onNodeClick={(_, rfNode) => onSelectNode(rfNode.data.original as ExplorationNode)}
  onPaneClick={() => onSelectNode(null)}
  onNodeDragStart={...}
  onNodeDrag={...}
  onNodeDragStop={...}
  minZoom={0.1}
  maxZoom={4}
>
  <Background color="rgba(23,42,92,0.04)" gap={32} />
  <Controls showInteractive={false} />
  <MiniMap nodeColor={n => nodeConfig(n.data?.type as string).fill} />
</ReactFlow>
```

### `fitView` (inside `GraphCanvas`, called via `useReactFlow`)

Pin to `@xyflow/react ^12.5.0` which fixes `fitView` to work correctly immediately after `setNodes` without async workarounds.

```typescript
const { fitView } = useReactFlow()
useEffect(() => {
  if (rfNodes.length > 0) {
    fitView({ padding: 0.15, duration: 400 })  // no requestAnimationFrame needed in ≥12.5.0
  }
}, [rfNodes.length])  // only when count changes
```

---

## Custom Node: `IdeaNode`

React Flow custom node component:

```tsx
// IdeaNodeType — explicit discriminated union for TypeScript
type IdeaNodeType = RFNode<RFNodeData, 'ideaNode'>

function IdeaNode({ data, selected }: NodeProps<IdeaNodeType>) {
  const cfg = nodeConfig(data.type)
  return (
    <div className={`ideaNode ${selected ? 'ideaNodeSelected' : ''}`}
         style={{ '--node-fill': cfg.fill, '--node-stroke': cfg.stroke, '--node-size': cfg.size + 'px' } as React.CSSProperties}>
      <div className="ideaNodeCircle">
        <span className="ideaNodeGlyph">{cfg.glyph}</span>
      </div>
      <div className="ideaNodeLabel">{data.title}</div>
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
    </div>
  )
}
```

Invisible handles (opacity 0) are needed for React Flow edge routing. Since D3-Force can place nodes in any direction, edges are drawn as `smoothstep` curves from whichever handle is geometrically closer — React Flow handles this automatically with `connectionMode: ConnectionMode.Loose` (default in v12).

---

## Visual Encoding

| Node type | `fill` | `stroke` | `size` (px circle) | `glyph` |
|-----------|--------|----------|---------------------|---------|
| topic | `#0f1f4a` | `#2851b0` | 80 | ✦ |
| direction | `#c56700` | `#f59e0b` | 64 | → |
| opportunity | `#b45309` | `#f59e0b` | 60 | ◈ |
| question | `#1e40af` | `#60a5fa` | 52 | ? |
| hypothesis | `#5b21b6` | `#a78bfa` | 52 | ∿ |
| idea | `#047857` | `#34d399` | 56 | ✦ |
| evidence | `#4b5563` | `#9ca3af` | 40 | · |
| artifact | `#b45309` | `#fbbf24` | 52 | ⬡ |
| claim | `#0e7490` | `#22d3ee` | 48 | ! |
| tension | `#9f1239` | `#fb7185` | 48 | ⚡ |
| decision | `#166534` | `#4ade80` | 48 | ✓ |
| unknown / fallback | `#6b7280` | `#d1d5db` | 44 | · |

Selected: `box-shadow: 0 0 0 3px #fff, 0 0 20px rgba(255,255,255,0.4)` on the circle div.

---

## Entry Animation

```css
@keyframes nodeEnter {
  from { opacity: 0; transform: scale(0.3); }
  to   { opacity: 1; transform: scale(1); }
}
.nodeEnter .ideaNodeCircle,
.nodeEnter .ideaNodeLabel {
  animation: nodeEnter 450ms cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  transform-origin: center;    /* valid — applies to HTML div, not SVG */
  transform-box: fill-box;     /* belt-and-suspenders for SVG context */
}
```

Animation targets the inner `div` elements (not the React Flow wrapper), so React Flow's position `transform` on the wrapper is unaffected. D3-Force naturally moves nodes from their spawn position (near parent) to equilibrium, creating the "flying out" growth effect independent of the CSS animation.

---

## Node Detail Card

```tsx
// Inside GraphCanvas, shown when selectedNode !== null
<div className="nodeDetailCard">
  <span className="nodeTypeBadge" style={{ background: nodeConfig(selectedNode.type).fill }}>
    {selectedNode.type}
  </span>
  <h3>{selectedNode.title}</h3>
  <p className="nodeDetailSummary">{selectedNode.summary}</p>
  {selectedNode.score > 0 && <span className="nodeScore">⚡ {selectedNode.score.toFixed(2)}</span>}
  {(selectedNode.type === 'direction' || selectedNode.type === 'opportunity') && (
    <button className="primaryAction" onClick={() => onExpandOpportunity(selectedNode)}>
      {t('graph.expandButton')}
    </button>
  )}
  <button className="miniAction" onClick={() => onSelectNode(null)}>{t('graph.dismiss')}</button>
</div>
```

`nodeDetailSummary` uses a CSS class with `-webkit-line-clamp: 3` to avoid the React inline style console warning.

---

## App.tsx Changes

```diff
- import { WorkbenchColumns } from './components/WorkbenchColumns'
+ import { GraphView } from './components/GraphView'

- const [selectedOpportunityId, setSelectedOpportunityId] = useState<string | null>(null)
+ const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)

- {exploration && view ? (
-   <WorkbenchColumns ... />
- ) : null}
+ {exploration ? (
+   <GraphView
+     session={exploration}
+     selectedNodeId={selectedNodeId}
+     onSelectNode={(node) => setSelectedNodeId(node?.id ?? null)}
+     onExpandOpportunity={handleExpandOpportunity}
+   />
+ ) : null}
```

`SidebarPanel` remains gated behind `view !== null` — no change.

---

## i18n additions

```typescript
// English
'graph.expandButton': 'Expand branch',
'graph.dismiss': 'Dismiss',
'graph.emptyHint': 'Starting exploration…',

// Chinese
'graph.expandButton': '展开此分支',
'graph.dismiss': '关闭',
'graph.emptyHint': '探索启动中…',
```

---

## Dependencies

```
@xyflow/react ^12.5.0   — React Flow v12 (rendering, zoom, pan, drag, selection)
d3-force                — D3 force simulation (position calculation only)
@types/d3-force         — TypeScript types (devDependency)
```

Only `d3-force` is needed — not the full `d3` package. This keeps bundle size minimal.

---

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/GraphView.tsx` | New |
| `frontend/src/App.tsx` | Replace WorkbenchColumns with GraphView |
| `frontend/src/App.css` | Add `.graphContainer`, `.ideaNode`, `.nodeDetailCard`, `.nodeEnter` |
| `frontend/src/lib/i18n.ts` | Add 3 graph keys per language |
| `frontend/package.json` | Add @xyflow/react, d3-force, @types/d3-force |

## Pre-Implementation Notes

1. **Add i18n keys first.** Add `graph.expandButton`, `graph.dismiss`, `graph.emptyHint` to both `translations.en` and `translations.zh` in `i18n.ts` _before_ writing `GraphView.tsx`. The `TranslationKey` type is derived from `as const` so TypeScript will reject unknown keys at compile time.

2. **`fitView` dependency array.** Add `fitView` to the `useEffect` dependency array in `GraphCanvas` (or add a justified `// eslint-disable-next-line` comment), as `exhaustive-deps` will warn otherwise.

3. **`buildWorkbenchView` opportunity ID.** After removing `selectedOpportunityId` from `App.tsx`, pass `exploration.activeOpportunityId` (server-authoritative) as the second argument to `buildWorkbenchView` so `SidebarPanel`'s strategy presets continue to work correctly.

## Out of Scope

- Graph export, edge labels, node editing, server-side layout
