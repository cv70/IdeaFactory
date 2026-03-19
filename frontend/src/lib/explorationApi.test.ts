import { describe, expect, it } from 'vitest'
import {
  createExploration,
  expandOpportunity as expandOpportunityRequest,
  getExploration,
  sendFeedback,
} from './explorationApi'

describe('explorationApi', () => {
  it('creates an exploration and returns a readable snapshot', async () => {
    const created = await createExploration({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost, explainable',
    })

    expect(created.code).toBe(200)
    expect(created.data.exploration.topic).toBe('AI education')

    const loaded = await getExploration(created.data.exploration.id)
    expect(loaded.code).toBe(200)
    expect(loaded.data.exploration.id).toBe(created.data.exploration.id)
  })

  it('returns 404 for an unknown exploration id', async () => {
    const missing = await getExploration('missing-exploration')

    expect(missing.code).toBe(404)
    expect(missing.msg).toContain('not found')
  })

  it('expands a stored opportunity branch', async () => {
    const created = await createExploration({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost, explainable',
    })
    const activeOpportunityId = created.data.exploration.activeOpportunityId

    const expanded = await expandOpportunityRequest(created.data.exploration.id, activeOpportunityId)

    expect(expanded.code).toBe(200)
    expect(expanded.data.exploration.nodes).toHaveLength(created.data.exploration.nodes.length)
  })

  it('updates favorites via feedback and returns the new snapshot', async () => {
    const created = await createExploration({
      topic: 'AI education',
      outputGoal: 'Research directions',
      constraints: 'Low-cost, explainable',
    })
    const firstIdeaId = 'idea-manual-favorite'

    const saved = await sendFeedback(created.data.exploration.id, {
      type: 'toggle_favorite',
      nodeId: firstIdeaId,
    })

    expect(saved.code).toBe(200)
    expect(saved.data.exploration.favorites).toContain(firstIdeaId)
  })
})
