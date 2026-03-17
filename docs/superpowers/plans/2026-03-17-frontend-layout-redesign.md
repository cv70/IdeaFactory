# Frontend Layout Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the IdeaFactory frontend from a stacked layout to a ChatGPT-style three-panel shell: collapsible left nav sidebar + main content area + right strategy panel.

**Architecture:** `App.tsx` becomes a `height: 100vh` flex-row container. A new `LeftSidebar` component handles brand, "New Exploration" button, and workspace list. A new `WorkspaceHeader` slim bar sits above the graph when a workspace is active. Explicit `viewMode` state (`'launch' | 'workspace'`) replaces the implicit `exploration !== null` condition. The right `SidebarPanel` content is unchanged.

**Tech Stack:** React 18, TypeScript, Vitest, @testing-library/react, plain CSS (no CSS frameworks)

**Spec:** `docs/superpowers/specs/2026-03-17-frontend-layout-redesign.md`

---

## File Map

| Status | Path | Responsibility |
|--------|------|----------------|
| NEW | `frontend/src/types/workspace.ts` | Shared `WorkspaceRecord` type |
| NEW | `frontend/src/components/LeftSidebar.tsx` | Collapsible left nav (brand + new exploration + workspace list + lang toggle) |
| NEW | `frontend/src/components/LeftSidebar.test.tsx` | Unit tests for LeftSidebar |
| NEW | `frontend/src/components/WorkspaceHeader.tsx` | Slim bar (workspace topic + archive button + error) |
| NEW | `frontend/src/components/WorkspaceHeader.test.tsx` | Unit tests for WorkspaceHeader |
| MODIFY | `frontend/src/lib/i18n.ts` | Add `nav.*` keys; remove dead `header.tagline/description`, `workspaces.label/title/description` |
| MODIFY | `frontend/src/App.tsx` | Three-panel JSX; `viewMode` + `sidebarCollapsed` state; `handleNewExploration`; remove auto-load effect |
| MODIFY | `frontend/src/App.css` | Layout overhaul: `appShell` → flex row, new sidebar/content/header classes, dead CSS removed |
| MODIFY | `frontend/src/index.css` | Ensure `html, body, #root { height: 100%; margin: 0 }` |
| MODIFY | `frontend/src/components/App.test.tsx` | Update workspace-switching and archive tests for new layout |
| MODIFY | `frontend/src/components/GraphView.tsx` | Remove fixed `height: 70vh`; rely on `flex: 1` from CSS |
| DELETE | `frontend/src/components/AppHeader.tsx` | Replaced by brand in LeftSidebar |
| DELETE | `frontend/src/components/WorkspaceManager.tsx` | Replaced by workspace list in LeftSidebar |

---

## Chunk 1: Foundation — Types and i18n

### Task 1: Extract WorkspaceRecord type

**Files:**
- Create: `frontend/src/types/workspace.ts`
- Modify: `frontend/src/App.tsx` (import only)

- [ ] **Step 1: Create `frontend/src/types/workspace.ts`**

```ts
export type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}
```

- [ ] **Step 2: Update `App.tsx` — swap local type for import**

At the top of `frontend/src/App.tsx`, add:
```ts
import type { WorkspaceRecord } from './types/workspace'
```

Remove the local type block (lines 33–37):
```ts
// DELETE:
type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}
```

- [ ] **Step 3: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: build succeeds (zero new errors)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/workspace.ts frontend/src/App.tsx
git commit -m "refactor: extract WorkspaceRecord to types/workspace.ts"
```

---

### Task 2: Update i18n translations

**Files:**
- Modify: `frontend/src/lib/i18n.ts`

- [ ] **Step 1: Run baseline tests before any changes**

```bash
cd frontend && npx vitest run 2>&1 | tail -20
```
Record which tests currently pass. This is the baseline to maintain.

- [ ] **Step 2: Edit `frontend/src/lib/i18n.ts`**

In the `en` block:
- Remove: `'header.tagline'`, `'header.description'`
- Remove: `'workspaces.label'`, `'workspaces.title'`, `'workspaces.description'`
- Add after `'header.langSwitch'`:
```ts
// Nav sidebar
'nav.newExploration': 'New Exploration',
'nav.recent': 'Recent',
```

In the `zh` block, same removals and add:
```ts
// Nav sidebar
'nav.newExploration': '新建探索',
'nav.recent': '最近',
```

Keys to **keep**: `header.title`, `header.langSwitch`, `workspaces.empty`, `workspaces.open`, `workspaces.archive` (still used in LeftSidebar).

- [ ] **Step 3: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: build succeeds (TypeScript will catch any code still referencing removed keys)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/i18n.ts
git commit -m "refactor: add nav.* i18n keys, remove dead workspace section / hero keys"
```

---

## Chunk 2: New Components

### Task 3: Create LeftSidebar (TDD)

**Files:**
- Create: `frontend/src/components/LeftSidebar.tsx`
- Create: `frontend/src/components/LeftSidebar.test.tsx`

- [ ] **Step 1: Write failing tests — create `frontend/src/components/LeftSidebar.test.tsx`**

```tsx
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
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
cd frontend && npx vitest run src/components/LeftSidebar.test.tsx 2>&1 | tail -10
```
Expected: FAIL — component doesn't exist yet

- [ ] **Step 3: Create `frontend/src/components/LeftSidebar.tsx`**

```tsx
import { useTranslation } from '../lib/i18n'
import type { Lang } from '../lib/i18n'
import type { WorkspaceRecord } from '../types/workspace'

type LeftSidebarProps = {
  workspaces: WorkspaceRecord[]
  activeWorkspaceId?: string
  collapsed: boolean
  loading?: boolean
  lang: Lang
  onNewExploration: () => void
  onSelectWorkspace: (id: string) => void
  onToggleCollapsed: () => void
  onSetLang: (lang: Lang) => void
}

export function LeftSidebar(props: LeftSidebarProps) {
  const { t } = useTranslation()

  return (
    <nav
      className={props.collapsed ? 'leftSidebar leftSidebarCollapsed' : 'leftSidebar'}
      aria-label="Main navigation"
    >
      <div className="sidebarBrand">
        {!props.collapsed && (
          <span className="sidebarBrandName">{t('header.title')}</span>
        )}
        <button
          type="button"
          className="sidebarIconBtn"
          aria-label={props.collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          onClick={props.onToggleCollapsed}
        >
          {props.collapsed ? '›' : '‹'}
        </button>
      </div>

      <div className="sidebarNav">
        <button
          type="button"
          className="sidebarNewBtn"
          aria-label={props.collapsed ? t('nav.newExploration') : undefined}
          onClick={props.onNewExploration}
          disabled={props.loading}
        >
          <span className="sidebarNewIcon" aria-hidden="true">+</span>
          {!props.collapsed && <span>{t('nav.newExploration')}</span>}
        </button>

        {!props.collapsed && (
          <p className="sidebarSectionLabel">{t('nav.recent')}</p>
        )}

        {props.workspaces.length === 0 && !props.collapsed ? (
          <p className="sidebarEmptyState">{t('workspaces.empty')}</p>
        ) : (
          props.workspaces.map((workspace) => (
            <button
              key={workspace.id}
              type="button"
              className={
                workspace.id === props.activeWorkspaceId
                  ? 'sidebarItem sidebarItemActive'
                  : 'sidebarItem'
              }
              aria-label={`Open workspace ${workspace.topic}`}
              title={props.collapsed ? workspace.topic : undefined}
              onClick={() => props.onSelectWorkspace(workspace.id)}
              disabled={props.loading}
            >
              <span className="sidebarItemDot" aria-hidden="true">
                {workspace.topic.charAt(0).toUpperCase()}
              </span>
              {!props.collapsed && (
                <span className="sidebarItemText">{workspace.topic}</span>
              )}
            </button>
          ))
        )}
      </div>

      <div className="sidebarBottom">
        <button
          type="button"
          className="sidebarIconBtn"
          aria-label="Switch language"
          onClick={() => props.onSetLang(props.lang === 'en' ? 'zh' : 'en')}
        >
          {props.lang === 'en' ? '中文' : 'English'}
        </button>
      </div>
    </nav>
  )
}
```

- [ ] **Step 4: Run tests — confirm PASS**

```bash
cd frontend && npx vitest run src/components/LeftSidebar.test.tsx 2>&1 | tail -20
```
Expected: All 11 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/LeftSidebar.tsx frontend/src/components/LeftSidebar.test.tsx
git commit -m "feat: add LeftSidebar component with tests"
```

---

### Task 4: Create WorkspaceHeader (TDD)

**Files:**
- Create: `frontend/src/components/WorkspaceHeader.tsx`
- Create: `frontend/src/components/WorkspaceHeader.test.tsx`

- [ ] **Step 1: Write failing tests — create `frontend/src/components/WorkspaceHeader.test.tsx`**

```tsx
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
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
cd frontend && npx vitest run src/components/WorkspaceHeader.test.tsx 2>&1 | tail -10
```
Expected: FAIL

- [ ] **Step 3: Create `frontend/src/components/WorkspaceHeader.tsx`**

```tsx
type WorkspaceHeaderProps = {
  topic: string
  loading?: boolean
  error?: string
  onArchive: () => void
}

export function WorkspaceHeader(props: WorkspaceHeaderProps) {
  return (
    <div className="workspaceHeader">
      <span className="workspaceTitle">{props.topic}</span>
      <div className="workspaceHeaderActions">
        {props.error && (
          <span className="workspaceError">{props.error}</span>
        )}
        <button
          type="button"
          className="miniAction"
          onClick={props.onArchive}
          disabled={props.loading}
        >
          Archive
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Run tests — confirm PASS**

```bash
cd frontend && npx vitest run src/components/WorkspaceHeader.test.tsx 2>&1 | tail -10
```
Expected: All 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/WorkspaceHeader.tsx frontend/src/components/WorkspaceHeader.test.tsx
git commit -m "feat: add WorkspaceHeader component with tests"
```

---

## Chunk 3: App.tsx Restructure and CSS Overhaul

### Task 5: Restructure App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Update imports**

Remove from top of `App.tsx`:
```ts
import { AppHeader } from './components/AppHeader'
import { WorkspaceManager } from './components/WorkspaceManager'
```

Add (the `WorkspaceRecord` import was already added in Task 1 — do not add it again):
```ts
import { LeftSidebar } from './components/LeftSidebar'
import { WorkspaceHeader } from './components/WorkspaceHeader'
```

- [ ] **Step 2: Add new state**

After the existing state declarations, add:
```ts
const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
const [viewMode, setViewMode] = useState<'launch' | 'workspace'>('launch')
```

- [ ] **Step 3: Add handleNewExploration**

After `handleExampleSelect`, add:
```ts
function handleNewExploration() {
  setViewMode('launch')
  setSidebarCollapsed(false)
  setError('')
}
```

- [ ] **Step 4: Update handleSubmit**

In `handleSubmit`, after `setLoading(false)` at the end of the success path, add:
```ts
setViewMode('workspace')
```

The last few lines of the success path should look like:
```ts
setExploration(response.data.exploration)
upsertWorkspaceHistory({
  id: response.data.exploration.id,
  topic: response.data.exploration.topic,
  updatedAt: Date.now(),
})
setStrategyHistory([])
mutationCursorRef.current = ''
setSelectedNodeId(response.data.exploration.activeOpportunityId)
setLoading(false)
setViewMode('workspace')  // add this line
```

- [ ] **Step 5: Update handleSelectWorkspace**

After `setLoading(false)` at the end of `handleSelectWorkspace`, add:
```ts
setViewMode('workspace')
```

(`setError('')` is already called near the top of this function.)

- [ ] **Step 6: Update handleArchiveWorkspace**

Add `setError('')` at the start of the function (alongside the existing `setLoading(true)`), and add `setViewMode('launch')` inside the active-workspace block:

```ts
async function handleArchiveWorkspace(workspaceId: string) {
  setLoading(true)
  setError('')    // add this line
  const ok = await archiveWorkspace(workspaceId)
  // ...
  if (exploration?.id === workspaceId) {
    setExploration(null)
    setSelectedNodeId(null)
    setStrategyHistory([])
    setViewMode('launch')  // add this line
  }
  // ...
}
```

- [ ] **Step 7: Remove the auto-load useEffect**

Delete these lines entirely from `App.tsx`:
```ts
useEffect(() => {
  if (exploration || workspaceHistory.length === 0) return
  const latest = workspaceHistory[0]
  void handleSelectWorkspace(latest.id)
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [workspaceHistory, exploration])
```

- [ ] **Step 8: Replace the JSX return**

Replace the entire `return (...)` block with:
```tsx
return (
  <LangContext.Provider value={{ lang, setLang, t: makeT(lang) }}>
    <div className="appShell">
      <LeftSidebar
        workspaces={workspaceHistory}
        activeWorkspaceId={exploration?.id}
        collapsed={sidebarCollapsed}
        loading={loading}
        lang={lang}
        onNewExploration={handleNewExploration}
        onSelectWorkspace={handleSelectWorkspace}
        onToggleCollapsed={() => setSidebarCollapsed((c) => !c)}
        onSetLang={setLang}
      />

      <main className="mainContent">
        {viewMode === 'launch' ? (
          <div className="launchCentered">
            <LaunchPanel
              topic={topic}
              outputGoal={outputGoal}
              constraints={constraints}
              loading={loading}
              examples={EXAMPLE_TOPICS}
              onTopicChange={setTopic}
              onOutputGoalChange={setOutputGoal}
              onConstraintsChange={setConstraints}
              onExampleSelect={handleExampleSelect}
              onSubmit={handleSubmit}
            />
            {error ? <p className="errorBanner">{error}</p> : null}
          </div>
        ) : (
          <>
            <WorkspaceHeader
              topic={exploration?.topic ?? ''}
              loading={loading}
              error={error || undefined}
              onArchive={() => exploration && void handleArchiveWorkspace(exploration.id)}
            />
            <GraphView
              session={exploration!}
              selectedNodeId={selectedNodeId}
              onSelectNode={(node) => setSelectedNodeId(node?.id ?? null)}
              onExpandOpportunity={handleExpandOpportunity}
            />
          </>
        )}
      </main>

      {view ? (
        <SidebarPanel
          savedIdeas={savedIdeas}
          view={view}
          strategy={exploration?.strategy}
          strategyBusy={strategyUpdating}
          onUpdateStrategy={handleUpdateStrategy}
          strategyHistory={strategyHistory}
          onRollbackStrategy={handleRollbackStrategy}
          onToggleFavorite={handleToggleFavorite}
          onSubmitIntervention={handleSubmitIntervention}
          lastInterventionIntent={lastInterventionIntent}
          lastInterventionStatus={lastInterventionStatus}
        />
      ) : null}
    </div>
  </LangContext.Provider>
)
```

- [ ] **Step 9: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -20
```
Expected: build succeeds (AppHeader and WorkspaceManager still exist on disk at this point)

- [ ] **Step 10: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat: restructure App.tsx — three-panel layout, viewMode state, handleNewExploration"
```

---

### Task 6: CSS Overhaul

**Files:**
- Modify: `frontend/src/index.css`
- Modify: `frontend/src/App.css`

- [ ] **Step 1: Update `frontend/src/index.css`**

The current file has conflicting rules that must be replaced (not appended):

Replace the existing `html` block:
```css
/* REPLACE: */
html {
  min-height: 100%;
}

/* WITH: */
html {
  height: 100%;
}
```

Replace the existing `body` block (keep `min-width: 320px` and `body::before`):
```css
/* REPLACE: */
body {
  margin: 0;
  min-width: 320px;
  min-height: 100vh;
}

/* WITH: */
body {
  margin: 0;
  min-width: 320px;
  height: 100%;
}
```

Replace the existing `#root` block:
```css
/* REPLACE: */
#root {
  min-height: 100vh;
  padding: 1.25rem 1rem;
}

/* WITH: */
#root {
  height: 100%;
}
```

- [ ] **Step 2: Rewrite `.appShell` in `frontend/src/App.css`**

Replace:
```css
.appShell {
  width: min(1520px, 100%);
  margin: 0 auto;
  display: grid;
  gap: 1.5rem;
}
```

With:
```css
.appShell {
  display: flex;
  flex-direction: row;
  height: 100vh;
  overflow: hidden;
  width: 100%;
}
```

- [ ] **Step 3: Remove `.appGrid` and responsive grid blocks**

Delete the entire block:
```css
.appGrid {
  display: grid;
  gap: 1.25rem;
  align-items: start;
}

.mainColumn {
  display: grid;
  gap: 1.25rem;
}

@media (min-width: 860px) {
  .appGrid {
    grid-template-columns: minmax(0, 1fr) 300px;
  }

  .sidebarPanel {
    position: sticky;
    top: 1.25rem;
    max-height: calc(100vh - 2.5rem);
    overflow-y: auto;
    scrollbar-width: thin;
    scrollbar-color: rgba(23, 42, 92, 0.15) transparent;
  }
}

@media (min-width: 1100px) {
  .appGrid {
    grid-template-columns: minmax(0, 1fr) 320px;
  }

  .workbench {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
```

- [ ] **Step 4: Add `.mainContent` and `.launchCentered`**

```css
.mainContent {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.launchCentered {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  overflow-y: auto;
  gap: 1rem;
}
```

- [ ] **Step 5: Update `.graphContainer`**

Replace:
```css
.graphContainer {
  position: relative;
  height: 70vh;
  min-height: 480px;
  border: 1px solid var(--panel-border);
  border-radius: 24px;
  background: var(--panel-bg);
  box-shadow: var(--panel-shadow);
  backdrop-filter: blur(20px);
  overflow: hidden;
}
```

With:
```css
.graphContainer {
  position: relative;
  flex: 1;
  min-height: 0;
  background: var(--panel-bg);
  overflow: hidden;
}
```

Also remove the now-orphaned child rule that referenced the old border-radius:
```css
/* DELETE this block: */
.graphContainer .react-flow {
  width: 100%;
  height: 100%;
  border-radius: 24px;
}
```

- [ ] **Step 6: Update `.sidebarPanel`**

The existing `App.css` has a plain `.sidebarPanel { display: grid; gap: 1.2rem }` rule (separate from the now-deleted `@media` block). **Replace** that existing rule with:
```css
.sidebarPanel {
  width: 320px;
  flex-shrink: 0;
  height: 100vh;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(23, 42, 92, 0.15) transparent;
  border-left: 1px solid var(--panel-border);
  display: grid;
  gap: 1.2rem;
  padding: 1.4rem;
}
```

Do **not** add a second `.sidebarPanel` rule — replace the existing one in place.

- [ ] **Step 7: Remove dead workspace card CSS**

Delete these blocks (lines ~370–402 in original file):
```css
.workspaceList { ... }
.workspaceCard { ... }
.workspaceCard:hover { ... }
.workspaceCardActive { ... }
.workspaceActions { ... }
```

- [ ] **Step 8: Add LeftSidebar CSS**

Append to `App.css`:
```css
/* ─── Left Sidebar ─────────────────────────────────────────────── */

.leftSidebar {
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  height: 100vh;
  border-right: 1px solid var(--panel-border);
  background: var(--panel-bg);
  backdrop-filter: blur(20px);
  overflow: hidden;
  transition: width 200ms ease;
}

.leftSidebarCollapsed {
  width: 56px;
}

.sidebarBrand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-bottom: 1px solid var(--panel-border);
  flex-shrink: 0;
  gap: 0.5rem;
  min-height: 56px;
}

.sidebarBrandName {
  font-size: 1rem;
  font-weight: 700;
  color: var(--main-ink);
  white-space: nowrap;
  overflow: hidden;
}

.sidebarNav {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 0.75rem 0.5rem;
  gap: 0.2rem;
  scrollbar-width: thin;
  scrollbar-color: rgba(23, 42, 92, 0.1) transparent;
}

.sidebarNewBtn {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 0.75rem;
  border-radius: 10px;
  background: linear-gradient(135deg, #101a3f, #2851b0);
  color: #fdf7ec;
  font: inherit;
  font-weight: 700;
  font-size: 0.88rem;
  border: none;
  cursor: pointer;
  width: 100%;
  margin-bottom: 0.5rem;
  white-space: nowrap;
  overflow: hidden;
  transition: opacity 150ms ease;
}

.sidebarNewBtn:hover:not(:disabled) {
  opacity: 0.88;
  transform: none;
}

.sidebarNewIcon {
  flex-shrink: 0;
  font-size: 1.1rem;
  line-height: 1;
}

.sidebarSectionLabel {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--accent-ink);
  padding: 0.4rem 0.5rem 0.2rem;
  margin: 0;
  white-space: nowrap;
}

.sidebarItem {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.5rem 0.6rem;
  border-radius: 8px;
  border: none;
  background: transparent;
  cursor: pointer;
  text-align: left;
  width: 100%;
  font: inherit;
  font-size: 0.875rem;
  color: var(--main-ink);
  transition: background 150ms ease;
  white-space: nowrap;
  overflow: hidden;
}

.sidebarItem:hover:not(:disabled) {
  background: rgba(23, 42, 92, 0.06);
  transform: none;
}

.sidebarItemActive {
  background: rgba(34, 79, 186, 0.1);
  color: #2851b0;
  font-weight: 600;
}

.sidebarItemDot {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: rgba(40, 81, 176, 0.12);
  color: #2851b0;
  font-size: 0.75rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebarItemText {
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.sidebarEmptyState {
  font-size: 0.82rem;
  color: var(--muted-ink);
  font-style: italic;
  padding: 0.35rem 0.5rem;
  white-space: nowrap;
  overflow: hidden;
  margin: 0;
}

.sidebarBottom {
  flex-shrink: 0;
  padding: 0.75rem 0.5rem;
  border-top: 1px solid var(--panel-border);
}

.sidebarIconBtn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.45rem 0.6rem;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 0.85rem;
  color: var(--muted-ink);
  transition: background 150ms ease;
  white-space: nowrap;
  width: 100%;
}

.sidebarIconBtn:hover {
  background: rgba(23, 42, 92, 0.07);
  transform: none;
}

/* ─── Workspace Header ──────────────────────────────────────────── */

.workspaceHeader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1rem;
  height: 48px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--panel-border);
  background: var(--panel-bg);
  gap: 1rem;
}

.workspaceTitle {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--main-ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.workspaceHeaderActions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

.workspaceError {
  font-size: 0.8rem;
  color: #8f2f17;
}
```

- [ ] **Step 9: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: build succeeds

- [ ] **Step 10: Commit**

```bash
git add frontend/src/App.css frontend/src/index.css
git commit -m "feat: overhaul CSS for three-panel layout — leftSidebar, workspaceHeader, mainContent"
```

---

## Chunk 4: Tests, Minor Updates, and Cleanup

### Task 7: Update App.test.tsx

**Files:**
- Modify: `frontend/src/components/App.test.tsx`

- [ ] **Step 1: Run current App tests**

```bash
cd frontend && npx vitest run src/components/App.test.tsx 2>&1
```
Note which tests fail and why before making changes.

- [ ] **Step 2: Update 'switches between historical workspaces' test**

The test currently submits two forms back-to-back. In the new layout, after the first workspace loads, the LaunchPanel is hidden — the user must click "New Exploration" to return to it.

**Before editing:** The baseline run from Step 1 will show which language the tests use. The existing tests use `'Topic'`, `'Start exploration'` (English) — if these pass, `App.tsx`'s `lang` defaults to `'en'` in the test environment. Use English throughout. If they fail with "unable to find label", switch to `'主题'`, `'开始探索'`, `'新建探索'`.

Replace the test body (the test named `'switches between historical workspaces'`):

```tsx
it('switches between historical workspaces', async () => {
  render(<App />)

  // Start first workspace
  fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
  fireEvent.change(screen.getByLabelText('Output goal'), { target: { value: 'Research directions' } })
  fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))
  expect((await screen.findAllByText(/Learning friction for AI education/)).length).toBeGreaterThan(0)

  // Navigate back to LaunchPanel to start a second workspace
  fireEvent.click(screen.getByRole('button', { name: 'New Exploration' }))

  // Start second workspace
  fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'Climate fintech' } })
  fireEvent.change(screen.getByLabelText('Output goal'), { target: { value: 'Venture opportunities' } })
  fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))
  expect((await screen.findAllByText(/Learning friction for Climate fintech/)).length).toBeGreaterThan(0)

  // Switch back to first workspace via sidebar
  fireEvent.click(screen.getByRole('button', { name: 'Open workspace AI education' }))
  await waitFor(() => {
    expect(screen.getAllByText(/Learning friction for AI education/).length).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 3: Update 'archives workspace from manager list' test**

The Archive button is now in WorkspaceHeader (shown when a workspace is active). Rename and keep the test body the same — the button accessible name 'Archive' is unchanged, but it only appears after a workspace loads:

```tsx
it('archives active workspace from workspace header', async () => {
  vi.spyOn(explorationApi, 'archiveWorkspace').mockResolvedValueOnce(true)

  render(<App />)

  fireEvent.change(screen.getByLabelText('Topic'), { target: { value: 'AI education' } })
  fireEvent.change(screen.getByLabelText('Output goal'), { target: { value: 'Research directions' } })
  fireEvent.click(screen.getByRole('button', { name: 'Start exploration' }))

  expect(await screen.findByRole('button', { name: 'Archive' })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Archive' }))

  await waitFor(() => {
    expect(screen.getByText('No historical workspaces yet.')).toBeInTheDocument()
  })
})
```

- [ ] **Step 4: Run all App tests**

```bash
cd frontend && npx vitest run src/components/App.test.tsx 2>&1
```
Expected: All pass. Fix any remaining failures by adapting text queries to match the actual rendered lang.

- [ ] **Step 5: Run full test suite**

```bash
cd frontend && npx vitest run 2>&1
```
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/App.test.tsx
git commit -m "test: update App.test.tsx for three-panel layout"
```

---

### Task 8: Remove GraphView fixed height

**Files:**
- Modify: `frontend/src/components/GraphView.tsx`

The `.graphContainer` CSS was already changed to `flex: 1; min-height: 0` in Task 6. The GraphView JSX just needs to use that class (it already does). Verify there is no inline `style={{ height }}` or `height` prop hardcoded in the GraphView component.

- [ ] **Step 1: Check GraphView for hardcoded height**

Read `frontend/src/components/GraphView.tsx` and search for any `height: 70vh` or fixed height in inline styles or JSX. If found, remove it. The `.graphContainer` CSS rule now handles sizing.

- [ ] **Step 2: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: success

> **Note:** If `GraphView.tsx` has no inline height (confirmed by reviewer), this task is a verification-only step — no code changes needed. Proceed to commit if no changes were made, or skip the commit step.

- [ ] **Step 3: Commit (only if changes were made)**

```bash
git add frontend/src/components/GraphView.tsx
git commit -m "refactor: remove hardcoded height from GraphView — rely on flex layout"
```

---

### Task 9: Delete AppHeader and WorkspaceManager

**Files:**
- Delete: `frontend/src/components/AppHeader.tsx`
- Delete: `frontend/src/components/WorkspaceManager.tsx`

- [ ] **Step 1: Delete the files**

```bash
rm frontend/src/components/AppHeader.tsx frontend/src/components/WorkspaceManager.tsx
```

- [ ] **Step 2: Build to confirm no dangling imports**

```bash
cd frontend && npm run build 2>&1 | tail -20
```
Expected: build succeeds. If import errors appear, find and remove the stray imports.

- [ ] **Step 3: Lint check**

```bash
cd frontend && npm run lint 2>&1 | tail -20
```
Fix any lint errors.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: delete AppHeader and WorkspaceManager — replaced by LeftSidebar"
```

---

### Task 10: Final verification

- [ ] **Step 1: Run full test suite**

```bash
cd frontend && npx vitest run 2>&1
```
Expected: All tests pass.

- [ ] **Step 2: Build and lint**

```bash
cd frontend && npm run build && npm run lint 2>&1 | tail -10
```
Expected: Clean build and zero lint errors.

- [ ] **Step 3: Manual smoke test**

Start the dev server:
```bash
cd frontend && npm run dev
```
Open http://localhost:5173 and verify:
- [ ] App opens to LaunchPanel centered in main area (no graph)
- [ ] Left sidebar shows brand name (Idea Factory / 创意工厂)
- [ ] Left sidebar shows "New Exploration" / "新建探索" button at top
- [ ] Left sidebar shows "No historical workspaces yet." empty state
- [ ] Submitting the form switches to GraphView with WorkspaceHeader above it
- [ ] WorkspaceHeader shows the workspace topic text and Archive button
- [ ] Workspace item appears in left sidebar, highlighted as active
- [ ] Clicking "New Exploration" returns to LaunchPanel; sidebar expands if collapsed
- [ ] Clicking a workspace item in the sidebar loads that workspace's graph
- [ ] Clicking Archive removes the workspace and returns to LaunchPanel
- [ ] Sidebar collapse button toggles between 240px (full) and 56px (icon-only)
- [ ] In collapsed state, workspace items show first-letter icons with topic tooltips
- [ ] Language toggle in sidebar bottom switches between 中文 and English
- [ ] Right strategy panel (SidebarPanel) is visible only when a workspace is active

- [ ] **Step 4: Final commit if any fixes were made**

```bash
git add -A
git commit -m "feat: complete three-panel layout redesign — LeftSidebar, WorkspaceHeader, viewMode"
```
