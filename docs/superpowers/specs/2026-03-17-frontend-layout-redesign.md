# Frontend Layout Redesign — ChatGPT/DeepSeek Style

**Date:** 2026-03-17
**Status:** Approved

## Goal

Redesign the IdeaFactory frontend layout from a stacked single-column + right-sidebar structure to a three-panel layout inspired by ChatGPT and DeepSeek web clients. Workspace selection moves to a collapsible left sidebar; the main content area shows either the Launch Panel (default) or a graph view; the existing strategy/intervention panel stays as a right panel.

## Current Layout (Before)

```
[AppHeader — full-width hero banner]
[mainColumn: LaunchPanel + WorkspaceManager + GraphView] | [SidebarPanel: strategy/intervention]
```

- `AppHeader`: large branded hero with tagline, title, description, lang toggle
- `mainColumn`: stacked vertically — LaunchPanel form, WorkspaceManager list, GraphView
- `SidebarPanel` (right, 300–320px): sticky, shown only when a workspace is active

## New Layout (After)

```
appShell (height: 100vh, display: flex, flex-direction: row)
├── LeftSidebar    (240px expanded / 56px collapsed)
├── MainContent    (flex: 1, overflow: hidden)
│   ├── [no active ws] LaunchPanel (vertically centered)
│   └── [active ws]   WorkspaceHeader + GraphView (flex: 1)
└── RightPanel     (320px, only visible when workspace is active)
    └── existing SidebarPanel content, unchanged
```

## Components

### Deleted
- `AppHeader` — hero banner removed entirely; brand moves to LeftSidebar top
- `WorkspaceManager` — workspace list functionality moves into LeftSidebar; file deleted entirely (no content migration needed beyond what moves to LeftSidebar)

### Modified
- `App.tsx` — top-level layout changes to three-panel flex; routing logic between LaunchPanel and GraphView states
- `LaunchPanel` — minor: remove `sectionIntro` heading overhead; center vertically in MainContent when shown as default page
- `GraphView` — change from fixed `height: 70vh` to `flex: 1` to fill remaining height
- `SidebarPanel` — no content changes; adapt sticky/scroll behavior to new flex container
- `lib/i18n.ts` — add new translation keys for LeftSidebar (new exploration button, recent label, etc.); remove or repurpose dead `workspaces.label`, `workspaces.title`, `workspaces.description`, `workspaces.empty`, `workspaces.open`, `workspaces.archive` keys that were only used by the deleted `WorkspaceManager`

### New: `LeftSidebar`

**Expanded state (240px):**
```
┌────────────────────────┐
│  ◀  IdeaFactory        │  ← brand + collapse button
│  ─────────────────     │
│  ＋  新建探索           │  ← primary action (top, prominent)
│  ─────────────────     │
│  最近                  │  ← section label
│  ● Workspace A         │  ← active, highlighted
│    Workspace B         │
│    Workspace C         │
│                        │
│  ─────────────────     │
│  ZH / EN               │  ← lang toggle (bottom)
└────────────────────────┘
```

**Collapsed state (56px, icons only):**
```
┌──────┐
│  ▶   │  ← expand button
│  ＋   │  ← new exploration
│  ●   │  ← active ws indicator
│  ·   │
│  🌐  │  ← lang toggle
└──────┘
```

**Behavior:**
- Clicking a workspace item loads that workspace into MainContent; item highlights
- Collapsed: hovering workspace items shows tooltip with workspace name
- Clicking "New Exploration" always switches MainContent to LaunchPanel; if the sidebar is currently collapsed, it also expands. Both actions (`onNewExploration` + `onToggleCollapsed`) are called at the `App.tsx` level inside a single `handleNewExploration` handler — `LeftSidebar` fires a single `onNewExploration` callback and `App.tsx` decides whether to also expand the sidebar.
- Lang toggle uses the `lang` and `onSetLang` props directly — it does **not** call `useTranslation()` for toggle rendering, to avoid doubly-managing lang state

**`WorkspaceRecord` type:** Currently defined privately in `App.tsx`. It must be extracted to `src/types/workspace.ts` (or defined at the top of `LeftSidebar.tsx` and re-imported in `App.tsx`) before both files can share it.

**Props:**
```ts
// WorkspaceRecord extracted to src/types/workspace.ts
type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}

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
```

### New: `WorkspaceHeader`

A slim bar (40–48px) shown above GraphView when a workspace is active.

**Contents:**
- Left: current workspace topic text
- Right: Archive button (`miniAction` style, disabled when `loading` is true) + inline error display (if any)

**`onArchive` signature:** `() => void`. The active workspace ID is already known at the `App.tsx` render site, so `App.tsx` binds the ID and passes a no-arg callback: `onArchive={() => handleArchiveWorkspace(exploration.id)}`.

**Props:**
```ts
type WorkspaceHeaderProps = {
  topic: string
  loading?: boolean
  error?: string
  onArchive: () => void
}
```

## State Changes in `App.tsx`

New state:
```ts
const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
// view mode: 'launch' shows LaunchPanel, 'workspace' shows GraphView
const [viewMode, setViewMode] = useState<'launch' | 'workspace'>('launch')
```

`handleNewExploration` (new handler in `App.tsx`):
```ts
function handleNewExploration() {
  setViewMode('launch')
  setSidebarCollapsed(false)  // expand sidebar if collapsed
}
```

Navigation behavior:
- Default on page load → `viewMode = 'launch'`, LaunchPanel is shown
- User clicks workspace in sidebar → load workspace, `setViewMode('workspace')`
- User submits LaunchPanel form successfully → `setExploration(...)`, `setViewMode('workspace')`
- User clicks "New Exploration" in sidebar → `handleNewExploration()` → `viewMode = 'launch'`, sidebar expands

> **Intentional regression:** The current `useEffect` at App.tsx line 381–386 auto-loads the most recent workspace on mount. This is **removed**. After the redesign, users always land on LaunchPanel on page load or refresh and must manually select a workspace from the sidebar. This is the intended behavior matching the ChatGPT-style UX.

Error state:
- `error` string is cleared (`setError('')`) when the user selects a workspace from the sidebar (`handleSelectWorkspace`) and when `handleNewExploration` is called, so stale errors do not persist across navigation.

## CSS Changes

- `appShell`: change from `display: grid; width: min(1520px, 100%); margin: 0 auto` to `display: flex; flex-direction: row; height: 100vh; overflow: hidden; width: 100%` — the 1520px max-width cap and centered margin are removed to allow true full-viewport three-panel layout
- `mainColumn` → `mainContent`: `flex: 1; display: flex; flex-direction: column; overflow: hidden`
- `appGrid` class removed
- Dead CSS from deleted `WorkspaceManager` removed: `.workspaceList`, `.workspaceCard`, `.workspaceCardActive`, `.workspaceActions` (App.css ~lines 370–402)
- New classes added: `.leftSidebar`, `.leftSidebarCollapsed`, `.sidebarNav`, `.sidebarItem`, `.sidebarItemActive`, `.workspaceHeader`
- `.graphContainer`: remove fixed `height: 70vh`; set `flex: 1` and `min-height: 0` so it fills the column
- `.sidebarPanel` sticky/scroll logic adapted to flex column context (the right panel's parent is now a flex column, not a grid)

## Error Handling

- Error banner (currently in `mainColumn`) moves to `WorkspaceHeader` when a workspace is active, or stays inline below LaunchPanel form when in new-exploration state

## Out of Scope

- No changes to backend
- No changes to `SidebarPanel` content or logic
- No changes to `GraphView` internals (D3 forces, node rendering, edge rendering)
- No changes to `LaunchPanel` form logic
- No responsive/mobile layout changes beyond what naturally follows from the new structure
