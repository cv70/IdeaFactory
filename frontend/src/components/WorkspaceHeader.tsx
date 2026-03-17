import { useTranslation } from '../lib/i18n'

type WorkspaceHeaderProps = {
  topic: string
  loading?: boolean
  error?: string
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
          onClick={props.onArchive}
          disabled={props.loading}
        >
          {t('workspaces.archive')}
        </button>
      </div>
    </div>
  )
}
