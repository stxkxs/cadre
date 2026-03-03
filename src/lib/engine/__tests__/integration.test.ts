import { describe, it, expect, vi } from 'vitest';
import { Executor } from '../executor';
import type { WorkflowNode, WorkflowEdge, ExecutionEvent } from '../types';

// Mock the provider registry with a realistic mock provider
vi.mock('../../providers/registry', () => {
  const mockProvider = {
    id: 'anthropic',
    name: 'Mock Anthropic',
    chat: vi.fn(),
    stream: vi.fn(),
    validateKey: vi.fn(),
  };

  return {
    providerRegistry: {
      get: vi.fn(() => mockProvider),
      has: vi.fn(() => true),
      __mockProvider: mockProvider,
    },
  };
});

vi.mock('fs', () => ({
  readdirSync: vi.fn(() => []),
  statSync: vi.fn(() => ({ size: 0 })),
  writeFileSync: vi.fn(),
}));

function makeNode(id: string, type: WorkflowNode['type'], data: Partial<WorkflowNode['data']> = {}): WorkflowNode {
  return { id, type, position: { x: 0, y: 0 }, data: { label: id, ...data } };
}

function makeEdge(id: string, source: string, target: string, extra: Partial<WorkflowEdge> = {}): WorkflowEdge {
  return { id, source, target, ...extra };
}

describe('Integration: Full Workflow Execution', () => {
  it('executes input → agent → condition → branch → output', async () => {
    const registry = await import('../../providers/registry');
    const mockProvider = (registry.providerRegistry as unknown as { __mockProvider: { stream: ReturnType<typeof vi.fn> } }).__mockProvider;

    mockProvider.stream.mockImplementation(async function* () {
      yield { type: 'text', content: 'The analysis is positive' };
      yield { type: 'done', content: '', tokens: { input: 50, output: 100 } };
    });

    // Build the workflow:
    // input → agent → condition → (true branch) output-yes
    //                            → (false branch) output-no
    const nodes: WorkflowNode[] = [
      makeNode('input', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic', model: 'claude-sonnet-4-6' }),
      makeNode('cond', 'condition', { condition: 'node_agent_output && node_agent_output.includes("positive")' }),
      makeNode('output-yes', 'output'),
      makeNode('output-no', 'output'),
    ];

    const edges: WorkflowEdge[] = [
      makeEdge('e1', 'input', 'agent'),
      makeEdge('e2', 'agent', 'cond'),
      makeEdge('e3', 'cond', 'output-yes', { condition: 'node_cond_output === "true"' }),
      makeEdge('e4', 'cond', 'output-no', { condition: 'node_cond_output === "false"' }),
    ];

    const events: ExecutionEvent[] = [];
    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
      variables: { input: 'Analyze this data' },
      onEvent: (event) => events.push(event),
    });

    const result = await executor.execute();

    // Verify end-to-end flow
    expect(result.status).toBe('completed');
    expect(result.nodeStates.input.status).toBe('completed');
    expect(result.nodeStates.agent.status).toBe('completed');
    expect(result.nodeStates.cond.status).toBe('completed');

    // Context should have the output populated
    expect(result.context).toHaveProperty('node_input_output');
    expect(result.context).toHaveProperty('node_agent_output');
    expect(result.context).toHaveProperty('node_cond_output');

    // Agent output should flow through
    expect(result.context.node_agent_output).toBe('The analysis is positive');

    // Verify token tracking
    expect(result.totalTokens.input).toBe(50);
    expect(result.totalTokens.output).toBe(100);
    expect(result.totalTokens.cost).toBeGreaterThan(0);

    // Verify event sequence
    const eventTypes = events.map(e => e.type);
    expect(eventTypes).toContain('node-start');
    expect(eventTypes).toContain('node-complete');
    expect(eventTypes).toContain('node-output');
    expect(eventTypes).toContain('run-complete');

    // run-complete should be the last event
    expect(events[events.length - 1].type).toBe('run-complete');

    // Timestamps should be present
    for (const event of events) {
      expect(event.timestamp).toBeInstanceOf(Date);
    }
  });

  it('handles workflow with parallel branches merging', async () => {
    const registry = await import('../../providers/registry');
    const mockProvider = (registry.providerRegistry as unknown as { __mockProvider: { stream: ReturnType<typeof vi.fn> } }).__mockProvider;

    let callCount = 0;
    mockProvider.stream.mockImplementation(async function* () {
      callCount++;
      yield { type: 'text', content: `Branch ${callCount} result` };
      yield { type: 'done', content: '', tokens: { input: 10, output: 20 } };
    });

    const nodes: WorkflowNode[] = [
      makeNode('input', 'input'),
      makeNode('branch-a', 'agent', { provider: 'anthropic' }),
      makeNode('branch-b', 'agent', { provider: 'anthropic' }),
      makeNode('output', 'output'),
    ];

    const edges: WorkflowEdge[] = [
      makeEdge('e1', 'input', 'branch-a'),
      makeEdge('e2', 'input', 'branch-b'),
      makeEdge('e3', 'branch-a', 'output'),
      makeEdge('e4', 'branch-b', 'output'),
    ];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
      variables: { input: 'Process this' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('completed');
    expect(result.nodeStates['branch-a'].status).toBe('completed');
    expect(result.nodeStates['branch-b'].status).toBe('completed');
    expect(result.nodeStates.output.status).toBe('completed');

    // Output should contain results from both branches
    expect(result.context.output).toBeTruthy();
    expect(result.totalTokens.input).toBe(20); // 10 * 2 branches
    expect(result.totalTokens.output).toBe(40); // 20 * 2 branches
  });

  it('cascades failure: failed agent skips downstream', async () => {
    const registry = await import('../../providers/registry');
    const mockProvider = (registry.providerRegistry as unknown as { __mockProvider: { stream: ReturnType<typeof vi.fn> } }).__mockProvider;

    mockProvider.stream.mockImplementation(async function* () {
      throw new Error('API rate limited');
    });

    const nodes: WorkflowNode[] = [
      makeNode('input', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('output', 'output'),
    ];

    const edges: WorkflowEdge[] = [
      makeEdge('e1', 'input', 'agent'),
      makeEdge('e2', 'agent', 'output'),
    ];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('failed');
    expect(result.nodeStates.input.status).toBe('completed');
    expect(result.nodeStates.agent.status).toBe('failed');
    expect(result.nodeStates.agent.error).toContain('API rate limited');
    expect(result.nodeStates.output.status).toBe('skipped');
  });
});
