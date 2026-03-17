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

### Removed
- `AppHeader` — hero banner removed entirely; brand moves to LeftSidebar top

### Modified
- `App.tsx` — top-level layout changes to three-panel flex; routing logic between LaunchPanel and GraphView states
- `LaunchPanel` — minor: remove `sectionIntro` heading overhead; center vertically in MainContent when shown as default page
- `GraphView` — change from fixed `height: 70vh` to `flex: 1` to fill remaining height
- `SidebarPanel` — no content changes; adapt sticky/scroll behavior to new flex container

### Removed
- `WorkspaceManager` — workspace list functionality moves into LeftSidebar; component deleted

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
- Collapsed: clicking "New Exploration" auto-expands sidebar and switches MainContent to LaunchPanel
- Lang toggle at bottom, same logic as before

**Props:**
```ts
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
- Right: Archive button (`miniAction` style) + inline error display (if any)

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
```

New derived view state (replaces auto-load-latest behavior):
- If `exploration` is null and user clicks "New Exploration" → show LaunchPanel
- If `exploration` is null and workspace history is loaded → **do not** auto-load latest (user starts on LaunchPanel by default)
- If user clicks a workspace in the sidebar → load and set `exploration`

> **Note:** The current `App.tsx` has a `useEffect` that auto-loads the most recent workspace on mount. This should be **removed** so the default state is the LaunchPanel, consistent with the new layout.

## CSS Changes

- `appShell`: change from `display: grid` to `display: flex; flex-direction: row; height: 100vh; overflow: hidden`
- `mainColumn` → `mainContent`: `flex: 1; display: flex; flex-direction: column; overflow: hidden`
- `appGrid` class removed
- New `.leftSidebar`, `.leftSidebarCollapsed`, `.workspaceHeader` classes added
- `.graphContainer`: remove fixed `height: 70vh`; set `flex: 1` so it fills the column
- `.sidebarPanel` sticky/scroll logic adapted to flex column context

## Error Handling

- Error banner (currently in `mainColumn`) moves to `WorkspaceHeader` when a workspace is active, or stays inline below LaunchPanel form when in new-exploration state

## Out of Scope

- No changes to backend
- No changes to `SidebarPanel` content or logic
- No changes to `GraphView` internals (D3 forces, node rendering, edge rendering)
- No changes to `LaunchPanel` form logic
- No responsive/mobile layout changes beyond what naturally follows from the new structure
