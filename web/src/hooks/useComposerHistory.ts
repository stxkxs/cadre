import { useState, useCallback, useRef } from 'react'
import type { Node, Edge } from '@xyflow/react'
import type { TaskNodeData } from '@/components/composer/flow-utils'

interface Snapshot {
  nodes: Node<TaskNodeData>[]
  edges: Edge[]
}

interface HistoryState {
  past: Snapshot[]
  present: Snapshot
  future: Snapshot[]
}

const MAX_HISTORY = 50

export function useComposerHistory(initialNodes: Node<TaskNodeData>[], initialEdges: Edge[]) {
  const [history, setHistory] = useState<HistoryState>({
    past: [],
    present: { nodes: initialNodes, edges: initialEdges },
    future: [],
  })

  const savedSnapshot = useRef<Snapshot>({ nodes: initialNodes, edges: initialEdges })
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const nodes = history.present.nodes
  const edges = history.present.edges

  const pushState = useCallback((snapshot: Snapshot) => {
    setHistory((h) => ({
      past: [...h.past.slice(-(MAX_HISTORY - 1)), h.present],
      present: snapshot,
      future: [],
    }))
  }, [])

  const pushImmediate = useCallback((newNodes: Node<TaskNodeData>[], newEdges: Edge[]) => {
    pushState({ nodes: newNodes, edges: newEdges })
  }, [pushState])

  const pushDebounced = useCallback((newNodes: Node<TaskNodeData>[], newEdges: Edge[]) => {
    // Update present without creating undo entry yet
    setHistory((h) => ({ ...h, present: { nodes: newNodes, edges: newEdges } }))
    if (debounceTimer.current) clearTimeout(debounceTimer.current)
    debounceTimer.current = setTimeout(() => {
      setHistory((h) => ({
        past: [...h.past.slice(-(MAX_HISTORY - 1)), h.past.length > 0 ? h.past[h.past.length - 1] : h.present],
        present: h.present,
        future: [],
      }))
    }, 500)
  }, [])

  // For property changes — update present directly (undo entry created on next pushImmediate)
  const updatePresent = useCallback((newNodes: Node<TaskNodeData>[], newEdges: Edge[]) => {
    setHistory((h) => ({ ...h, present: { nodes: newNodes, edges: newEdges } }))
  }, [])

  const undo = useCallback(() => {
    setHistory((h) => {
      if (h.past.length === 0) return h
      const previous = h.past[h.past.length - 1]
      return {
        past: h.past.slice(0, -1),
        present: previous,
        future: [h.present, ...h.future],
      }
    })
  }, [])

  const redo = useCallback(() => {
    setHistory((h) => {
      if (h.future.length === 0) return h
      const next = h.future[0]
      return {
        past: [...h.past, h.present],
        present: next,
        future: h.future.slice(1),
      }
    })
  }, [])

  const canUndo = history.past.length > 0
  const canRedo = history.future.length > 0

  const isDirty = JSON.stringify(history.present) !== JSON.stringify(savedSnapshot.current)

  const markClean = useCallback(() => {
    savedSnapshot.current = history.present
  }, [history.present])

  return {
    nodes,
    edges,
    pushImmediate,
    pushDebounced,
    updatePresent,
    undo,
    redo,
    canUndo,
    canRedo,
    isDirty,
    markClean,
  }
}
