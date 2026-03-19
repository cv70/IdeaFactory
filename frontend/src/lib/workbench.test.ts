import { describe, expect, it } from 'vitest'
import {
  buildWorkbenchView,
  createExplorationSession,
  expandOpportunity,
  toggleFavoriteIdea,
} from './workbench'

describe('workbench domain helpers', () => {
  it('creates an exploration session with only a topic anchor', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    expect(session.topic).toBe('AI education')
    expect(session.outputGoal).toBe('research directions')
    expect(session.constraints).toBe('low-cost, testable')
    expect(session.nodes).toHaveLength(1)
    expect(session.nodes[0]?.type).toBe('topic')
    expect(session.edges).toHaveLength(0)
    expect(session.activeOpportunityId).toBe(session.nodes[0]?.id)
  })

  it('returns null workbench view until opportunities are created', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    expect(buildWorkbenchView(session)).toBeNull()
  })

  it('expands an opportunity branch with additional ideas', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    const expanded = expandOpportunity(session, session.activeOpportunityId)
    expect(expanded).toEqual(session)
  })

  it('toggles favorite ideas without breaking the workbench state', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    const ideaId = 'idea-manual-favorite'

    const saved = toggleFavoriteIdea(session, ideaId)
    expect(saved.favorites).toContain(ideaId)

    const unsaved = toggleFavoriteIdea(saved, ideaId)
    expect(unsaved.favorites).not.toContain(ideaId)
  })
})
