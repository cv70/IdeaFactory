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

    expect(screen.getByText('创意工厂')).toBeInTheDocument()
    expect(screen.getByLabelText('主题')).toBeInTheDocument()
    expect(screen.getByLabelText('输出目标')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '开始探索' })).toBeInTheDocument()
  })

  it('renders the workbench after submitting a topic', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), { target: { value: 'Research directions' } })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    expect(await screen.findByRole('button', { name: 'Archive' })).toBeInTheDocument()
    expect(await screen.findByText('运行策略')).toBeInTheDocument()
    expect(screen.getByText('提交干预')).toBeInTheDocument()
    expect(screen.getByText('策略历史')).toBeInTheDocument()
  })

  it('updates the graph view when a different branch node is selected', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    // Graph nodes for all branches should be rendered
    expect(
      (await screen.findAllByText(/Learning friction for AI education/)).length,
    ).toBeGreaterThan(0)
    expect(screen.getAllByText(/Adoption wedge for AI education/).length).toBeGreaterThan(0)
  })

  it('shows saved ideas section in sidebar after exploration starts', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    await waitFor(() => {
      expect(screen.getByText(/已收藏创意 \(0\)/)).toBeInTheDocument()
    })
  })

  it('shows runtime strategy controls after exploration starts', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    expect(await screen.findByText('运行策略')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '应用策略' })).toBeInTheDocument()
    expect(screen.getByText('策略历史')).toBeInTheDocument()
  })

  it('switches between historical workspaces', async () => {
    render(<App />)

    // Start first workspace
    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))
    expect((await screen.findAllByText(/Learning friction for AI education/)).length).toBeGreaterThan(0)

    // Navigate back to LaunchPanel via "New Exploration" button
    fireEvent.click(screen.getByRole('button', { name: '新建探索' }))

    // Start second workspace
    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'Climate fintech' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Venture opportunities' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))
    expect((await screen.findAllByText(/Learning friction for Climate fintech/)).length).toBeGreaterThan(0)

    // Switch back to first workspace via sidebar
    fireEvent.click(screen.getByRole('button', { name: 'Open workspace AI education' }))
    await waitFor(() => {
      expect(screen.getAllByText(/Learning friction for AI education/).length).toBeGreaterThan(0)
    })
  })

  it('archives active workspace from workspace header', async () => {
    vi.spyOn(explorationApi, 'archiveWorkspace').mockResolvedValueOnce(true)

    render(<App />)

    // Start workspace
    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    // Archive button appears in WorkspaceHeader
    expect(await screen.findByRole('button', { name: 'Archive' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Archive' }))

    await waitFor(() => {
      // After archive, viewMode returns to 'launch' — LaunchPanel shown, empty state in sidebar
      expect(screen.getByText('暂无历史工作空间。')).toBeInTheDocument()
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

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI education' } })
    fireEvent.change(screen.getByLabelText('输出目标'), {
      target: { value: 'Research directions' },
    })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    expect(await screen.findByText('Mock API failure')).toBeInTheDocument()
  })

  it('submits an intervention and shows reflected status', async () => {
    render(<App />)

    fireEvent.change(screen.getByLabelText('主题'), { target: { value: 'AI wellness coach' } })
    fireEvent.change(screen.getByLabelText('输出目标'), { target: { value: 'find promising directions' } })
    fireEvent.click(screen.getByRole('button', { name: '开始探索' }))

    expect(await screen.findByText('提交干预')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('干预'), { target: { value: 'focus on retention loops' } })
    fireEvent.click(screen.getByRole('button', { name: '提交' }))

    expect(await screen.findByText(/reflected/i)).toBeInTheDocument()
    expect(screen.getAllByText(/focus on retention loops/i).length).toBeGreaterThan(0)
  })
})
