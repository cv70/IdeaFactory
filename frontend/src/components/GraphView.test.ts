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
