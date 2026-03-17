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
