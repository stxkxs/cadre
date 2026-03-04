import { Graph } from './graph';
import { Scheduler } from './scheduler';
import { RunContext } from './context';
import { providerRegistry } from '../providers/registry';
import type { WorkflowNode, RunState, ExecutionEvent } from './types';
import type { ProviderMessage } from '../providers/base';
import { readdirSync, statSync, writeFileSync } from 'fs';
import { join, relative } from 'path';

export interface ExecutorOptions {
  apiKeys: Record<string, string>;
  variables?: Record<string, string>;
  workspacePath?: string;
  onEvent?: (event: ExecutionEvent) => void;
}

export class Executor {
  private graph: Graph;
  private scheduler: Scheduler;
  private context: RunContext;
  private apiKeys: Record<string, string>;
  private workspacePath?: string;
  private aborted = false;

  constructor(
    nodes: WorkflowNode[],
    edges: import('./types').WorkflowEdge[],
    options: ExecutorOptions
  ) {
    this.graph = new Graph(nodes, edges);
    this.scheduler = new Scheduler(this.graph);
    this.context = new RunContext(options.variables || {});
    this.apiKeys = options.apiKeys;
    this.workspacePath = options.workspacePath;

    if (options.onEvent) {
      this.context.onEvent(options.onEvent);
    }
  }

  async execute(): Promise<RunState> {
    const validation = this.graph.validate();
    if (!validation.valid) {
      throw new Error(`Invalid workflow: ${validation.errors.join(', ')}`);
    }

    const startTime = new Date();
    let totalInputTokens = 0;
    let totalOutputTokens = 0;

    while (!this.scheduler.isComplete() && !this.aborted) {
      const batch = this.scheduler.getNextBatch(this.context);
      if (!batch) break;

      if (batch.isParallel) {
        await Promise.all(
          batch.nodeIds.map(nodeId => this.executeNode(nodeId))
        );
      } else {
        for (const nodeId of batch.nodeIds) {
          await this.executeNode(nodeId);
        }
      }

      for (const nodeId of batch.nodeIds) {
        const state = this.context.getNodeState(nodeId);
        if (state.tokens) {
          totalInputTokens += state.tokens.input;
          totalOutputTokens += state.tokens.output;
        }
        this.scheduler.markCompleted(nodeId);
      }
    }

    const status = this.aborted
      ? 'cancelled' as const
      : Object.values(this.context.getAllNodeStates()).some(s => s.status === 'failed')
        ? 'failed' as const
        : 'completed' as const;

    // Estimate cost based on token usage
    const cost = this.estimateCost(totalInputTokens, totalOutputTokens);

    const runState: RunState = {
      runId: crypto.randomUUID(),
      workflowId: '',
      status,
      nodeStates: this.context.getAllNodeStates(),
      context: this.context.getAll(),
      startedAt: startTime,
      completedAt: new Date(),
      totalTokens: {
        input: totalInputTokens,
        output: totalOutputTokens,
        cost,
      },
    };

    this.context.emit({
      type: 'run-complete',
      data: runState,
      timestamp: new Date(),
    });

    return runState;
  }

  abort(): void {
    this.aborted = true;
  }

  getState(): { nodeStates: Record<string, unknown>; totalTokens: { input: number; output: number; cost: number } } {
    const nodeStates = this.context.getAllNodeStates();
    let input = 0, output = 0;
    for (const state of Object.values(nodeStates)) {
      if (state.tokens) {
        input += state.tokens.input;
        output += state.tokens.output;
      }
    }
    return { nodeStates, totalTokens: { input, output, cost: this.estimateCost(input, output) } };
  }

  private async executeNode(nodeId: string): Promise<void> {
    const node = this.graph.getNode(nodeId);
    if (!node) return;

    this.context.setNodeState(nodeId, { status: 'running', startedAt: new Date() });
    this.context.emit({ type: 'node-start', nodeId, data: { label: node.data.label }, timestamp: new Date() });

    try {
      // Check if any predecessor failed — skip this node if so
      const predecessors = this.graph.getPredecessors(nodeId);
      const anyPredFailed = predecessors.some(p => {
        const state = this.context.getNodeState(p);
        return state.status === 'failed';
      });
      if (anyPredFailed) {
        this.context.setNodeState(nodeId, { status: 'skipped', completedAt: new Date() });
        return;
      }

      let retries = node.data.retries || 0;
      let lastError: Error | null = null;
      const defaultTimeout = node.type === 'agent' && node.data.provider === 'claude-code' ? 600 : 120;
      const timeoutMs = (node.data.timeout || defaultTimeout) * 1000;

      while (retries >= 0) {
        const abortController = new AbortController();
        const timer = setTimeout(() => abortController.abort(), timeoutMs);
        try {
          await Promise.race([
            this.executeNodeByType(node, abortController.signal),
            new Promise<never>((_, reject) => {
              abortController.signal.addEventListener('abort', () => {
                reject(new Error(`Node "${node.data.label}" timed out after ${node.data.timeout || defaultTimeout}s`));
              });
            }),
          ]);
          lastError = null;
          break;
        } catch (error) {
          lastError = error as Error;
          retries--;
          if (retries >= 0) {
            await new Promise(r => setTimeout(r, 1000 * Math.pow(2, (node.data.retries || 0) - retries - 1))); // Exponential backoff
          }
        } finally {
          clearTimeout(timer);
        }
      }

      if (lastError) throw lastError;

      const output = this.context.getNodeOutput(nodeId);
      this.context.setNodeState(nodeId, { status: 'completed', completedAt: new Date(), output });
      this.context.emit({
        type: 'node-complete',
        nodeId,
        data: { output: this.context.getNodeOutput(nodeId) },
        timestamp: new Date(),
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.context.setNodeState(nodeId, {
        status: 'failed',
        error: errorMessage,
        completedAt: new Date(),
      });
      this.context.emit({ type: 'node-error', nodeId, data: { error: errorMessage }, timestamp: new Date() });
    }
  }

  private async executeNodeByType(node: WorkflowNode, signal?: AbortSignal): Promise<void> {
    switch (node.type) {
      case 'agent':
        await this.executeAgentNode(node, signal);
        break;
      case 'condition':
        this.executeConditionNode(node);
        break;
      case 'input':
        this.executeInputNode(node);
        break;
      case 'output':
        this.executeOutputNode(node);
        break;
      case 'parallel':
        // Parallel nodes are handled by the scheduler
        break;
      case 'loop':
        await this.executeLoopNode(node);
        break;
    }
  }

  private estimateCost(inputTokens: number, outputTokens: number): number {
    // Calculate cost based on actual providers used in the workflow
    const providerRates: Record<string, { input: number; output: number }> = {
      anthropic: { input: 3.0, output: 15.0 },   // Claude Sonnet per million tokens
      openai: { input: 2.5, output: 10.0 },       // GPT-4o per million tokens
      groq: { input: 0.59, output: 0.79 },        // Llama-3.3-70b via Groq
      'claude-code': { input: 3.0, output: 15.0 }, // Same as Anthropic
    };

    // Find the primary provider from nodes
    const agentNodes = this.graph.getNodes().filter(n => n.type === 'agent' && n.data.provider);
    const primaryProvider = agentNodes[0]?.data.provider || 'anthropic';
    const rates = providerRates[primaryProvider] || providerRates.anthropic;

    return (inputTokens * rates.input + outputTokens * rates.output) / 1_000_000;
  }

  private async executeAgentNode(node: WorkflowNode, signal?: AbortSignal): Promise<void> {
    const provider = node.data.provider;
    if (!provider) throw new Error('Agent node must have a provider');

    const apiKey = this.apiKeys[provider] || '';
    if (!apiKey && provider !== 'claude-code') throw new Error(`No API key configured for ${provider}`);

    const providerInstance = providerRegistry.get(provider);
    const workspaceMode = node.data.workspace || 'off';
    const workspaceEnabled = workspaceMode !== 'off' && !!this.workspacePath;

    // Snapshot workspace files before execution (for diff)
    let filesBefore: Map<string, number> | undefined;
    if (workspaceEnabled) {
      filesBefore = this.scanWorkspaceFiles(this.workspacePath!);
    }

    // Build messages from context — label each predecessor's output so the agent knows the source
    const predecessors = this.graph.getPredecessors(node.id);
    const previousOutputs = predecessors
      .map(p => {
        const output = this.context.getNodeOutput(p);
        if (!output) return null;
        const predNode = this.graph.getNode(p);
        const label = predNode?.data.label || p;
        return predecessors.length > 1
          ? `[Output from "${label}"]\n${output}`
          : output;
      })
      .filter(Boolean)
      .join('\n\n');

    const messages: ProviderMessage[] = [];
    if (node.data.systemPrompt) {
      messages.push({ role: 'system', content: node.data.systemPrompt });
    }

    const userContent = previousOutputs || (this.context.get('input') as string) || 'Begin';
    messages.push({ role: 'user', content: userContent });

    let fullOutput = '';
    const stream = providerInstance.stream(messages, {
      model: node.data.model || '',
      temperature: node.data.temperature,
      maxTokens: node.data.maxTokens,
      workspacePath: workspaceEnabled ? this.workspacePath : undefined,
      workspace: workspaceMode,
      maxTurns: node.data.maxTurns,
      signal,
    }, apiKey);

    try {
      for await (const chunk of stream) {
        if (chunk.type === 'text') {
          fullOutput += chunk.content;
          this.context.emit({
            type: 'node-output',
            nodeId: node.id,
            data: { chunk: chunk.content },
            timestamp: new Date(),
          });
        } else if (chunk.type === 'done' && chunk.tokens) {
          this.context.setNodeState(node.id, { tokens: chunk.tokens });
        }
      }
    } finally {
      // Always save output — even partial output on timeout/cancellation
      if (fullOutput) {
        this.context.setNodeOutput(node.id, fullOutput);
        this.context.setNodeState(node.id, { output: fullOutput });
      }

      // Workspace post-processing: scan for files even on timeout
      // (Claude Code may have created files before being killed)
      if (workspaceEnabled && this.workspacePath) {
        if (fullOutput) {
          const safeLabel = (node.data.label || node.id).replace(/[^a-zA-Z0-9_-]/g, '_');
          const outputPath = join(this.workspacePath, `${safeLabel}.md`);
          writeFileSync(outputPath, fullOutput, 'utf-8');
        }

        const filesAfter = this.scanWorkspaceFiles(this.workspacePath);
        const changedFiles: { path: string; size: number }[] = [];

        for (const [filePath, size] of filesAfter) {
          const prevSize = filesBefore?.get(filePath);
          if (prevSize === undefined || prevSize !== size) {
            changedFiles.push({ path: filePath, size });
          }
        }

        if (changedFiles.length > 0) {
          this.context.setNodeState(node.id, { files: changedFiles });
        }
      }
    }
  }

  private scanWorkspaceFiles(dirPath: string): Map<string, number> {
    const files = new Map<string, number>();
    try {
      this.walkDir(dirPath, dirPath, files);
    } catch {
      // Directory may not exist yet
    }
    return files;
  }

  private walkDir(baseDir: string, currentDir: string, files: Map<string, number>): void {
    const entries = readdirSync(currentDir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = join(currentDir, entry.name);
      if (entry.isDirectory()) {
        // Skip hidden dirs and node_modules
        if (entry.name.startsWith('.') || entry.name === 'node_modules') continue;
        this.walkDir(baseDir, fullPath, files);
      } else if (entry.isFile()) {
        const relativePath = relative(baseDir, fullPath);
        const stat = statSync(fullPath);
        files.set(relativePath, stat.size);
      }
    }
  }

  private executeConditionNode(node: WorkflowNode): void {
    const condition = node.data.condition;
    if (!condition) throw new Error('Condition node must have a condition expression');

    const result = this.context.evaluateCondition(condition);
    this.context.setNodeOutput(node.id, String(result));

    // Skip branches based on condition result
    const outgoingEdges = this.graph.getOutgoingEdges(node.id);
    for (const edge of outgoingEdges) {
      if (edge.condition) {
        const edgeResult = this.context.evaluateCondition(edge.condition);
        if (!edgeResult) {
          this.context.setNodeState(edge.target, { status: 'skipped' });
        }
      }
    }
  }

  private executeInputNode(node: WorkflowNode): void {
    const defaultValue = (node.data as Record<string, unknown>).defaultValue as string || '';
    const inputData = this.context.get('input') as string || defaultValue;
    this.context.setNodeOutput(node.id, inputData);
  }

  private executeOutputNode(node: WorkflowNode): void {
    const predecessors = this.graph.getPredecessors(node.id);
    const outputs = predecessors
      .map(p => this.context.getNodeOutput(p))
      .filter(Boolean)
      .join('\n\n');
    this.context.setNodeOutput(node.id, outputs);
    this.context.set('output', outputs);
  }

  private async executeLoopNode(node: WorkflowNode): Promise<void> {
    const maxIterations = 10;
    let iteration = 0;

    while (iteration < maxIterations) {
      iteration++;
      this.context.set(`loop_${node.id}_iteration`, iteration);

      if (node.data.condition) {
        const shouldContinue = this.context.evaluateCondition(node.data.condition);
        if (!shouldContinue) break;
      }

      this.context.setNodeOutput(node.id, `Loop iteration ${iteration}`);
    }
  }
}
