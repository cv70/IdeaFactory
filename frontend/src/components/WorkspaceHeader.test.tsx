import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { WorkspaceHeader } from './WorkspaceHeader'

function renderHeader(overrides: Partial<React.ComponentProps<typeof WorkspaceHeader>> = {}) {
  const props = {
    topic: 'AI education',
    loading: false,
    error: undefined as string | undefined,
    onArchive: vi.fn(),
    ...overrides,
  }
  return { ...render(<WorkspaceHeader {...props} />), props }
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

  it('shows error message when provided', () => {
    renderHeader({ error: 'Something went wrong' })
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })
})
