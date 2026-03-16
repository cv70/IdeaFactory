import * as d3 from 'd3-force'
import {
  useEffect, useRef, useMemo,
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
    // rfNodes intentionally omitted: including it would re-run this effect on every D3 tick (position updates).
    // enteringIds is computed once per data change, which is the correct granularity.
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
