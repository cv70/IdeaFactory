import { describe, expect, it } from 'vitest'
import {
  buildWorkbenchView,
  createExplorationSession,
  expandOpportunity,
  toggleFavoriteIdea,
} from './workbench'

describe('workbench domain helpers', () => {
  it('creates an exploration session from user input', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    expect(session.topic).toBe('AI education')
    expect(session.outputGoal).toBe('research directions')
    expect(session.constraints).toBe('low-cost, testable')
    expect(session.nodes.some((node) => node.type === 'opportunity')).toBe(true)
  })

  it('builds a workbench view around the active opportunity branch', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    const target = session.nodes.find((node) => node.type === 'opportunity' && node.id !== session.activeOpportunityId)
    expect(target).toBeDefined()

    const view = buildWorkbenchView({
      ...session,
      activeOpportunityId: target!.id,
    })

    expect(view.activeOpportunity.id).toBe(target!.id)
    expect(view.questionTrail.length).toBeGreaterThan(0)
    expect(view.ideaCards.length).toBeGreaterThan(0)
  })

  it('expands an opportunity branch with additional ideas', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    const initialView = buildWorkbenchView(session)
    const expanded = expandOpportunity(session, session.activeOpportunityId)
    const nextView = buildWorkbenchView(expanded)

    expect(nextView.ideaCards.length).toBeGreaterThan(initialView.ideaCards.length)
    expect(expanded.runs.at(-1)?.focus).toContain(session.activeOpportunityId)
  })

  it('toggles favorite ideas without breaking the workbench state', () => {
    const session = createExplorationSession({
      topic: 'AI education',
      outputGoal: 'research directions',
      constraints: 'low-cost, testable',
    })

    const ideaId = buildWorkbenchView(session).ideaCards[0].id

    const saved = toggleFavoriteIdea(session, ideaId)
    expect(saved.favorites).toContain(ideaId)

    const unsaved = toggleFavoriteIdea(saved, ideaId)
    expect(unsaved.favorites).not.toContain(ideaId)
  })
})
