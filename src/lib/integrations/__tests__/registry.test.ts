import { describe, it, expect } from 'vitest';
import { integrationRegistry } from '../registry';
import type { IntegrationId } from '@/types/integration';

describe('IntegrationRegistry', () => {
  const expectedIds: IntegrationId[] = [
    'github', 'linear', 'notion', 'slack', 'figma',
    'jira', 'confluence', 'google-docs', 'loom', 'coda',
  ];

  it('has all 10 integrations registered', () => {
    const ids = integrationRegistry.getIds();
    expect(ids).toHaveLength(10);
  });

  it.each(expectedIds)('has %s registered', (id) => {
    expect(integrationRegistry.has(id)).toBe(true);
  });

  it('get returns correct integration', () => {
    const github = integrationRegistry.get('github');
    expect(github.id).toBe('github');
    expect(github.name).toBeTruthy();
  });

  it('get throws for unknown integration', () => {
    expect(() => integrationRegistry.get('unknown' as IntegrationId)).toThrow('Unknown integration: unknown');
  });

  it('has returns false for unknown integration', () => {
    expect(integrationRegistry.has('nonexistent')).toBe(false);
  });

  it('getAll returns all integrations', () => {
    const all = integrationRegistry.getAll();
    expect(all).toHaveLength(10);
    const ids = all.map(i => i.id);
    for (const expected of expectedIds) {
      expect(ids).toContain(expected);
    }
  });

  it('getIds returns all integration IDs', () => {
    const ids = integrationRegistry.getIds();
    for (const expected of expectedIds) {
      expect(ids).toContain(expected);
    }
  });

  it('each integration has required properties', () => {
    for (const integration of integrationRegistry.getAll()) {
      expect(integration.id).toBeTruthy();
      expect(integration.name).toBeTruthy();
      expect(integration.config).toBeDefined();
      expect(integration.config.id).toBe(integration.id);
      expect(integration.config.capabilities).toBeInstanceOf(Array);
      expect(integration.config.oauth).toBeDefined();
    }
  });

  it('each integration has getActions method', () => {
    for (const integration of integrationRegistry.getAll()) {
      const actions = integration.getActions();
      expect(Array.isArray(actions)).toBe(true);
    }
  });
});
