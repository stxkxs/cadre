export type ModelProvider = 'anthropic' | 'openai' | 'groq' | 'claude-code';

export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
}

export interface WorkflowNode {
  id: string;
  type: 'agent' | 'condition' | 'input' | 'output' | 'parallel' | 'loop';
  position: { x: number; y: number };
  data: {
    label: string;
    provider?: ModelProvider;
    model?: string;
    systemPrompt?: string;
    temperature?: number;
    maxTokens?: number;
    tools?: ToolDefinition[];
    condition?: string;
    retries?: number;
    timeout?: number;
    maxTurns?: number;
    workspace?: 'off' | 'safe' | 'full';
  };
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  label?: string;
  condition?: string;
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  variables: Record<string, string>;
  createdAt: Date;
  updatedAt: Date;
  userId: string;
}

export interface RunState {
  runId: string;
  workflowId: string;
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled';
  nodeStates: Record<string, NodeRunState>;
  context: Record<string, unknown>;
  startedAt: Date;
  completedAt?: Date;
  totalTokens: { input: number; output: number; cost: number };
}

export interface NodeRunState {
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  output?: string;
  error?: string;
  tokens?: { input: number; output: number };
  files?: { path: string; size: number }[];
  startedAt?: Date;
  completedAt?: Date;
}

export interface ExecutionEvent {
  type: 'node-start' | 'node-output' | 'node-complete' | 'node-error' | 'run-complete' | 'run-error';
  nodeId?: string;
  data: unknown;
  timestamp: Date;
}
