import type { ExplorationMutation, ExplorationSession } from '../types/exploration'

export function applyExplorationMutations(
  session: ExplorationSession,
  mutations: ExplorationMutation[],
): ExplorationSession {
  const next: ExplorationSession = {
    ...session,
    nodes: [...session.nodes],
    edges: [...session.edges],
    runs: [...session.runs],
    favorites: [...session.favorites],
  }

  for (const mutation of mutations) {
    switch (mutation.kind) {
      case 'node_added':
        if (mutation.node && !next.nodes.some((node) => node.id === mutation.node!.id)) {
          next.nodes.push(mutation.node)
        }
        break
      case 'edge_added':
        if (mutation.edge && !next.edges.some((edge) => edge.id === mutation.edge!.id)) {
          next.edges.push(mutation.edge)
        }
        break
      case 'run_added':
        if (mutation.run && !next.runs.some((run) => run.id === mutation.run!.id)) {
          next.runs.push(mutation.run)
        }
        break
      case 'favorites_updated':
        next.favorites = mutation.favorites ? [...mutation.favorites] : []
        break
      case 'active_opportunity_set':
        if (mutation.active_opportunity_id) {
          next.activeOpportunityId = mutation.active_opportunity_id
        }
        break
      case 'strategy_updated':
        if (mutation.strategy) {
          next.strategy = mutation.strategy
        }
        break
      default:
        break
    }
  }

  return next
}
