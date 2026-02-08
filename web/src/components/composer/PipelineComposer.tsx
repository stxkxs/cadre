import { useCallback, useState, useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  useNodesState,
  useEdgesState,
  type Connection,
  type Node,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { TaskNode } from './TaskNode'
import { NodePalette } from './NodePalette'
import { PropertiesPanel } from './PropertiesPanel'
import { ComposerToolbar } from './ComposerToolbar'
import { flowToCrewConfig, crewConfigToFlow, type TaskNodeData } from './flow-utils'
import { useCreateCrew, useUpdateCrew, useValidateCrew } from '@/hooks/useCrews'
import { useStartRun } from '@/hooks/useRuns'
import { toast } from 'sonner'
import type { Crew } from '@/types'

const nodeTypes = { taskNode: TaskNode }

interface PipelineComposerProps {
  initialCrew?: Crew
}

export function PipelineComposer({ initialCrew }: PipelineComposerProps) {
  const initial = initialCrew ? crewConfigToFlow(initialCrew) : { nodes: [], edges: [] }
  const [nodes, setNodes, onNodesChange] = useNodesState(initial.nodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initial.edges)
  const [selectedNode, setSelectedNode] = useState<Node<TaskNodeData> | null>(null)

  const [meta, setMeta] = useState<{
    name: string
    description: string
    process: string
    manager: string
    errorStrategy: string
    concurrency: number
  }>({
    name: initialCrew?.name || '',
    description: initialCrew?.description || '',
    process: initialCrew?.process || 'sequential',
    manager: initialCrew?.manager || '',
    errorStrategy: initialCrew?.error_strategy || 'fail-fast',
    concurrency: initialCrew?.concurrency || 0,
  })

  const createCrew = useCreateCrew()
  const updateCrew = useUpdateCrew()
  const validateCrew = useValidateCrew()
  const startRun = useStartRun()

  const onConnect = useCallback(
    (connection: Connection) => setEdges((eds) => addEdge({ ...connection, animated: true }, eds)),
    [setEdges],
  )

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      setSelectedNode(node as Node<TaskNodeData>)
    },
    [],
  )

  const onPaneClick = useCallback(() => setSelectedNode(null), [])

  const addTask = useCallback(
    (name: string, agent: string) => {
      const newNode: Node<TaskNodeData> = {
        id: name + '-' + Date.now(),
        type: 'taskNode',
        position: { x: 250, y: (nodes.length) * 150 },
        data: { label: name, agent },
      }
      setNodes((nds) => [...nds, newNode])
    },
    [nodes.length, setNodes],
  )

  const updateNodeData = useCallback(
    (id: string, data: Partial<TaskNodeData>) => {
      setNodes((nds) =>
        nds.map((n) =>
          n.id === id ? { ...n, data: { ...n.data, ...data } } : n,
        ),
      )
      if (selectedNode?.id === id) {
        setSelectedNode((prev) => prev ? { ...prev, data: { ...prev.data, ...data } } : null)
      }
    },
    [selectedNode, setNodes],
  )

  const handleSave = useCallback(() => {
    if (!meta.name) {
      toast.error('Crew name is required')
      return
    }
    const crew = flowToCrewConfig(nodes as Node<TaskNodeData>[], edges, meta)
    if (initialCrew) {
      updateCrew.mutate(
        { name: meta.name, crew },
        {
          onSuccess: () => toast.success('Crew saved'),
          onError: (err) => toast.error('Save failed: ' + err.message),
        },
      )
    } else {
      createCrew.mutate(crew, {
        onSuccess: () => toast.success('Crew saved'),
        onError: (err) => toast.error('Save failed: ' + err.message),
      })
    }
  }, [nodes, edges, meta, initialCrew, createCrew, updateCrew])

  const handleValidate = useCallback(() => {
    if (!meta.name) {
      toast.error('Save the crew first')
      return
    }
    validateCrew.mutate(meta.name, {
      onSuccess: (result) => {
        if (result.valid) {
          toast.success('Crew is valid')
        } else {
          toast.error('Validation errors: ' + result.errors.join(', '))
        }
      },
    })
  }, [meta.name, validateCrew])

  const handleRun = useCallback(() => {
    if (!meta.name) {
      toast.error('Save the crew first')
      return
    }
    startRun.mutate(
      { crew: meta.name },
      {
        onSuccess: (data) => toast.success(`Run started: ${data.id.slice(0, 8)}`),
        onError: (err) => toast.error('Run failed: ' + err.message),
      },
    )
  }, [meta.name, startRun])

  // Handle drop of agents from palette onto canvas
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()
      const agentName = event.dataTransfer.getData('application/cadre-agent')
      if (!agentName) return

      const reactFlowBounds = (event.target as HTMLElement).closest('.react-flow')?.getBoundingClientRect()
      if (!reactFlowBounds) return

      const position = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      }

      const newNode: Node<TaskNodeData> = {
        id: `task-${Date.now()}`,
        type: 'taskNode',
        position,
        data: { label: `${agentName}-task`, agent: agentName },
      }
      setNodes((nds) => [...nds, newNode])
    },
    [setNodes],
  )

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  // Memoize selectedNode from current nodes
  const currentSelectedNode = useMemo(
    () => (selectedNode ? (nodes.find((n) => n.id === selectedNode.id) as Node<TaskNodeData> | undefined) : null),
    [nodes, selectedNode],
  )

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      <ComposerToolbar
        crewName={meta.name}
        crewDescription={meta.description}
        process={meta.process}
        manager={meta.manager}
        errorStrategy={meta.errorStrategy}
        concurrency={meta.concurrency}
        onSave={handleSave}
        onValidate={handleValidate}
        onRun={handleRun}
        onMetaChange={setMeta}
        saving={createCrew.isPending || updateCrew.isPending}
        validating={validateCrew.isPending}
      />
      <div className="flex flex-1 overflow-hidden">
        <NodePalette onAddTask={addTask} />
        <div className="flex-1" onDrop={onDrop} onDragOver={onDragOver}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            nodeTypes={nodeTypes}
            fitView
            className="bg-background"
          >
            <Background gap={20} size={1} />
            <Controls />
            <MiniMap className="!bg-card" />
          </ReactFlow>
        </div>
        {currentSelectedNode && (
          <PropertiesPanel node={currentSelectedNode} onUpdate={updateNodeData} />
        )}
      </div>
    </div>
  )
}
