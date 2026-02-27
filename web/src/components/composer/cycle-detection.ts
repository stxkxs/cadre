import type { Edge } from '@xyflow/react'

/**
 * Checks if adding an edge from source to target would create a cycle in the DAG.
 * Uses DFS from target to see if source is reachable.
 */
export function wouldCreateCycle(source: string, target: string, edges: Edge[]): boolean {
  if (source === target) return true

  // Build adjacency list from existing edges
  const adj = new Map<string, string[]>()
  for (const edge of edges) {
    const neighbors = adj.get(edge.source) || []
    neighbors.push(edge.target)
    adj.set(edge.source, neighbors)
  }

  // DFS from target — if we can reach source, adding source->target creates a cycle
  const visited = new Set<string>()
  const stack = [target]

  while (stack.length > 0) {
    const node = stack.pop()!
    if (node === source) return true
    if (visited.has(node)) continue
    visited.add(node)

    const neighbors = adj.get(node) || []
    for (const neighbor of neighbors) {
      if (!visited.has(neighbor)) {
        stack.push(neighbor)
      }
    }
  }

  return false
}
