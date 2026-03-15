import type { ExplorationInput, ExplorationSession, WorkbenchView } from './exploration'

export type ApiResponse<T> = {
  code: number
  msg?: string
  data: T
}

export type ExplorationPayload = {
  exploration: ExplorationSession
  presentation: WorkbenchView | null
}

export type CreateExplorationRequest = ExplorationInput

export type FeedbackRequest = {
  type: 'toggle_favorite'
  nodeId: string
}
