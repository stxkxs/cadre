import type { Node, Edge } from '@xyflow/react'
import dagre from 'dagre'
import type { Crew, CrewTask } from '@/types'

export interface TaskNodeData {
  label: string
  agent: string
  description?: string
  status?: string
  timeout?: string
  retry?: { max_attempts: number; backoff: string }
  inputs?: TaskInputItem[]
  outputs?: TaskOutputItem[]
  [key: string]: unknown
}

export interface TaskInputItem {
  name: string
  type: string
  required: boolean
}

export interface TaskOutputItem {
  name: string
  type: string
}

export function crewConfigToFlow(crew: Crew): { nodes: Node<TaskNodeData>[]; edges: Edge[] } {
  const nodes: Node<TaskNodeData>[] = crew.tasks.map((task, i) => ({
    id: task.name,
    type: 'taskNode',
    position: { x: 0, y: i * 150 },
    data: {
      label: task.name,
      agent: task.agent,
      description: task.description,
      timeout: task.timeout,
      retry: task.retry ? { max_attempts: task.retry.max_attempts, backoff: task.retry.backoff } : undefined,
      inputs: task.inputs,
      outputs: task.outputs,
    },
  }))

  const edges: Edge[] = []
  for (const task of crew.tasks) {
    if (task.depends_on) {
      for (const dep of task.depends_on) {
        edges.push({
          id: `${dep}-${task.name}`,
          source: dep,
          target: task.name,
          animated: true,
        })
      }
    }
  }

  return applyDagreLayout(nodes, edges)
}

export function flowToCrewConfig(
  nodes: Node<TaskNodeData>[],
  edges: Edge[],
  meta: { name: string; description: string; process: string; manager?: string; errorStrategy?: string; concurrency?: number; maxIterations?: number },
): Crew {
  const agents = [...new Set(nodes.map((n) => n.data.agent).filter(Boolean))]

  const tasks: CrewTask[] = nodes.map((node) => {
    const deps = edges
      .filter((e) => e.target === node.id)
      .map((e) => e.source)

    const crewTask: CrewTask = {
      name: node.data.label || node.id,
      agent: node.data.agent || '',
      ...(deps.length > 0 ? { depends_on: deps } : {}),
      ...(node.data.description ? { description: node.data.description } : {}),
      ...(node.data.timeout ? { timeout: node.data.timeout } : {}),
      ...(node.data.retry?.max_attempts ? { retry: node.data.retry } : {}),
      ...(node.data.inputs?.length ? { inputs: node.data.inputs } : {}),
      ...(node.data.outputs?.length ? { outputs: node.data.outputs } : {}),
    }

    return crewTask
  })

  return {
    name: meta.name,
    description: meta.description,
    agents,
    process: meta.process as Crew['process'],
    tasks,
    ...(meta.manager ? { manager: meta.manager } : {}),
    ...(meta.errorStrategy ? { error_strategy: meta.errorStrategy } : {}),
    ...(meta.concurrency ? { concurrency: meta.concurrency } : {}),
    ...(meta.maxIterations ? { max_iterations: meta.maxIterations } : {}),
  }
}

export function applyDagreLayout(
  nodes: Node<TaskNodeData>[],
  edges: Edge[],
): { nodes: Node<TaskNodeData>[]; edges: Edge[] } {
  const g = new dagre.graphlib.Graph()
  g.setDefaultEdgeLabel(() => ({}))
  g.setGraph({ rankdir: 'TB', nodesep: 80, ranksep: 120 })

  for (const node of nodes) {
    g.setNode(node.id, { width: 240, height: 80 })
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target)
  }

  dagre.layout(g)

  const layoutNodes = nodes.map((node) => {
    const pos = g.node(node.id)
    return {
      ...node,
      position: { x: pos.x - 120, y: pos.y - 40 },
    }
  })

  return { nodes: layoutNodes, edges }
}
