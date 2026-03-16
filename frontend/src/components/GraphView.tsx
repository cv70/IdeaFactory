import * as d3 from 'd3-force'
import {
  useState, useEffect, useRef, useMemo,
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
