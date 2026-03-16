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
