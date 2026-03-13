import type { ExplorationSession } from '../types/exploration'

export function createMockExplorationRepository() {
  const sessions = new Map<string, ExplorationSession>()

  return {
    get(id: string) {
      return sessions.get(id)
    },
    set(session: ExplorationSession) {
      sessions.set(session.id, session)
      return session
    },
    update(id: string, updater: (session: ExplorationSession) => ExplorationSession) {
      const current = sessions.get(id)
      if (!current) return null
      const next = updater(current)
      sessions.set(id, next)
      return next
    },
    clear() {
      sessions.clear()
    },
  }
}

export type MockExplorationRepository = ReturnType<typeof createMockExplorationRepository>
