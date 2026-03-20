import { describe, it, expect, vi } from 'vitest';
import { createHmac } from 'crypto';
import { BaseIntegration } from '../base';
import type {
  IntegrationId,
  IntegrationConfig,
  IntegrationAction,
  IntegrationCredentials,
  IntegrationActionInput,
  IntegrationActionOutput,
  WebhookPayload,
  WebhookEvent,
} from '@/types/integration';

// Concrete stub subclass for testing abstract class methods
class StubIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'github';
  readonly name = 'Stub';
  readonly config: IntegrationConfig = {
    id: 'github',
    name: 'Stub',
    icon: 'stub',
    color: '#000',
    capabilities: ['read'],
    oauth: {
      authorizationUrl: 'https://example.com/auth',
      tokenUrl: 'https://example.com/token',
      scopes: ['repo'],
    },
  };

  executeFn = vi.fn<(action: IntegrationActionInput, credentials: IntegrationCredentials) => Promise<IntegrationActionOutput>>();
  refreshFn = vi.fn<(credentials: IntegrationCredentials) => Promise<IntegrationCredentials>>();

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl('https://example.com/auth', {
      client_id: 'test-client',
      state,
      redirect_uri: redirectUri,
      scope: 'repo',
    });
  }

  async exchangeCode(): Promise<IntegrationCredentials> {
    return { accessToken: 'token' };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    return this.refreshFn(credentials);
  }

  getActions(): IntegrationAction[] {
    return [];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    return this.executeFn(action, credentials);
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['x-hub-signature-256'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret, 'sha256=');
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    return {
      integrationId: 'github',
      eventType: payload.headers['x-github-event'] || 'unknown',
      payload: JSON.parse(payload.body),
    };
  }

  testConnection(): Promise<boolean> {
    return Promise.resolve(true);
  }
}

describe('BaseIntegration', () => {
  it('safeExecute returns result on success', async () => {
    const integration = new StubIntegration();
    integration.executeFn.mockResolvedValue({ success: true, data: { id: 1 } });

    const result = await integration.safeExecute(
      { actionId: 'test', params: {} },
      { accessToken: 'token' }
    );

    expect(result.success).toBe(true);
    expect(result.data).toEqual({ id: 1 });
  });

  it('safeExecute catches errors and returns failure', async () => {
    const integration = new StubIntegration();
    integration.executeFn.mockRejectedValue(new Error('API rate limit'));

    const result = await integration.safeExecute(
      { actionId: 'test', params: {} },
      { accessToken: 'token' }
    );

    expect(result.success).toBe(false);
    expect(result.error).toBe('API rate limit');
  });

  it('safeExecute handles non-Error exceptions', async () => {
    const integration = new StubIntegration();
    integration.executeFn.mockRejectedValue('string error');

    const result = await integration.safeExecute(
      { actionId: 'test', params: {} },
      { accessToken: 'token' }
    );

    expect(result.success).toBe(false);
    expect(result.error).toBe('Integration action failed');
  });

  it('safeRefreshToken returns refreshed credentials on success', async () => {
    const integration = new StubIntegration();
    const refreshed: IntegrationCredentials = { accessToken: 'new-token' };
    integration.refreshFn.mockResolvedValue(refreshed);

    const result = await integration.safeRefreshToken({ accessToken: 'old-token' });
    expect(result.accessToken).toBe('new-token');
  });

  it('safeRefreshToken returns original credentials on failure', async () => {
    const integration = new StubIntegration();
    integration.refreshFn.mockRejectedValue(new Error('refresh failed'));

    const original: IntegrationCredentials = { accessToken: 'old-token' };
    const result = await integration.safeRefreshToken(original);
    expect(result.accessToken).toBe('old-token');
  });

  it('buildOAuthUrl constructs correct URL with params', () => {
    const integration = new StubIntegration();
    const url = integration.getAuthorizationUrl('my-state', 'http://localhost/callback');

    const parsed = new URL(url);
    expect(parsed.origin).toBe('https://example.com');
    expect(parsed.pathname).toBe('/auth');
    expect(parsed.searchParams.get('client_id')).toBe('test-client');
    expect(parsed.searchParams.get('state')).toBe('my-state');
    expect(parsed.searchParams.get('redirect_uri')).toBe('http://localhost/callback');
    expect(parsed.searchParams.get('scope')).toBe('repo');
  });

  it('verifyHmacSha256 validates correct signature', () => {
    const integration = new StubIntegration();
    const secret = 'webhook-secret';
    const body = '{"action":"opened"}';
    const expected = createHmac('sha256', secret).update(body).digest('hex');

    const payload: WebhookPayload = {
      headers: { 'x-hub-signature-256': `sha256=${expected}` },
      body,
      rawBody: Buffer.from(body),
    };

    expect(integration.verifyWebhookSignature(payload, secret)).toBe(true);
  });

  it('verifyHmacSha256 rejects invalid signature', () => {
    const integration = new StubIntegration();
    const payload: WebhookPayload = {
      headers: { 'x-hub-signature-256': 'sha256=invalid' },
      body: '{"action":"opened"}',
      rawBody: Buffer.from('{"action":"opened"}'),
    };

    expect(() => integration.verifyWebhookSignature(payload, 'secret')).toThrow();
  });

  it('verifyWebhookSignature returns false when header missing', () => {
    const integration = new StubIntegration();
    const payload: WebhookPayload = {
      headers: {},
      body: '{}',
      rawBody: Buffer.from('{}'),
    };

    expect(integration.verifyWebhookSignature(payload, 'secret')).toBe(false);
  });
});
