import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Executor } from '../executor';
import type { WorkflowNode, WorkflowEdge } from '../types';
import type { StreamChunk } from '../../providers/base';

// Mock the provider registry
vi.mock('../../providers/registry', () => {
  const mockProvider = {
    id: 'anthropic',
    name: 'Mock',
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

// Mock fs operations
vi.mock('fs', () => ({
  readdirSync: vi.fn(() => []),
  statSync: vi.fn(() => ({ size: 0 })),
  writeFileSync: vi.fn(),
}));

function makeNode(id: string, type: WorkflowNode['type'] = 'agent', data: Partial<WorkflowNode['data']> = {}): WorkflowNode {
  return { id, type, position: { x: 0, y: 0 }, data: { label: id, ...data } };
}

function makeEdge(id: string, source: string, target: string): WorkflowEdge {
  return { id, source, target };
}

async function* mockStream(chunks: StreamChunk[]): AsyncGenerator<StreamChunk> {
  for (const chunk of chunks) {
    yield chunk;
  }
}

describe('Executor', () => {
  let mockProvider: { stream: ReturnType<typeof vi.fn>; chat: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    vi.clearAllMocks();
    const registry = await import('../../providers/registry');
    mockProvider = (registry.providerRegistry as unknown as { __mockProvider: typeof mockProvider }).__mockProvider;
  });

  it('executes a linear pipeline: input → agent → output', async () => {
    mockProvider.stream.mockReturnValue(
      mockStream([
        { type: 'text', content: 'Agent response' },
        { type: 'done', content: '', tokens: { input: 10, output: 20 } },
      ]),
    );

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
      variables: { input: 'Hello world' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('completed');
    expect(result.nodeStates.in.status).toBe('completed');
    expect(result.nodeStates.agent.status).toBe('completed');
    expect(result.nodeStates.out.status).toBe('completed');
    expect(result.context).toHaveProperty('output');
  });

  it('executes parallel branches', async () => {
    mockProvider.stream.mockReturnValue(
      mockStream([
        { type: 'text', content: 'Response' },
        { type: 'done', content: '', tokens: { input: 5, output: 10 } },
      ]),
    );

    const nodes = [
      makeNode('in', 'input'),
      makeNode('a1', 'agent', { provider: 'anthropic' }),
      makeNode('a2', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [
      makeEdge('e1', 'in', 'a1'),
      makeEdge('e2', 'in', 'a2'),
      makeEdge('e3', 'a1', 'out'),
      makeEdge('e4', 'a2', 'out'),
    ];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('completed');
    expect(result.nodeStates.a1.status).toBe('completed');
    expect(result.nodeStates.a2.status).toBe('completed');
  });

  it('propagates errors and skips downstream nodes', async () => {
    mockProvider.stream.mockImplementation(() => {
      throw new Error('Provider error');
    });

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('failed');
    expect(result.nodeStates.agent.status).toBe('failed');
    expect(result.nodeStates.out.status).toBe('skipped');
  });

  it('aborts execution when cancelled', async () => {
    let resolveStream: () => void;
    const blockPromise = new Promise<void>(r => { resolveStream = r; });

    mockProvider.stream.mockImplementation(async function* () {
      yield { type: 'text', content: 'partial' };
      await blockPromise;
      yield { type: 'done', content: '', tokens: { input: 0, output: 0 } };
    });

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const promise = executor.execute();
    // Give time for execution to start
    await new Promise(r => setTimeout(r, 50));
    executor.abort();
    resolveStream!();

    const result = await promise;
    expect(result.status).toBe('cancelled');
  });

  it('tracks cost estimation', async () => {
    mockProvider.stream.mockReturnValue(
      mockStream([
        { type: 'text', content: 'Response' },
        { type: 'done', content: '', tokens: { input: 1000, output: 500 } },
      ]),
    );

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const result = await executor.execute();
    expect(result.totalTokens.input).toBe(1000);
    expect(result.totalTokens.output).toBe(500);
    expect(result.totalTokens.cost).toBeGreaterThan(0);
  });

  it('emits events in correct sequence', async () => {
    mockProvider.stream.mockReturnValue(
      mockStream([
        { type: 'text', content: 'Hello' },
        { type: 'done', content: '', tokens: { input: 5, output: 5 } },
      ]),
    );

    const events: string[] = [];
    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
      onEvent: (event) => events.push(`${event.type}:${event.nodeId || 'run'}`),
    });

    await executor.execute();

    expect(events).toContain('node-start:in');
    expect(events).toContain('node-complete:in');
    expect(events).toContain('node-start:agent');
    expect(events).toContain('node-output:agent');
    expect(events).toContain('node-complete:agent');
    expect(events).toContain('node-start:out');
    expect(events).toContain('node-complete:out');
    expect(events).toContain('run-complete:run');
  });

  it('retries failed nodes with backoff', async () => {
    let callCount = 0;
    mockProvider.stream.mockImplementation(async function* () {
      callCount++;
      if (callCount < 3) throw new Error('Transient error');
      yield { type: 'text' as const, content: 'Success' };
      yield { type: 'done' as const, content: '', tokens: { input: 5, output: 5 } };
    });

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic', retries: 2 }),
      makeNode('out', 'output'),
    ];
    const edges = [makeEdge('e1', 'in', 'agent'), makeEdge('e2', 'agent', 'out')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    const result = await executor.execute();
    expect(result.status).toBe('completed');
    expect(callCount).toBe(3);
  });

  it('rejects invalid workflows', async () => {
    const executor = new Executor([], [], {
      apiKeys: { anthropic: 'test-key' },
    });

    await expect(executor.execute()).rejects.toThrow('Invalid workflow');
  });

  it('getState returns current state and token totals', async () => {
    mockProvider.stream.mockReturnValue(
      mockStream([
        { type: 'text', content: 'Hello' },
        { type: 'done', content: '', tokens: { input: 10, output: 20 } },
      ]),
    );

    const nodes = [
      makeNode('in', 'input'),
      makeNode('agent', 'agent', { provider: 'anthropic' }),
    ];
    const edges = [makeEdge('e1', 'in', 'agent')];

    const executor = new Executor(nodes, edges, {
      apiKeys: { anthropic: 'test-key' },
    });

    await executor.execute();
    const state = executor.getState();
    expect(state.totalTokens.input).toBe(10);
    expect(state.totalTokens.output).toBe(20);
  });
});
