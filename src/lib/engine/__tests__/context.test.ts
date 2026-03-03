import { describe, it, expect, vi } from 'vitest';
import { RunContext } from '../context';

describe('RunContext', () => {
  describe('get/set', () => {
    it('stores and retrieves values', () => {
      const ctx = new RunContext();
      ctx.set('key', 'value');
      expect(ctx.get('key')).toBe('value');
    });

    it('returns undefined for missing keys', () => {
      const ctx = new RunContext();
      expect(ctx.get('missing')).toBeUndefined();
    });

    it('initializes with provided data', () => {
      const ctx = new RunContext({ greeting: 'hello' });
      expect(ctx.get('greeting')).toBe('hello');
    });

    it('returns a copy from getAll', () => {
      const ctx = new RunContext({ a: 1 });
      const all = ctx.getAll();
      all.b = 2;
      expect(ctx.get('b')).toBeUndefined();
    });
  });

  describe('node output isolation', () => {
    it('stores node output with namespaced key', () => {
      const ctx = new RunContext();
      ctx.setNodeOutput('node1', 'output1');
      expect(ctx.getNodeOutput('node1')).toBe('output1');
      expect(ctx.get('node_node1_output')).toBe('output1');
    });

    it('isolates outputs between nodes', () => {
      const ctx = new RunContext();
      ctx.setNodeOutput('a', 'outputA');
      ctx.setNodeOutput('b', 'outputB');
      expect(ctx.getNodeOutput('a')).toBe('outputA');
      expect(ctx.getNodeOutput('b')).toBe('outputB');
    });
  });

  describe('node state', () => {
    it('returns default pending state for unknown nodes', () => {
      const ctx = new RunContext();
      expect(ctx.getNodeState('unknown').status).toBe('pending');
    });

    it('merges partial state updates', () => {
      const ctx = new RunContext();
      ctx.setNodeState('n1', { status: 'running' });
      ctx.setNodeState('n1', { output: 'hello' });
      const state = ctx.getNodeState('n1');
      expect(state.status).toBe('running');
      expect(state.output).toBe('hello');
    });

    it('returns a copy from getAllNodeStates', () => {
      const ctx = new RunContext();
      ctx.setNodeState('n1', { status: 'completed' });
      const all = ctx.getAllNodeStates();
      expect(all.n1.status).toBe('completed');
    });
  });

  describe('events', () => {
    it('emits events and records them', () => {
      const ctx = new RunContext();
      ctx.emit({ type: 'node-start', nodeId: 'a', data: {}, timestamp: new Date() });
      expect(ctx.getEvents()).toHaveLength(1);
      expect(ctx.getEvents()[0].type).toBe('node-start');
    });

    it('notifies subscribers on emit', () => {
      const ctx = new RunContext();
      const listener = vi.fn();
      ctx.onEvent(listener);
      ctx.emit({ type: 'node-start', nodeId: 'a', data: {}, timestamp: new Date() });
      expect(listener).toHaveBeenCalledOnce();
    });

    it('unsubscribes correctly', () => {
      const ctx = new RunContext();
      const listener = vi.fn();
      const unsub = ctx.onEvent(listener);
      unsub();
      ctx.emit({ type: 'node-start', nodeId: 'a', data: {}, timestamp: new Date() });
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('evaluateCondition', () => {
    it('evaluates simple boolean expressions', () => {
      const ctx = new RunContext({ count: 5 });
      expect(ctx.evaluateCondition('count > 3')).toBe(true);
      expect(ctx.evaluateCondition('count > 10')).toBe(false);
    });

    it('evaluates string comparisons', () => {
      const ctx = new RunContext({ status: 'active' });
      expect(ctx.evaluateCondition('status === "active"')).toBe(true);
    });

    it('blocks forbidden patterns', () => {
      const ctx = new RunContext();
      expect(ctx.evaluateCondition('process.exit(1)')).toBe(false);
      expect(ctx.evaluateCondition('require("fs")')).toBe(false);
      expect(ctx.evaluateCondition('eval("1")')).toBe(false);
      expect(ctx.evaluateCondition('globalThis.x')).toBe(false);
    });

    it('rejects expressions over 1000 characters', () => {
      const ctx = new RunContext();
      const longExpr = 'true && '.repeat(200) + 'true';
      expect(ctx.evaluateCondition(longExpr)).toBe(false);
    });

    it('handles evaluation errors gracefully', () => {
      const ctx = new RunContext();
      expect(ctx.evaluateCondition('nonexistent.property.deep')).toBe(false);
    });

    it('does not pass functions to the sandbox', () => {
      const ctx = new RunContext({ fn: () => 'bad' });
      expect(ctx.evaluateCondition('typeof fn === "undefined"')).toBe(true);
    });
  });
});
