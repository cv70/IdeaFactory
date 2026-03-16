# Graph View Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 3-column `WorkbenchColumns` with a live, force-directed graph that grows in real time as the agent produces nodes, using React Flow (rendering) + D3-Force (layout physics).

**Architecture:** `GraphView` (outer, owns D3 simulation + ReactFlowProvider) → `GraphCanvas` (inner, lives inside `<ReactFlow>`, calls `useReactFlow()` for fitView, renders `NodeDetailCard`). Custom `IdeaNode` component renders each node as a circle + label with type-specific visual encoding. D3-Force mutates node positions on each tick; `onTick` pushes them to React Flow via `setRfNodes`.

**Tech Stack:** `@xyflow/react ^12.5.0`, `d3-force`, `@types/d3-force` (dev), React 19, TypeScript, Vitest

**Spec:** `docs/superpowers/specs/2026-03-16-graph-view-design.md`

---

## Chunk 1: Setup — Dependencies, i18n, CSS import

### Task 1: Install dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install packages**

```bash
cd frontend && npm install @xyflow/react@^12.5.0 d3-force && npm install --save-dev @types/d3-force
```

Expected: packages install without error; `package.json` gains `@xyflow/react` under `dependencies`, `d3-force` under `dependencies`, `@types/d3-force` under `devDependencies`.

- [ ] **Step 2: Verify no broken types**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors (or only pre-existing errors in `.test.` files).

- [ ] **Step 3: Commit**

```bash
cd frontend && git add package.json package-lock.json && git commit -m "feat: add @xyflow/react, d3-force dependencies"
```

---

### Task 2: Add i18n keys for graph view

**Files:**
- Modify: `frontend/src/lib/i18n.ts`

**Why first:** `TranslationKey` is derived from `as const` — TypeScript rejects unknown keys at compile time. `GraphView.tsx` calls `t('graph.expandButton')` etc., so these must exist before the component is created.

- [ ] **Step 1: Add English keys to i18n.ts**

In `frontend/src/lib/i18n.ts`, inside the `en` object after the last `'sidebar.runs.*'` entries (before the closing `},`), add:

```typescript
    // Graph view
    'graph.expandButton': 'Expand branch',
    'graph.dismiss': 'Dismiss',
    'graph.emptyHint': 'Starting exploration…',
```

- [ ] **Step 2: Add Chinese keys to i18n.ts**

In `frontend/src/lib/i18n.ts`, inside the `zh` object after the last `'sidebar.runs.*'` entries (before the closing `},`), add:

```typescript
    // Graph view
    'graph.expandButton': '展开此分支',
    'graph.dismiss': '关闭',
    'graph.emptyHint': '探索启动中…',
```

- [ ] **Step 3: Verify TypeScript accepts the new keys**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/i18n.ts && git commit -m "feat: add graph i18n keys (en + zh)"
```

---

### Task 3: Add React Flow base CSS import

**Files:**
- Modify: `frontend/src/index.css`

React Flow requires its own base stylesheet for handles, edges, controls, and minimap to render. Must be imported before any custom overrides.

- [ ] **Step 1: Add the import to index.css**

Add this line at the top of `frontend/src/index.css`, before the Google Fonts import:

```css
@import '@xyflow/react/dist/style.css';
```

Final top of file should look like:
```css
@import '@xyflow/react/dist/style.css';
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300..700&display=swap');
```

- [ ] **Step 2: Verify build still succeeds**

```bash
cd frontend && npx vite build 2>&1 | tail -5
```

Expected: `✓ built in` line, no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/index.css && git commit -m "feat: import React Flow base CSS"
```

---

## Chunk 2: GraphView.tsx — Utilities, IdeaNode, GraphCanvas, GraphView

### Task 4: Write utility tests (nodeConfig, buildRFNode, buildRFEdge)

These are pure functions extracted from GraphView. Test them first before implementing.

**Files:**
- Create: `frontend/src/components/GraphView.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/components/GraphView.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { nodeConfig, buildRFNode, buildRFEdge } from './GraphView'
import type { Node as ExplorationNode, Edge as ExplorationEdge } from '../types/exploration'

const makeNode = (overrides: Partial<ExplorationNode> = {}): ExplorationNode => ({
  id: 'n1',
  sessionId: 's1',
  type: 'idea',
  title: 'Test idea',
  summary: 'A summary',
  status: 'active',
  score: 0.8,
  depth: 1,
  metadata: {},
  evidenceSummary: '',
  ...overrides,
})

describe('nodeConfig', () => {
  it('returns correct config for topic', () => {
    const cfg = nodeConfig('topic')
    expect(cfg.fill).toBe('#0f1f4a')
    expect(cfg.size).toBe(80)
    expect(cfg.glyph).toBe('✦')
  })

  it('returns correct config for idea', () => {
    const cfg = nodeConfig('idea')
    expect(cfg.fill).toBe('#047857')
    expect(cfg.size).toBe(56)
  })

  it('returns fallback for unknown type', () => {
    const cfg = nodeConfig('nonexistent')
    expect(cfg.fill).toBe('#6b7280')
    expect(cfg.size).toBe(44)
  })
})

describe('buildRFNode', () => {
  it('builds a React Flow node from ExplorationNode', () => {
    const node = makeNode({ id: 'n1', type: 'question' })
    const rfNode = buildRFNode(node, false)
    expect(rfNode.id).toBe('n1')
    expect(rfNode.type).toBe('ideaNode')
    expect(rfNode.data.type).toBe('question')
    expect(rfNode.data.title).toBe('Test idea')
    expect(rfNode.hidden).toBe(false)
  })

  it('marks entering nodes as hidden', () => {
    const node = makeNode()
    const rfNode = buildRFNode(node, true)
    expect(rfNode.hidden).toBe(true)
  })
})

describe('buildRFEdge', () => {
  it('builds a React Flow edge from ExplorationEdge', () => {
    const edge: ExplorationEdge = { id: 'e1', from: 'n1', to: 'n2', type: 'leads_to' }
    const rfEdge = buildRFEdge(edge)
    expect(rfEdge.id).toBe('e1')
    expect(rfEdge.source).toBe('n1')
    expect(rfEdge.target).toBe('n2')
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts 2>&1 | tail -15
```

Expected: FAIL — `GraphView.test.ts` cannot find `nodeConfig`, `buildRFNode`, `buildRFEdge` (not yet exported).

---

### Task 5: Create GraphView.tsx — exported utilities

**Files:**
- Create: `frontend/src/components/GraphView.tsx`

- [ ] **Step 1: Create the file with types and exported utilities**

Create `frontend/src/components/GraphView.tsx` with just the utility layer first:

```typescript
import * as d3 from 'd3-force'
import {
  useState, useEffect, useRef, useMemo,
  type ReactNode,
} from 'react'
import {
  ReactFlow, ReactFlowProvider,
  useNodesState, useEdgesState, useReactFlow,
  Background, Controls, MiniMap,
  Handle, Position,
  type NodeProps,
} from '@xyflow/react'
import type { Node as RFNode, Edge as RFEdge } from '@xyflow/react'
import type { Node as ExplorationNode, Edge as ExplorationEdge, ExplorationSession } from '../types/exploration'
import { useTranslation } from '../lib/i18n'

// ─── Domain types ─────────────────────────────────────────────────────────────

export type RFNodeData = {
  type: string
  title: string
  original: ExplorationNode
}

type IdeaNodeType = RFNode<RFNodeData, 'ideaNode'>

type SimNode = d3.SimulationNodeDatum & {
  id: string
  type: string
  title: string
  [key: string]: unknown
}

type SimLink = d3.SimulationLinkDatum<SimNode>

// ─── Visual encoding ──────────────────────────────────────────────────────────

type NodeVisual = { fill: string; stroke: string; size: number; glyph: string }

const NODE_CONFIGS: Record<string, NodeVisual> = {
  topic:       { fill: '#0f1f4a', stroke: '#2851b0', size: 80, glyph: '✦' },
  direction:   { fill: '#c56700', stroke: '#f59e0b', size: 64, glyph: '→' },
  opportunity: { fill: '#b45309', stroke: '#f59e0b', size: 60, glyph: '◈' },
  question:    { fill: '#1e40af', stroke: '#60a5fa', size: 52, glyph: '?' },
  hypothesis:  { fill: '#5b21b6', stroke: '#a78bfa', size: 52, glyph: '∿' },
  idea:        { fill: '#047857', stroke: '#34d399', size: 56, glyph: '✦' },
  evidence:    { fill: '#4b5563', stroke: '#9ca3af', size: 40, glyph: '·' },
  artifact:    { fill: '#b45309', stroke: '#fbbf24', size: 52, glyph: '⬡' },
  claim:       { fill: '#0e7490', stroke: '#22d3ee', size: 48, glyph: '!' },
  tension:     { fill: '#9f1239', stroke: '#fb7185', size: 48, glyph: '⚡' },
  decision:    { fill: '#166534', stroke: '#4ade80', size: 48, glyph: '✓' },
}

const FALLBACK_CONFIG: NodeVisual = { fill: '#6b7280', stroke: '#d1d5db', size: 44, glyph: '·' }

export function nodeConfig(type: string): NodeVisual {
  return NODE_CONFIGS[type] ?? FALLBACK_CONFIG
}

export function nodeRadius(type: string): number {
  return nodeConfig(type).size / 2
}

// ─── RF node / edge builders ──────────────────────────────────────────────────

export function buildRFNode(node: ExplorationNode, entering: boolean): RFNode<RFNodeData, 'ideaNode'> {
  return {
    id: node.id,
    type: 'ideaNode',
    position: { x: 0, y: 0 },
    hidden: entering,
    data: { type: node.type, title: node.title, original: node },
  }
}

export function buildRFEdge(edge: ExplorationEdge): RFEdge {
  return { id: edge.id, source: edge.from, target: edge.to }
}
```

**Note:** Leave the rest of the file empty for now — we'll add the components in the following steps. The file should compile and export the utilities.

- [ ] **Step 2: Run the failing tests — they should now pass**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts 2>&1 | tail -15
```

Expected: 7 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/GraphView.tsx frontend/src/components/GraphView.test.ts
git commit -m "feat: add GraphView utility functions + tests (nodeConfig, buildRFNode, buildRFEdge)"
```

---

### Task 6: Add IdeaNode component to GraphView.tsx

**Files:**
- Modify: `frontend/src/components/GraphView.tsx`

- [ ] **Step 1: Append IdeaNode to GraphView.tsx**

Append after the `buildRFEdge` function:

```typescript
// ─── Custom node ──────────────────────────────────────────────────────────────

function IdeaNode({ data, selected }: NodeProps<IdeaNodeType>) {
  const cfg = nodeConfig(data.type)
  return (
    <div
      className={`ideaNode${selected ? ' ideaNodeSelected' : ''}`}
      style={{
        '--node-fill': cfg.fill,
        '--node-stroke': cfg.stroke,
        '--node-size': `${cfg.size}px`,
      } as React.CSSProperties}
    >
      <div className="ideaNodeCircle">
        <span className="ideaNodeGlyph">{cfg.glyph}</span>
      </div>
      <div className="ideaNodeLabel">{data.title}</div>
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
    </div>
  )
}

const nodeTypes = { ideaNode: IdeaNode }
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/GraphView.tsx && git commit -m "feat: add IdeaNode custom React Flow node"
```

---

### Task 7: Add GraphCanvas (inner component) to GraphView.tsx

**Files:**
- Modify: `frontend/src/components/GraphView.tsx`

`GraphCanvas` must live inside `<ReactFlow>` so it can call `useReactFlow()`. It owns the `fitView` effect and renders the `NodeDetailCard`.

- [ ] **Step 1: Append GraphCanvas to GraphView.tsx**

Append after `const nodeTypes = { ideaNode: IdeaNode }`:

```typescript
// ─── Node detail card ─────────────────────────────────────────────────────────

type NodeDetailCardProps = {
  selectedNode: ExplorationNode | null
  onSelectNode: (node: ExplorationNode | null) => void
  onExpandOpportunity: (node: ExplorationNode) => void
}

function NodeDetailCard({ selectedNode, onSelectNode, onExpandOpportunity }: NodeDetailCardProps) {
  const { t } = useTranslation()
  if (!selectedNode) return null
  const cfg = nodeConfig(selectedNode.type)
  return (
    <div className="nodeDetailCard">
      <span className="nodeTypeBadge" style={{ background: cfg.fill }}>
        {selectedNode.type}
      </span>
      <h3>{selectedNode.title}</h3>
      <p className="nodeDetailSummary">{selectedNode.summary}</p>
      {selectedNode.score > 0 && (
        <span className="nodeScore">⚡ {selectedNode.score.toFixed(2)}</span>
      )}
      {(selectedNode.type === 'direction' || selectedNode.type === 'opportunity') && (
        <button className="primaryAction" onClick={() => onExpandOpportunity(selectedNode)}>
          {t('graph.expandButton')}
        </button>
      )}
      <button className="miniAction" onClick={() => onSelectNode(null)}>
        {t('graph.dismiss')}
      </button>
    </div>
  )
}

// ─── GraphCanvas (inner — must live inside <ReactFlow>) ───────────────────────

type GraphCanvasProps = {
  rfNodes: RFNode<RFNodeData, 'ideaNode'>[]
  rfEdges: RFEdge[]
  onNodesChange: ReturnType<typeof useNodesState<RFNode<RFNodeData, 'ideaNode'>>>[2]
  onEdgesChange: ReturnType<typeof useEdgesState>[2]
  selectedNode: ExplorationNode | null
  onSelectNode: (node: ExplorationNode | null) => void
  onExpandOpportunity: (node: ExplorationNode) => void
  onNodeDragStart: (event: React.MouseEvent, node: RFNode) => void
  onNodeDrag: (event: React.MouseEvent, node: RFNode) => void
  onNodeDragStop: (event: React.MouseEvent, node: RFNode) => void
}

function GraphCanvas({
  rfNodes,
  rfEdges,
  onNodesChange,
  onEdgesChange,
  selectedNode,
  onSelectNode,
  onExpandOpportunity,
  onNodeDragStart,
  onNodeDrag,
  onNodeDragStop,
}: GraphCanvasProps) {
  const { fitView } = useReactFlow()

  useEffect(() => {
    if (rfNodes.length > 0) {
      fitView({ padding: 0.15, duration: 400 })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rfNodes.length, fitView])

  return (
    <>
      <ReactFlow
        nodes={rfNodes}
        edges={rfEdges}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={{
          type: 'smoothstep',
          style: { stroke: 'rgba(23,42,92,0.15)', strokeWidth: 1.5 },
        }}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodesDraggable={true}
        nodesConnectable={false}
        elementsSelectable={true}
        panOnDrag={true}
        zoomOnScroll={true}
        fitView
        fitViewOptions={{ padding: 0.15, duration: 400 }}
        onNodeClick={(_, rfNode) =>
          onSelectNode(rfNode.data.original as ExplorationNode)
        }
        onPaneClick={() => onSelectNode(null)}
        onNodeDragStart={onNodeDragStart}
        onNodeDrag={onNodeDrag}
        onNodeDragStop={onNodeDragStop}
        minZoom={0.1}
        maxZoom={4}
      >
        <Background color="rgba(23,42,92,0.04)" gap={32} />
        <Controls showInteractive={false} />
        <MiniMap nodeColor={(n) => nodeConfig(n.data?.type as string).fill} />
      </ReactFlow>
      <NodeDetailCard
        selectedNode={selectedNode}
        onSelectNode={onSelectNode}
        onExpandOpportunity={onExpandOpportunity}
      />
    </>
  )
}
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/GraphView.tsx && git commit -m "feat: add GraphCanvas and NodeDetailCard"
```

---

### Task 8: Add GraphView (outer component) to GraphView.tsx

**Files:**
- Modify: `frontend/src/components/GraphView.tsx`

This is the main exported component. Owns D3 simulation (via `useRef`), `simNodeMap`, drag state, `useNodesState`/`useEdgesState`, data prep `useMemo`, and simulation sync `useEffect`.

- [ ] **Step 1: Append GraphView to GraphView.tsx**

Append after the `GraphCanvas` function:

```typescript
// ─── GraphView (outer — owns D3 sim, ReactFlowProvider) ──────────────────────

export type GraphViewProps = {
  session: ExplorationSession
  selectedNodeId: string | null
  onSelectNode: (node: ExplorationNode | null) => void
  onExpandOpportunity: (node: ExplorationNode) => void
}

export function GraphView({ session, selectedNodeId, onSelectNode, onExpandOpportunity }: GraphViewProps) {
  const { t } = useTranslation()

  // D3 simulation refs
  const simRef = useRef<d3.Simulation<SimNode, SimLink> | null>(null)
  const simNodeMap = useRef<Map<string, SimNode>>(new Map())

  // React Flow state
  const [rfNodes, setRfNodes, onNodesChange] = useNodesState<RFNode<RFNodeData, 'ideaNode'>>([])
  const [rfEdges, setRfEdges, onEdgesChange] = useEdgesState([])

  // Derived selected node
  const selectedNode = useMemo(
    () => (selectedNodeId ? (session.nodes.find((n) => n.id === selectedNodeId) ?? null) : null),
    [session.nodes, selectedNodeId],
  )

  // ─── Data preparation ────────────────────────────────────────────────────────
  const { allNodes, allEdges } = useMemo(() => {
    // 1. Anchor: find real topic node or create synthetic
    const realTopic = session.nodes.find((n) => n.type === 'topic')
    const anchorId = realTopic?.id ?? `__topic__${session.id}`
    const syntheticTopic: ExplorationNode | null = realTopic
      ? null
      : {
          id: anchorId,
          sessionId: session.id,
          type: 'topic',
          title: session.topic,
          summary: session.outputGoal,
          status: 'active',
          score: 1,
          depth: 0,
          metadata: {},
          evidenceSummary: '',
        }

    // 2. Synthetic edges: connect unlinked direction/opportunity nodes to anchor
    const incomingIds = new Set(session.edges.map((e) => e.to))
    const syntheticEdges: ExplorationEdge[] = session.nodes
      .filter((n) => (n.type === 'direction' || n.type === 'opportunity') && !incomingIds.has(n.id))
      .map((n) => ({ id: `__e__${anchorId}__${n.id}`, from: anchorId, to: n.id, type: 'leads_to' as const }))

    return {
      allNodes: syntheticTopic ? [syntheticTopic, ...session.nodes] : session.nodes,
      allEdges: [...session.edges, ...syntheticEdges],
    }
  }, [session.nodes, session.edges, session.id, session.topic, session.outputGoal])

  // ─── D3 simulation init (once at mount) ──────────────────────────────────────
  useEffect(() => {
    function onTick() {
      setRfNodes((prev) =>
        prev.map((rfNode) => {
          const simNode = simNodeMap.current.get(rfNode.id)
          if (!simNode || simNode.x == null) return rfNode
          return { ...rfNode, position: { x: simNode.x, y: simNode.y } }
        }),
      )
    }

    const sim = d3
      .forceSimulation<SimNode>()
      .force(
        'link',
        d3.forceLink<SimNode, SimLink>().id((d) => d.id).distance(130).strength(0.4),
      )
      .force('charge', d3.forceManyBody<SimNode>().strength((d) => -(nodeRadius(d.type as string) * 18)))
      .force('center', d3.forceCenter(0, 0).strength(0.05))
      .force('collide', d3.forceCollide<SimNode>((d) => nodeRadius(d.type as string) + 12).strength(0.8))
      .alphaDecay(0.02)
      .on('tick', onTick)

    simRef.current = sim
    return () => { sim.stop() }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ─── Simulation sync (on session data change) ─────────────────────────────────
  useEffect(() => {
    const sim = simRef.current
    if (!sim) return

    const map = simNodeMap.current
    let hasNew = false

    // 1. Add new nodes, update existing
    for (const node of allNodes) {
      if (!map.has(node.id)) {
        const parentEdge = allEdges.find((e) => e.to === node.id)
        const parent = parentEdge ? map.get(parentEdge.from) : null
        map.set(node.id, {
          ...node,
          x: parent?.x != null ? parent.x + (Math.random() - 0.5) * 30 : (Math.random() - 0.5) * 60,
          y: parent?.y != null ? parent.y + (Math.random() - 0.5) * 30 : (Math.random() - 0.5) * 60,
        } as SimNode)
        hasNew = true
      } else {
        Object.assign(map.get(node.id)!, node)
      }
    }

    // 2. Remove stale nodes
    const nodeIds = new Set(allNodes.map((n) => n.id))
    for (const id of map.keys()) {
      if (!nodeIds.has(id)) map.delete(id)
    }

    // 3. Update RF nodes (entering nodes start hidden)
    const enteringIds = new Set(
      allNodes.filter((n) => !rfNodes.find((r) => r.id === n.id)).map((n) => n.id),
    )
    setRfNodes(allNodes.map((n) => buildRFNode(n, enteringIds.has(n.id))))
    setRfEdges(
      allEdges
        .filter((e) => nodeIds.has(e.from) && nodeIds.has(e.to))
        .map(buildRFEdge),
    )

    // 4. Update D3 simulation
    sim.nodes([...map.values()])
    const linkForce = sim.force<d3.ForceLink<SimNode, SimLink>>('link')
    linkForce?.links(
      allEdges
        .filter((e) => nodeIds.has(e.from) && nodeIds.has(e.to))
        .map((e) => ({ source: e.from, target: e.to })),
    )

    // 5. Restart simulation on new nodes
    if (hasNew) sim.alpha(0.5).restart()

    // 6. Pin topic node at origin
    for (const simNode of map.values()) {
      if (simNode.type === 'topic') {
        simNode.fx = 0
        simNode.fy = 0
      }
    }

    // 7. Reveal entering nodes with animation
    if (enteringIds.size > 0) {
      const t1 = setTimeout(() => {
        setRfNodes((prev) =>
          prev.map((n) => (enteringIds.has(n.id) ? { ...n, hidden: false, className: 'nodeEnter' } : n)),
        )
      }, 16)
      const t2 = setTimeout(() => {
        setRfNodes((prev) =>
          prev.map((n) => (enteringIds.has(n.id) ? { ...n, className: '' } : n)),
        )
      }, 520)
      return () => {
        clearTimeout(t1)
        clearTimeout(t2)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allNodes, allEdges])

  // ─── Drag handlers (React Flow → D3 fx/fy) ───────────────────────────────────
  function handleNodeDragStart(_event: React.MouseEvent, node: RFNode) {
    const simNode = simNodeMap.current.get(node.id)
    if (simNode) {
      simNode.fx = node.position.x
      simNode.fy = node.position.y
    }
    simRef.current?.alphaTarget(0.3).restart()
  }

  function handleNodeDrag(_event: React.MouseEvent, node: RFNode) {
    const simNode = simNodeMap.current.get(node.id)
    if (simNode) {
      simNode.fx = node.position.x
      simNode.fy = node.position.y
    }
  }

  function handleNodeDragStop(_event: React.MouseEvent, node: RFNode) {
    const simNode = simNodeMap.current.get(node.id)
    if (simNode) {
      simNode.fx = null
      simNode.fy = null
    }
    simRef.current?.alphaTarget(0)
  }

  // ─── Render ──────────────────────────────────────────────────────────────────
  if (allNodes.length === 0) {
    return (
      <div className="graphContainer graphEmpty">
        <p>{t('graph.emptyHint')}</p>
      </div>
    )
  }

  return (
    <div className="graphContainer">
      <ReactFlowProvider>
        <GraphCanvas
          rfNodes={rfNodes}
          rfEdges={rfEdges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          selectedNode={selectedNode}
          onSelectNode={onSelectNode}
          onExpandOpportunity={onExpandOpportunity}
          onNodeDragStart={handleNodeDragStart}
          onNodeDrag={handleNodeDrag}
          onNodeDragStop={handleNodeDragStop}
        />
      </ReactFlowProvider>
    </div>
  )
}
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors.

- [ ] **Step 3: Run existing tests to ensure nothing is broken**

```bash
cd frontend && npx vitest run src/components/GraphView.test.ts 2>&1 | tail -10
```

Expected: 7 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/GraphView.tsx && git commit -m "feat: complete GraphView component with D3-Force simulation"
```

---

## Chunk 3: App.tsx wiring + CSS styles + build

### Task 9: Update App.tsx to use GraphView

**Files:**
- Modify: `frontend/src/App.tsx`

Three changes per the spec:
1. Import `GraphView` instead of `WorkbenchColumns`
2. Rename `selectedOpportunityId` → `selectedNodeId`
3. Pass `exploration.activeOpportunityId` to `buildWorkbenchView` (server-authoritative ID for SidebarPanel strategy presets)

- [ ] **Step 1: Replace the WorkbenchColumns import**

In `frontend/src/App.tsx`, change:
```typescript
import { WorkbenchColumns } from './components/WorkbenchColumns'
```
to:
```typescript
import { GraphView } from './components/GraphView'
```

- [ ] **Step 2: Rename selectedOpportunityId → selectedNodeId**

In `frontend/src/App.tsx`:

Change the state declaration:
```typescript
// old:
const [selectedOpportunityId, setSelectedOpportunityId] = useState<string | null>(null)
// new:
const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
```

Update all references (use replace-all):
- `selectedOpportunityId` → `selectedNodeId`
- `setSelectedOpportunityId` → `setSelectedNodeId`

There are approximately 5 occurrences total:
1. `useState` declaration
2. `buildWorkbenchView` call in `useMemo`
3. `onSnapshot` callback (2 places)
4. `handleSubmit` (at end, setting after create)
5. `handleSelectWorkspace`
6. `handleExpandOpportunity`
7. `handleSelectOpportunity` function body

- [ ] **Step 3: Fix buildWorkbenchView to use server-authoritative ID**

In `frontend/src/App.tsx`, find the `useMemo` that calls `buildWorkbenchView`:

```typescript
// old:
const view = useMemo(() => {
  if (!exploration) return null
  return buildWorkbenchView(exploration, selectedNodeId ?? undefined)
}, [exploration, selectedNodeId])
```

Change to pass `exploration.activeOpportunityId` as fallback to ensure SidebarPanel strategy presets work correctly even when `selectedNodeId` is null:

```typescript
// new:
const view = useMemo(() => {
  if (!exploration) return null
  return buildWorkbenchView(exploration, exploration.activeOpportunityId)
}, [exploration])
```

Note: `selectedNodeId` is now only used for highlighting the selected node in the graph. `SidebarPanel` uses `view.activeOpportunity` which is derived from `exploration.activeOpportunityId`.

- [ ] **Step 4: Replace the WorkbenchColumns JSX with GraphView**

In `frontend/src/App.tsx`, find:
```typescript
{exploration && view ? (
  <WorkbenchColumns
    session={{
      ...exploration,
      activeOpportunityId: selectedOpportunityId ?? exploration.activeOpportunityId,
    }}
    view={view}
    onSelectOpportunity={handleSelectOpportunity}
    onExpandOpportunity={handleExpandOpportunity}
    onToggleFavorite={handleToggleFavorite}
  />
) : null}
```

Replace with:
```typescript
{exploration ? (
  <GraphView
    session={exploration}
    selectedNodeId={selectedNodeId}
    onSelectNode={(node) => setSelectedNodeId(node?.id ?? null)}
    onExpandOpportunity={handleExpandOpportunity}
  />
) : null}
```

- [ ] **Step 5: Remove the now-unused handleSelectOpportunity function**

Delete the function:
```typescript
function handleSelectOpportunity(opportunity: Node) {
  setSelectedOpportunityId(opportunity.id)
}
```

- [ ] **Step 6: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -20
```

Expected: zero errors.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/App.tsx && git commit -m "feat: wire GraphView into App, replace WorkbenchColumns"
```

---

### Task 10: Add graph CSS styles to App.css

**Files:**
- Modify: `frontend/src/App.css`

Add all graph-specific CSS classes. The `graphContainer` needs explicit height because React Flow requires a positioned container with known dimensions.

- [ ] **Step 1: Append graph styles to the end of App.css**

Append to `frontend/src/App.css`:

```css
/* ─── Graph View ──────────────────────────────────────────────────── */

.graphContainer {
  position: relative;
  height: 70vh;
  min-height: 480px;
  border: 1px solid var(--panel-border);
  border-radius: 24px;
  background: var(--panel-bg);
  box-shadow: var(--panel-shadow);
  backdrop-filter: blur(20px);
  overflow: hidden;
}

.graphEmpty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--muted-ink);
  font-style: italic;
  font-size: 0.9rem;
}

/* React Flow wrapper must fill the container */
.graphContainer .react-flow {
  border-radius: 24px;
}

/* ─── Idea Node ───────────────────────────────────────────────────── */

.ideaNode {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.ideaNodeCircle {
  width: var(--node-size);
  height: var(--node-size);
  border-radius: 50%;
  background: var(--node-fill);
  border: 2px solid var(--node-stroke);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  transition: box-shadow 150ms ease;
}

.ideaNodeSelected .ideaNodeCircle {
  box-shadow: 0 0 0 3px #fff, 0 0 20px rgba(255, 255, 255, 0.4);
}

.ideaNodeGlyph {
  font-size: calc(var(--node-size) * 0.36);
  color: rgba(255, 255, 255, 0.9);
  line-height: 1;
  user-select: none;
}

.ideaNodeLabel {
  max-width: 120px;
  text-align: center;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--main-ink);
  line-height: 1.3;
  word-break: break-word;
  pointer-events: none;
}

/* ─── Node Entry Animation ────────────────────────────────────────── */

@keyframes nodeEnter {
  from { opacity: 0; transform: scale(0.3); }
  to   { opacity: 1; transform: scale(1); }
}

.nodeEnter .ideaNodeCircle,
.nodeEnter .ideaNodeLabel {
  animation: nodeEnter 450ms cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  transform-origin: center;
  transform-box: fill-box;
}

/* ─── Node Detail Card ────────────────────────────────────────────── */

.nodeDetailCard {
  position: absolute;
  bottom: 1.5rem;
  right: 1.5rem;
  width: 280px;
  padding: 1.2rem;
  border: 1px solid var(--panel-border);
  border-radius: 20px;
  background: rgba(255, 252, 248, 0.96);
  box-shadow: 0 8px 32px rgba(30, 45, 90, 0.12);
  backdrop-filter: blur(12px);
  display: grid;
  gap: 0.6rem;
  z-index: 10;
}

.nodeTypeBadge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  color: #fff;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  width: fit-content;
}

.nodeDetailCard h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  color: var(--main-ink);
}

.nodeDetailSummary {
  margin: 0;
  font-size: 0.84rem;
  color: var(--muted-ink);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.nodeScore {
  font-size: 0.8rem;
  color: var(--accent-ink);
  font-weight: 600;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/App.css && git commit -m "feat: add graph container, IdeaNode, NodeDetailCard CSS"
```

---

### Task 11: Full build verification

- [ ] **Step 1: TypeScript strict check (non-test files)**

```bash
cd frontend && npx tsc --noEmit --skipLibCheck 2>&1 | grep -v "\.test\." | head -30
```

Expected: zero errors.

- [ ] **Step 2: Run all tests**

```bash
cd frontend && npx vitest run 2>&1 | tail -15
```

Expected: all tests pass (GraphView.test.ts: 7 passes; other test files may have pre-existing errors — only failures in new files are blocking).

- [ ] **Step 3: Production build**

```bash
cd frontend && npx vite build 2>&1 | tail -8
```

Expected: `✓ built in` line, no errors.

- [ ] **Step 4: Commit if any remaining files are unstaged**

```bash
cd frontend && git status
```

If clean, no commit needed. If not, stage and commit remaining changes.

---

## Summary of files changed

| File | Change |
|------|--------|
| `frontend/package.json` | Add @xyflow/react, d3-force, @types/d3-force |
| `frontend/src/lib/i18n.ts` | Add graph.expandButton, graph.dismiss, graph.emptyHint (EN + ZH) |
| `frontend/src/index.css` | Add @xyflow/react CSS import at top |
| `frontend/src/components/GraphView.tsx` | New — full graph component |
| `frontend/src/components/GraphView.test.ts` | New — utility function tests |
| `frontend/src/App.tsx` | Replace WorkbenchColumns with GraphView, rename selectedOpportunityId → selectedNodeId |
| `frontend/src/App.css` | Add graphContainer, ideaNode, nodeEnter, nodeDetailCard styles |
