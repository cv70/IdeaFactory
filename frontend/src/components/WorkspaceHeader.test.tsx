import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { LangContext, makeT } from '../lib/i18n'
import { WorkspaceHeader } from './WorkspaceHeader'

function renderHeader(overrides: Partial<React.ComponentProps<typeof WorkspaceHeader>> = {}) {
  const props = {
    topic: 'AI education',
    loading: false,
    error: undefined as string | undefined,
    onArchive: vi.fn(),
    workspaceStatus: 'active' as const,
    onTogglePause: vi.fn(),
    ...overrides,
  }
  return {
    ...render(
      <LangContext.Provider value={{ lang: 'en', setLang: vi.fn(), t: makeT('en') }}>
        <WorkspaceHeader {...props} />
      </LangContext.Provider>
    ),
    props,
  }
}

describe('WorkspaceHeader', () => {
  it('renders workspace topic', () => {
    renderHeader()
    expect(screen.getByText('AI education')).toBeInTheDocument()
  })

  it('renders archive button', () => {
    renderHeader()
    expect(screen.getByRole('button', { name: 'Archive' })).toBeInTheDocument()
  })

  it('calls onArchive when archive button is clicked', () => {
    const { props } = renderHeader()
    fireEvent.click(screen.getByRole('button', { name: 'Archive' }))
    expect(props.onArchive).toHaveBeenCalledOnce()
  })

  it('disables archive button when loading', () => {
    renderHeader({ loading: true })
    expect(screen.getByRole('button', { name: 'Archive' })).toBeDisabled()
  })

  it('shows pause button when workspace is active', () => {
    renderHeader({ workspaceStatus: 'active' })
    expect(screen.getByRole('button', { name: 'Pause' })).toBeInTheDocument()
  })

  it('shows resume button when workspace is paused', () => {
    renderHeader({ workspaceStatus: 'paused' })
    expect(screen.getByRole('button', { name: 'Resume' })).toBeInTheDocument()
  })

  it('calls onTogglePause when pause button is clicked', () => {
    const { props } = renderHeader({ workspaceStatus: 'active' })
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    expect(props.onTogglePause).toHaveBeenCalledOnce()
  })

  it('shows error message when provided', () => {
    renderHeader({ error: 'Something went wrong' })
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })
})
