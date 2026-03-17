import { describe, it, expect } from 'vitest'
import { nodeConfig, nodeRadius, buildRFNode, buildRFEdge, circleBoundaryPoints } from './GraphView'
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

describe('nodeRadius', () => {
  it('returns half the node size', () => {
    expect(nodeRadius('topic')).toBe(40)    // size 80 / 2
    expect(nodeRadius('idea')).toBe(28)     // size 56 / 2
    expect(nodeRadius('unknown')).toBe(22)  // fallback size 44 / 2
  })
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
    expect(rfEdge.type).toBe('floating')
  })
})

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
