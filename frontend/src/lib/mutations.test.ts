import { describe, expect, it } from 'vitest'
import { createExplorationSession } from './workbench'
import { applyExplorationMutations } from './mutations'

describe('applyExplorationMutations', () => {
  it('applies runtime node/edge/run mutations incrementally', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost',
    })

    const baseNodeCount = session.nodes.length
    const baseEdgeCount = session.edges.length
    const baseRunCount = session.runs.length

    const next = applyExplorationMutations(session, [
      {
        id: 'm-1',
        workspace_id: session.id,
        kind: 'node_added',
        source: 'runtime',
        node: {
          id: 'idea-extra',
          sessionId: session.id,
          type: 'idea',
          title: 'Extra idea',
          summary: 'Extra summary',
          status: 'active',
          score: 0.7,
          depth: 4,
          parentContext: session.activeOpportunityId,
          metadata: {
            branchId: session.activeOpportunityId,
            slot: 'idea-99',
          },
          evidenceSummary: 'runtime',
        },
        created_at: Date.now(),
      },
      {
        id: 'm-2',
        workspace_id: session.id,
        kind: 'edge_added',
        source: 'runtime',
        edge: {
          id: 'edge-extra',
          from: session.nodes[0].id,
          to: 'idea-extra',
          type: 'expands',
        },
        created_at: Date.now(),
      },
      {
        id: 'm-3',
        workspace_id: session.id,
        kind: 'run_added',
        source: 'runtime',
        run: {
          id: 'run-99',
          round: 99,
          focus: session.activeOpportunityId,
          summary: 'runtime expand',
        },
        created_at: Date.now(),
      },
    ])

    expect(next.nodes.length).toBe(baseNodeCount + 1)
    expect(next.edges.length).toBe(baseEdgeCount + 1)
    expect(next.runs.length).toBe(baseRunCount + 1)
  })

  it('applies favorite and active opportunity mutations', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost',
    })
    const targetOpportunity = session.nodes.find((node) => node.type === 'opportunity' && node.id !== session.activeOpportunityId)
    const targetIdea = session.nodes.find((node) => node.type === 'idea')
    expect(targetOpportunity).toBeDefined()
    expect(targetIdea).toBeDefined()

    const next = applyExplorationMutations(session, [
      {
        id: 'm-4',
        workspace_id: session.id,
        kind: 'active_opportunity_set',
        source: 'intervention',
        active_opportunity_id: targetOpportunity!.id,
        created_at: Date.now(),
      },
      {
        id: 'm-5',
        workspace_id: session.id,
        kind: 'favorites_updated',
        source: 'intervention',
        favorites: [targetIdea!.id],
        created_at: Date.now(),
      },
    ])

    expect(next.activeOpportunityId).toBe(targetOpportunity!.id)
    expect(next.favorites).toContain(targetIdea!.id)
  })

  it('applies strategy update mutation', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost',
    })

    const next = applyExplorationMutations(session, [
      {
        id: 'm-6',
        workspace_id: session.id,
        kind: 'strategy_updated',
        source: 'intervention',
        strategy: {
          interval_ms: 1200,
          max_runs: 60,
          expansion_mode: 'round_robin',
        },
        created_at: Date.now(),
      },
    ])

    expect(next.strategy?.interval_ms).toBe(1200)
    expect(next.strategy?.max_runs).toBe(60)
    expect(next.strategy?.expansion_mode).toBe('round_robin')
  })
})
