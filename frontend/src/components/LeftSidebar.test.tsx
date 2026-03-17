import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { LangContext, makeT } from '../lib/i18n'
import { LeftSidebar } from './LeftSidebar'

const workspaces = [
  { id: 'ws-1', topic: 'AI education', updatedAt: 1700000000000 },
  { id: 'ws-2', topic: 'Climate fintech', updatedAt: 1700000001000 },
]

function renderSidebar(overrides: Partial<React.ComponentProps<typeof LeftSidebar>> = {}) {
  const props = {
    workspaces,
    activeWorkspaceId: undefined as string | undefined,
    collapsed: false,
    loading: false,
    lang: 'en' as const,
    onNewExploration: vi.fn(),
    onSelectWorkspace: vi.fn(),
    onToggleCollapsed: vi.fn(),
    onSetLang: vi.fn(),
    ...overrides,
  }
  return {
    ...render(
      <LangContext.Provider value={{ lang: 'en', setLang: vi.fn(), t: makeT('en') }}>
        <LeftSidebar {...props} />
      </LangContext.Provider>
    ),
    props,
  }
}

describe('LeftSidebar', () => {
  it('renders brand name when expanded', () => {
    renderSidebar()
    expect(screen.getByText('Idea Factory')).toBeInTheDocument()
  })

  it('renders new exploration button', () => {
    renderSidebar()
    expect(screen.getByRole('button', { name: 'New Exploration' })).toBeInTheDocument()
  })

  it('renders workspace items with accessible names', () => {
    renderSidebar()
    expect(screen.getByRole('button', { name: 'Open workspace AI education' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open workspace Climate fintech' })).toBeInTheDocument()
  })

  it('highlights active workspace', () => {
    renderSidebar({ activeWorkspaceId: 'ws-1' })
    const btn = screen.getByRole('button', { name: 'Open workspace AI education' })
    expect(btn.className).toContain('sidebarItemActive')
  })

  it('calls onSelectWorkspace with workspace id when item clicked', () => {
    const { props } = renderSidebar()
    fireEvent.click(screen.getByRole('button', { name: 'Open workspace AI education' }))
    expect(props.onSelectWorkspace).toHaveBeenCalledWith('ws-1')
  })

  it('calls onNewExploration when New Exploration button clicked', () => {
    const { props } = renderSidebar()
    fireEvent.click(screen.getByRole('button', { name: 'New Exploration' }))
    expect(props.onNewExploration).toHaveBeenCalledOnce()
  })

  it('calls onToggleCollapsed when collapse button clicked', () => {
    const { props } = renderSidebar()
    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    expect(props.onToggleCollapsed).toHaveBeenCalledOnce()
  })

  it('calls onSetLang with toggled lang when lang button clicked', () => {
    const { props } = renderSidebar({ lang: 'en' })
    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }))
    expect(props.onSetLang).toHaveBeenCalledWith('zh')
  })

  it('shows empty state text when no workspaces', () => {
    renderSidebar({ workspaces: [] })
    expect(screen.getByText('No historical workspaces yet.')).toBeInTheDocument()
  })

  it('hides brand name in collapsed state', () => {
    renderSidebar({ collapsed: true })
    expect(screen.queryByText('Idea Factory')).not.toBeInTheDocument()
  })

  it('shows expand button in collapsed state', () => {
    renderSidebar({ collapsed: true })
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument()
  })
})
