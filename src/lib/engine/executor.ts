import { Graph } from './graph';
import { Scheduler } from './scheduler';
import { RunContext } from './context';
import { ClaudeCodeProvider } from '../providers/claude-code';
import type { WorkflowNode, RunState, ExecutionEvent } from './types';
import { readdirSync, statSync, writeFileSync } from 'fs';
import { join, relative } from 'path';

export interface ExecutorOptions {
  variables?: Record<string, string>;
  workspacePath?: string;
  onEvent?: (event: ExecutionEvent) => void;
}

const claudeCode = new ClaudeCodeProvider();

export class Executor {
  private graph: Graph;
  private scheduler: Scheduler;
  private context: RunContext;
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
      const timeoutMs = (node.data.timeout || 600) * 1000;

      while (retries >= 0) {
        const abortController = new AbortController();
        const timer = setTimeout(() => abortController.abort(), timeoutMs);
        try {
          await Promise.race([
            this.executeNodeByType(node, abortController.signal),
            new Promise<never>((_, reject) => {
              abortController.signal.addEventListener('abort', () => {
                reject(new Error(`Node "${node.data.label}" timed out after ${node.data.timeout || 600}s`));
              });
            }),
          ]);
          lastError = null;
          break;
        } catch (error) {
          lastError = error as Error;
          retries--;
          if (retries >= 0) {
            await new Promise(r => setTimeout(r, 1000 * Math.pow(2, (node.data.retries || 0) - retries - 1)));
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
        await this.executeConditionNode(node);
        break;
      case 'router':
        await this.executeRouterNode(node);
        break;
      case 'transform':
        await this.executeTransformNode(node);
        break;
      case 'gate':
        await this.executeGateNode(node);
        break;
      case 'input':
        this.executeInputNode(node);
        break;
      case 'output':
        this.executeOutputNode(node);
        break;
      case 'loop':
        await this.executeLoopNode(node);
        break;
    }
  }

  private estimateCost(inputTokens: number, outputTokens: number): number {
    // Claude Code rate (Claude Sonnet equivalent)
    return (inputTokens * 3.0 + outputTokens * 15.0) / 1_000_000;
  }

  private async executeAgentNode(node: WorkflowNode, signal?: AbortSignal): Promise<void> {
    const workspaceMode = node.data.workspace || 'off';
    const workspaceEnabled = workspaceMode !== 'off' && !!this.workspacePath;

    let filesBefore: Map<string, number> | undefined;
    if (workspaceEnabled) {
      filesBefore = this.scanWorkspaceFiles(this.workspacePath!);
    }

    // Build prompt from predecessor outputs
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

    const systemPrompt = node.data.systemPrompt;

    let fullOutput = '';
    const stream = claudeCode.stream(
      [
        ...(systemPrompt ? [{ role: 'system' as const, content: systemPrompt }] : []),
        { role: 'user' as const, content: previousOutputs || (this.context.get('input') as string) || 'Begin' },
      ],
      {
        model: 'claude-code',
        workspacePath: workspaceEnabled ? this.workspacePath : undefined,
        workspace: workspaceMode,
        maxTurns: node.data.maxTurns,
        signal,
      },
      ''
    );

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
      if (fullOutput) {
        this.context.setNodeOutput(node.id, fullOutput);
        this.context.setNodeState(node.id, { output: fullOutput });
      }

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

  private walkDir(baseDir: string, currentDir: string, files: Map<string, number>, maxDepth = 10): void {
    if (maxDepth <= 0 || files.size >= 10_000) return;

    const entries = readdirSync(currentDir, { withFileTypes: true });
    for (const entry of entries) {
      if (files.size >= 10_000) return;

      const fullPath = join(currentDir, entry.name);
      if (entry.isDirectory()) {
        if (entry.name.startsWith('.') || entry.name === 'node_modules') continue;
        this.walkDir(baseDir, fullPath, files, maxDepth - 1);
      } else if (entry.isFile()) {
        const relativePath = relative(baseDir, fullPath);
        const stat = statSync(fullPath);
        files.set(relativePath, stat.size);
      }
    }
  }

  private async executeRouterNode(node: WorkflowNode): Promise<void> {
    const routes = node.data.routes || [];
    if (routes.length < 2) throw new Error('Router node must have at least 2 routes');

    let matchedRoute: string | null = null;

    for (const route of routes) {
      if (!route.condition) {
        // Route without condition = default fallback
        if (!matchedRoute) matchedRoute = route.label;
        continue;
      }
      const result = await this.context.evaluateCondition(route.condition);
      if (result) {
        matchedRoute = route.label;
        break;
      }
    }

    // If no route matched, use the first route as default
    if (!matchedRoute) matchedRoute = routes[0].label;

    this.context.setNodeOutput(node.id, matchedRoute);

    // Skip downstream nodes connected to non-matching routes
    const outgoingEdges = this.graph.getOutgoingEdges(node.id);
    for (const edge of outgoingEdges) {
      // Edge sourceHandle carries the route label
      const edgeRoute = edge.sourceHandle || edge.label;
      if (edgeRoute && edgeRoute !== matchedRoute) {
        this.context.setNodeState(edge.target, { status: 'skipped' });
      }
    }
  }

  private async executeTransformNode(node: WorkflowNode): Promise<void> {
    const template = node.data.template;
    if (!template) throw new Error('Transform node must have a template');

    // Interpolate {{node_X_output}} and {{variable}} patterns
    const output = template.replace(/\{\{(\w+)\}\}/g, (_, varName) => {
      // Check node outputs first
      const nodeOutput = this.context.get(`node_${varName}_output`);
      if (nodeOutput !== undefined) return String(nodeOutput);
      // Check context variables
      const contextVal = this.context.get(varName);
      if (contextVal !== undefined) return String(contextVal);
      return '';
    });

    this.context.setNodeOutput(node.id, output);
  }

  private async executeGateNode(node: WorkflowNode): Promise<void> {
    const message = node.data.gateMessage || 'Approval required to continue';

    this.context.setNodeState(node.id, { status: 'waiting' });
    this.context.emit({
      type: 'node-waiting',
      nodeId: node.id,
      data: { message },
      timestamp: new Date(),
    });

    // Poll for gate decision — check DB every 2 seconds
    const maxWaitMs = 3600_000; // 1 hour
    const pollIntervalMs = 2000;
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitMs) {
      if (this.aborted) throw new Error('Run was cancelled');

      // Check if gate has been resolved via context
      const decision = this.context.get(`gate_${node.id}_decision`);
      if (decision === 'approved') {
        this.context.setNodeOutput(node.id, 'approved');
        return;
      }
      if (decision === 'rejected') {
        throw new Error(`Gate "${node.data.label}" was rejected`);
      }

      await new Promise(r => setTimeout(r, pollIntervalMs));
    }

    throw new Error(`Gate "${node.data.label}" timed out after 1 hour`);
  }

  private async executeConditionNode(node: WorkflowNode): Promise<void> {
    const condition = node.data.condition;
    if (!condition) throw new Error('Condition node must have a condition expression');

    const result = await this.context.evaluateCondition(condition);
    this.context.setNodeOutput(node.id, String(result));

    const outgoingEdges = this.graph.getOutgoingEdges(node.id);
    for (const edge of outgoingEdges) {
      if (edge.condition) {
        const edgeResult = await this.context.evaluateCondition(edge.condition);
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
    const maxIterations = (node.data as Record<string, unknown>).maxIterations as number || 10;
    let iteration = 0;

    while (iteration < maxIterations) {
      iteration++;
      this.context.set(`loop_${node.id}_iteration`, iteration);

      if (node.data.condition) {
        const shouldContinue = await this.context.evaluateCondition(node.data.condition);
        if (!shouldContinue) break;
      }

      this.context.setNodeOutput(node.id, `Loop iteration ${iteration}`);
    }
  }
}
