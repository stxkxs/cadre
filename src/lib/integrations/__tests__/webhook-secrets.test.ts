import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('getWebhookSecret', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    vi.resetModules();
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  async function getWebhookSecretFn() {
    const mod = await import('../webhook-secrets');
    return mod.getWebhookSecret;
  }

  it('returns provider-specific secret when set', async () => {
    process.env.WEBHOOK_SECRET_GITHUB = 'github-specific-secret';
    process.env.CADRE_ENV = 'local';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(getWebhookSecret('github')).toBe('github-specific-secret');
  });

  it('falls back to generic WEBHOOK_SECRET', async () => {
    process.env.WEBHOOK_SECRET = 'generic-secret';
    process.env.CADRE_ENV = 'local';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(getWebhookSecret('linear')).toBe('generic-secret');
  });

  it('returns empty string in local env when no secret', async () => {
    delete process.env.WEBHOOK_SECRET;
    delete process.env.WEBHOOK_SECRET_SLACK;
    process.env.CADRE_ENV = 'local';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(getWebhookSecret('slack')).toBe('');
  });

  it('returns empty string in dev env when no secret', async () => {
    delete process.env.WEBHOOK_SECRET;
    delete process.env.WEBHOOK_SECRET_FIGMA;
    process.env.CADRE_ENV = 'dev';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(getWebhookSecret('figma')).toBe('');
  });

  it('throws in prod when no secret configured', async () => {
    delete process.env.WEBHOOK_SECRET;
    delete process.env.WEBHOOK_SECRET_GITHUB;
    process.env.CADRE_ENV = 'prod';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(() => getWebhookSecret('github')).toThrow('Webhook secret not configured');
  });

  it('throws in staging when no secret configured', async () => {
    delete process.env.WEBHOOK_SECRET;
    delete process.env.WEBHOOK_SECRET_LINEAR;
    process.env.CADRE_ENV = 'staging';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(() => getWebhookSecret('linear')).toThrow('Webhook secret not configured');
  });

  it('prefers provider-specific over generic', async () => {
    process.env.WEBHOOK_SECRET = 'generic';
    process.env.WEBHOOK_SECRET_GITHUB = 'specific';
    process.env.CADRE_ENV = 'local';
    const getWebhookSecret = await getWebhookSecretFn();
    expect(getWebhookSecret('github')).toBe('specific');
  });
});
