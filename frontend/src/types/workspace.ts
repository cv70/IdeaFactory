export type WorkspaceStatus = 'draft' | 'active' | 'paused' | 'archived'

export type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}
