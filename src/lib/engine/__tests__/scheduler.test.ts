import { describe, it, expect } from 'vitest';
import { Scheduler } from '../scheduler';
import { Graph } from '../graph';
import { RunContext } from '../context';
import type { WorkflowNode, WorkflowEdge } from '../types';

function makeNode(id: string, type: WorkflowNode['type'] = 'agent', data: Partial<WorkflowNode['data']> = {}): WorkflowNode {
  return { id, type, position: { x: 0, y: 0 }, data: { label: id, ...data } };
}

function makeEdge(id: string, source: string, target: string): WorkflowEdge {
  return { id, source, target };
}

describe('Scheduler', () => {
  it('returns batches in order for a linear graph', () => {
    const graph = new Graph(
      [makeNode('a'), makeNode('b'), makeNode('c')],
      [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'c')],
    );
    const scheduler = new Scheduler(graph);
    const context = new RunContext();

    // First batch: 'a' (no predecessors)
    const batch1 = scheduler.getNextBatch(context);
    expect(batch1?.nodeIds).toEqual(['a']);
    expect(batch1?.isParallel).toBe(false);

    // Complete 'a'
    context.setNodeState('a', { status: 'completed' });
    scheduler.markCompleted('a');

    // Second batch: 'b'
    const batch2 = scheduler.getNextBatch(context);
    expect(batch2?.nodeIds).toEqual(['b']);

    context.setNodeState('b', { status: 'completed' });
    scheduler.markCompleted('b');

    // Third batch: 'c'
    const batch3 = scheduler.getNextBatch(context);
    expect(batch3?.nodeIds).toEqual(['c']);
  });

  it('returns parallel nodes in a single batch for diamond graph', () => {
    const graph = new Graph(
      [makeNode('a'), makeNode('b'), makeNode('c'), makeNode('d')],
      [
        makeEdge('e1', 'a', 'b'),
        makeEdge('e2', 'a', 'c'),
        makeEdge('e3', 'b', 'd'),
        makeEdge('e4', 'c', 'd'),
      ],
    );
    const scheduler = new Scheduler(graph);
    const context = new RunContext();

    const batch1 = scheduler.getNextBatch(context);
    expect(batch1?.nodeIds).toEqual(['a']);

    context.setNodeState('a', { status: 'completed' });
    scheduler.markCompleted('a');

    // b and c should be scheduled together
    const batch2 = scheduler.getNextBatch(context);
    expect(batch2?.nodeIds).toContain('b');
    expect(batch2?.nodeIds).toContain('c');
    expect(batch2?.isParallel).toBe(true);
  });

  it('tracks completion correctly', () => {
    const graph = new Graph([makeNode('a'), makeNode('b')], [makeEdge('e1', 'a', 'b')]);
    const scheduler = new Scheduler(graph);
    const context = new RunContext();

    expect(scheduler.isComplete()).toBe(false);

    context.setNodeState('a', { status: 'completed' });
    scheduler.markCompleted('a');
    expect(scheduler.isComplete()).toBe(false);

    context.setNodeState('b', { status: 'completed' });
    scheduler.markCompleted('b');
    expect(scheduler.isComplete()).toBe(true);
  });

  it('treats skipped predecessors as done', () => {
    const graph = new Graph(
      [makeNode('a'), makeNode('b'), makeNode('c')],
      [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'c')],
    );
    const scheduler = new Scheduler(graph);
    const context = new RunContext();

    context.setNodeState('a', { status: 'completed' });
    scheduler.markCompleted('a');

    // Skip b
    context.setNodeState('b', { status: 'skipped' });

    // c should now be schedulable (b is skipped = done)
    const batch = scheduler.getNextBatch(context);
    expect(batch?.nodeIds).toContain('c');
  });

  it('returns null when no nodes are ready', () => {
    const graph = new Graph(
      [makeNode('a'), makeNode('b')],
      [makeEdge('e1', 'a', 'b')],
    );
    const scheduler = new Scheduler(graph);
    const context = new RunContext();

    // Mark a as running (not completed)
    context.setNodeState('a', { status: 'running' });
    const batch = scheduler.getNextBatch(context);
    // Only 'a' has no predecessors, but it's running. 'b' waits on 'a'.
    expect(batch).toBeNull();
  });
});
