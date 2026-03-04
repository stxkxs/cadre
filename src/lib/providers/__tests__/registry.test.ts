import { describe, it, expect } from 'vitest';
import { providerRegistry } from '../registry';

describe('ProviderRegistry', () => {
  it('has all 4 providers registered', () => {
    const ids = providerRegistry.getIds();
    expect(ids).toContain('anthropic');
    expect(ids).toContain('openai');
    expect(ids).toContain('groq');
    expect(ids).toContain('claude-code');
    expect(ids).toHaveLength(4);
  });

  it('returns correct provider instances', () => {
    expect(providerRegistry.get('anthropic').name).toBe('Anthropic');
    expect(providerRegistry.get('openai').name).toBe('OpenAI');
    expect(providerRegistry.get('groq').name).toBe('Groq');
    expect(providerRegistry.get('claude-code').name).toBe('Claude Code');
  });

  it('throws for unknown provider', () => {
    expect(() => providerRegistry.get('unknown' as 'anthropic')).toThrow('Unknown provider: unknown');
  });

  it('has() returns true for registered providers', () => {
    expect(providerRegistry.has('anthropic')).toBe(true);
    expect(providerRegistry.has('openai')).toBe(true);
  });

  it('has() returns false for unregistered providers', () => {
    expect(providerRegistry.has('unknown')).toBe(false);
  });

  it('getAll returns all provider instances', () => {
    const all = providerRegistry.getAll();
    expect(all).toHaveLength(4);
  });
});
