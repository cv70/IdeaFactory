import type { WorkspaceStatus } from '../types/workspace'
import { useTranslation } from '../lib/i18n'

type WorkspaceHeaderProps = {
  topic: string
  workspaceStatus: WorkspaceStatus
  loading?: boolean
  error?: string
  onTogglePause: () => void
  onArchive: () => void
}

export function WorkspaceHeader(props: WorkspaceHeaderProps) {
  const { t } = useTranslation()
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
          onClick={props.onTogglePause}
          disabled={props.loading}
        >
          {props.workspaceStatus === 'paused'
            ? t('workspaces.resume')
            : t('workspaces.pause')}
        </button>
        <button
          type="button"
          className="miniAction"
          onClick={props.onArchive}
          disabled={props.loading}
        >
          {t('workspaces.archive')}
        </button>
      </div>
    </div>
  )
}
