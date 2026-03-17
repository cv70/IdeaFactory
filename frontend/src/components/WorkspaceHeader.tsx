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
