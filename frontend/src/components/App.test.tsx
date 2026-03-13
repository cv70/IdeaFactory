import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from '../App'
import * as explorationApi from '../lib/explorationApi'

describe('idea factory app', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('starts with a launch panel', () => {
    render(<App />)

    expect(screen.getByText('Idea Factory')).toBeInTheDocument()
    expect(screen.getByLabelText('Topic')).toBeInTheDocument()
    expect(screen.getByLabelText('Output goal')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Start exploration' })).toBeInTheDocument()
  })

  it('renders the workbench after submitting a topic', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), { target: { value: 'Research directions' } })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    expect(await screen.findByText('Opportunity map')).toBeInTheDocument()
    expect(screen.getByText('Question trail')).toBeInTheDocument()
    expect(screen.getByText('Materialized ideas')).toBeInTheDocument()
  })

  it('updates the middle column when a different branch is selected', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    const branchButton = await screen.findByRole('button', {
      name: /Adoption wedge for AI education/,
    })
    fireEvent.click(branchButton)

    expect(
      screen.getByText('What narrow entry point would make the topic immediately usable?'),
    ).toBeInTheDocument()
  })

  it('adds favorited ideas to the sidebar', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    fireEvent.click((await screen.findAllByRole('button', { name: 'Save idea' }))[0])

    await waitFor(() => {
      expect(screen.getByText(/Saved ideas \(1\)/)).toBeInTheDocument()
    })
  })

  it('shows runtime strategy controls after exploration starts', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    expect(await screen.findByText('Runtime strategy')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Apply strategy' })).toBeInTheDocument()
    expect(screen.getByText('Strategy history')).toBeInTheDocument()
  })

  it('switches between historical workspaces', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))
    expect(await screen.findByText(/Learning friction for AI education/)).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'Climate fintech' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Venture opportunities' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))
    expect(await screen.findByText(/Learning friction for Climate fintech/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open workspace AI education' }))
    await waitFor(() => {
      expect(screen.getByText(/Learning friction for AI education/)).toBeInTheDocument()
    })
  })

  it('archives workspace from manager list', async () => {
    vi.spyOn(explorationApi, 'archiveWorkspace').mockResolvedValueOnce(true)

    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    expect(await screen.findByRole('button', { name: 'Archive' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Archive' }))

    await waitFor(() => {
      expect(screen.getByText('No historical workspaces yet.')).toBeInTheDocument()
    })
  })

  it('shows an error message when exploration creation fails', async () => {
    vi.spyOn(explorationApi, 'createExploration').mockResolvedValueOnce({
      code: 500,
      msg: 'Mock API failure',
      data: {
        exploration: {
          id: '',
          topic: '',
          outputGoal: '',
          constraints: '',
          activeOpportunityId: '',
          nodes: [],
          edges: [],
          favorites: [],
          runs: [],
        },
        presentation: {
          opportunities: [],
          activeOpportunity: undefined as never,
          questionTrail: [],
          hypothesisTrail: [],
          ideaCards: [],
          savedIdeas: [],
          runNotes: [],
        },
      },
    })

    render(<App />)

    fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('Output goal'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

    expect(await screen.findByText('Mock API failure')).toBeInTheDocument()
  })
})
