import { describe, it, expect } from 'vitest';
import { Graph } from '../graph';
import type { WorkflowNode, WorkflowEdge } from '../types';

function makeNode(id: string, type: WorkflowNode['type'] = 'agent', data: Partial<WorkflowNode['data']> = {}): WorkflowNode {
  return { id, type, position: { x: 0, y: 0 }, data: { label: id, ...data } };
}

function makeEdge(id: string, source: string, target: string, extra: Partial<WorkflowEdge> = {}): WorkflowEdge {
  return { id, source, target, ...extra };
}

describe('Graph', () => {
  describe('node/edge lookups', () => {
    it('returns a node by id', () => {
      const graph = new Graph([makeNode('a')], []);
      expect(graph.getNode('a')?.id).toBe('a');
    });

    it('returns undefined for unknown node', () => {
      const graph = new Graph([makeNode('a')], []);
      expect(graph.getNode('x')).toBeUndefined();
    });

    it('returns all nodes', () => {
      const nodes = [makeNode('a'), makeNode('b')];
      const graph = new Graph(nodes, []);
      expect(graph.getNodes()).toHaveLength(2);
    });

    it('returns outgoing edges for a node', () => {
      const edges = [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'a', 'c')];
      const graph = new Graph([makeNode('a'), makeNode('b'), makeNode('c')], edges);
      expect(graph.getOutgoingEdges('a')).toHaveLength(2);
    });

    it('returns incoming edges for a node', () => {
      const edges = [makeEdge('e1', 'a', 'c'), makeEdge('e2', 'b', 'c')];
      const graph = new Graph([makeNode('a'), makeNode('b'), makeNode('c')], edges);
      expect(graph.getIncomingEdges('c')).toHaveLength(2);
    });
  });

  describe('predecessors and successors', () => {
    it('returns successors of a node', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'a', 'c')],
      );
      expect(graph.getSuccessors('a')).toEqual(['b', 'c']);
    });

    it('returns predecessors of a node', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'c'), makeEdge('e2', 'b', 'c')],
      );
      expect(graph.getPredecessors('c')).toEqual(['a', 'b']);
    });

    it('returns empty arrays for isolated nodes', () => {
      const graph = new Graph([makeNode('a')], []);
      expect(graph.getSuccessors('a')).toEqual([]);
      expect(graph.getPredecessors('a')).toEqual([]);
    });
  });

  describe('start and end nodes', () => {
    it('finds start nodes (no incoming edges)', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'c')],
      );
      const starts = graph.getStartNodes();
      expect(starts).toHaveLength(1);
      expect(starts[0].id).toBe('a');
    });

    it('finds end nodes (no outgoing edges)', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'c')],
      );
      const ends = graph.getEndNodes();
      expect(ends).toHaveLength(1);
      expect(ends[0].id).toBe('c');
    });
  });

  describe('topological sort', () => {
    it('sorts a linear graph', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'c')],
      );
      const sorted = graph.topologicalSort();
      expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('b'));
      expect(sorted.indexOf('b')).toBeLessThan(sorted.indexOf('c'));
    });

    it('sorts a diamond graph', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c'), makeNode('d')],
        [
          makeEdge('e1', 'a', 'b'),
          makeEdge('e2', 'a', 'c'),
          makeEdge('e3', 'b', 'd'),
          makeEdge('e4', 'c', 'd'),
        ],
      );
      const sorted = graph.topologicalSort();
      expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('b'));
      expect(sorted.indexOf('a')).toBeLessThan(sorted.indexOf('c'));
      expect(sorted.indexOf('b')).toBeLessThan(sorted.indexOf('d'));
      expect(sorted.indexOf('c')).toBeLessThan(sorted.indexOf('d'));
    });

    it('throws on cycle', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'a')],
      );
      expect(() => graph.topologicalSort()).toThrow('Cycle detected');
    });
  });

  describe('validate', () => {
    it('validates a correct graph', () => {
      const graph = new Graph(
        [makeNode('input1', 'input'), makeNode('agent1', 'agent', { provider: 'anthropic' }), makeNode('output1', 'output')],
        [makeEdge('e1', 'input1', 'agent1'), makeEdge('e2', 'agent1', 'output1')],
      );
      const result = graph.validate();
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('reports empty graph', () => {
      const graph = new Graph([], []);
      const result = graph.validate();
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Workflow must have at least one node');
    });

    it('reports missing provider on agent node', () => {
      const graph = new Graph([makeNode('a', 'agent')], []);
      const result = graph.validate();
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('must have a provider'))).toBe(true);
    });

    it('reports self-loops', () => {
      const graph = new Graph(
        [makeNode('a', 'agent', { provider: 'anthropic' })],
        [makeEdge('e1', 'a', 'a')],
      );
      const result = graph.validate();
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('self-loop'))).toBe(true);
    });

    it('reports missing condition expression', () => {
      const graph = new Graph([makeNode('c', 'condition')], []);
      const result = graph.validate();
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('must have a condition expression'))).toBe(true);
    });

    it('reports cycle via validate', () => {
      const graph = new Graph(
        [makeNode('a', 'agent', { provider: 'anthropic' }), makeNode('b', 'agent', { provider: 'anthropic' })],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'b', 'a')],
      );
      const result = graph.validate();
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('cycle'))).toBe(true);
    });

    it('reports unknown edge source/target', () => {
      const graph = new Graph(
        [makeNode('a')],
        [makeEdge('e1', 'a', 'nonexistent')],
      );
      const result = graph.validate();
      expect(result.errors.some(e => e.includes('unknown target'))).toBe(true);
    });
  });

  describe('hasParallelBranches', () => {
    it('returns true when node has multiple successors', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b'), makeNode('c')],
        [makeEdge('e1', 'a', 'b'), makeEdge('e2', 'a', 'c')],
      );
      expect(graph.hasParallelBranches('a')).toBe(true);
    });

    it('returns false when node has single successor', () => {
      const graph = new Graph(
        [makeNode('a'), makeNode('b')],
        [makeEdge('e1', 'a', 'b')],
      );
      expect(graph.hasParallelBranches('a')).toBe(false);
    });
  });
});
