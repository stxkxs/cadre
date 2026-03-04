import type { NodeRunState, ExecutionEvent } from './types';

export class RunContext {
  private data: Record<string, unknown> = {};
  private nodeStates: Record<string, NodeRunState> = {};
  private events: ExecutionEvent[] = [];
  private eventListeners: ((event: ExecutionEvent) => void)[] = [];

  constructor(initialData: Record<string, unknown> = {}) {
    this.data = { ...initialData };
  }

  get(key: string): unknown {
    return this.data[key];
  }

  set(key: string, value: unknown): void {
    this.data[key] = value;
  }

  getAll(): Record<string, unknown> {
    return { ...this.data };
  }

  setNodeOutput(nodeId: string, output: string): void {
    this.data[`node_${nodeId}_output`] = output;
  }

  getNodeOutput(nodeId: string): string | undefined {
    return this.data[`node_${nodeId}_output`] as string | undefined;
  }

  getNodeState(nodeId: string): NodeRunState {
    return this.nodeStates[nodeId] || { status: 'pending' };
  }

  setNodeState(nodeId: string, state: Partial<NodeRunState>): void {
    this.nodeStates[nodeId] = {
      ...this.nodeStates[nodeId],
      ...state,
    } as NodeRunState;
  }

  getAllNodeStates(): Record<string, NodeRunState> {
    return { ...this.nodeStates };
  }

  emit(event: ExecutionEvent): void {
    this.events.push(event);
    for (const listener of this.eventListeners) {
      listener(event);
    }
  }

  onEvent(listener: (event: ExecutionEvent) => void): () => void {
    this.eventListeners.push(listener);
    return () => {
      this.eventListeners = this.eventListeners.filter(l => l !== listener);
    };
  }

  getEvents(): ExecutionEvent[] {
    return [...this.events];
  }

  evaluateCondition(expression: string): boolean {
    try {
      // Block dangerous patterns — prevent access to process, require, import, global objects
      const forbidden = /\b(process|require|import|eval|Function|globalThis|global|window|document|fetch|XMLHttpRequest|__dirname|__filename)\b/;
      if (forbidden.test(expression)) {
        console.warn(`[condition] Blocked dangerous expression: ${expression}`);
        return false;
      }

      // Limit expression length to prevent abuse
      if (expression.length > 1000) {
        console.warn('[condition] Expression too long, rejected');
        return false;
      }

      // Create a sandboxed context with only the workflow data
      const sandboxedContext: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(this.data)) {
        // Only pass primitive values and plain objects, not functions
        if (typeof value !== 'function') {
          sandboxedContext[key] = value;
        }
      }

      const fn = new Function(
        'context',
        `with(context) { return Boolean(${expression}); }`
      );
      return fn(sandboxedContext);
    } catch (error) {
      console.warn(`[condition] Failed to evaluate: ${expression}`, error);
      return false;
    }
  }
}
