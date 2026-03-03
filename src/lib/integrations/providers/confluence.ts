import { BaseIntegration } from '../base';
import { INTEGRATION_CONFIGS } from '@/types/integration-configs';
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

export class ConfluenceIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'confluence';
  readonly name = 'Confluence';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'confluence')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.CONFLUENCE_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(' '),
      state,
      response_type: 'code',
      prompt: 'consent',
      audience: 'api.atlassian.com',
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: process.env.CONFLUENCE_CLIENT_ID,
        client_secret: process.env.CONFLUENCE_CLIENT_SECRET,
        grant_type: 'authorization_code',
        code,
        redirect_uri: redirectUri,
      }),
    });
    const data = await response.json();

    // Fetch accessible resources to obtain cloudId
    const resourcesRes = await fetch('https://api.atlassian.com/oauth/token/accessible-resources', {
      headers: { Authorization: `Bearer ${data.access_token}`, Accept: 'application/json' },
    });
    const resources = await resourcesRes.json();
    const cloudId = Array.isArray(resources) && resources.length > 0 ? resources[0].id : undefined;

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : undefined,
      metadata: { cloudId },
    };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    if (!credentials.refreshToken) throw new Error('No refresh token available');
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: process.env.CONFLUENCE_CLIENT_ID,
        client_secret: process.env.CONFLUENCE_CLIENT_SECRET,
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }),
    });
    const data = await response.json();
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token ?? credentials.refreshToken,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : credentials.tokenExpiresAt,
      metadata: credentials.metadata,
    };
  }

  getActions(): IntegrationAction[] {
    return [
      {
        id: 'read-page',
        name: 'Read Page',
        direction: 'read',
        inputSchema: { pageId: 'string' },
        outputSchema: { page: 'object' },
      },
      {
        id: 'create-page',
        name: 'Create Page',
        direction: 'write',
        inputSchema: { spaceKey: 'string', title: 'string', body: 'string', parentId: 'string' },
        outputSchema: { page: 'object' },
      },
      {
        id: 'update-page',
        name: 'Update Page',
        direction: 'write',
        inputSchema: { pageId: 'string', title: 'string', body: 'string', version: 'number' },
        outputSchema: { page: 'object' },
      },
      {
        id: 'search-pages',
        name: 'Search Pages',
        direction: 'read',
        inputSchema: { query: 'string', spaceKey: 'string', limit: 'number' },
        outputSchema: { results: 'array' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const cloudId = credentials.metadata?.cloudId as string | undefined;
    if (!cloudId) {
      return { success: false, data: null, error: 'Missing cloudId in credentials metadata' };
    }

    const baseUrl = `https://api.atlassian.com/ex/confluence/${cloudId}/wiki/api/v2`;
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      Accept: 'application/json',
      'Content-Type': 'application/json',
    };

    switch (action.actionId) {
      case 'read-page': {
        const { pageId } = action.params as { pageId: string };
        const res = await fetch(`${baseUrl}/pages/${pageId}?body-format=storage`, { headers });
        const page = await res.json();
        if (!res.ok) return { success: false, data: null, error: page.message ?? 'Failed to read page' };
        return { success: true, data: { page } };
      }
      case 'create-page': {
        const { spaceKey, title, body, parentId } = action.params as {
          spaceKey: string;
          title: string;
          body: string;
          parentId?: string;
        };
        const payload: Record<string, unknown> = {
          spaceId: spaceKey,
          status: 'current',
          title,
          body: { representation: 'storage', value: body },
        };
        if (parentId) payload.parentId = parentId;
        const res = await fetch(`${baseUrl}/pages`, {
          method: 'POST',
          headers,
          body: JSON.stringify(payload),
        });
        const page = await res.json();
        if (!res.ok) return { success: false, data: null, error: page.message ?? 'Failed to create page' };
        return { success: true, data: { page } };
      }
      case 'update-page': {
        const { pageId, title, body, version } = action.params as {
          pageId: string;
          title: string;
          body: string;
          version: number;
        };
        const res = await fetch(`${baseUrl}/pages/${pageId}`, {
          method: 'PUT',
          headers,
          body: JSON.stringify({
            id: pageId,
            status: 'current',
            title,
            body: { representation: 'storage', value: body },
            version: { number: version },
          }),
        });
        const page = await res.json();
        if (!res.ok) return { success: false, data: null, error: page.message ?? 'Failed to update page' };
        return { success: true, data: { page } };
      }
      case 'search-pages': {
        const { query, spaceKey, limit = 25 } = action.params as {
          query: string;
          spaceKey?: string;
          limit?: number;
        };
        const params = new URLSearchParams({ title: query, limit: String(limit) });
        if (spaceKey) params.set('space-key', spaceKey);
        const res = await fetch(`${baseUrl}/pages?${params.toString()}`, { headers });
        const data = await res.json();
        if (!res.ok) return { success: false, data: null, error: data.message ?? 'Failed to search pages' };
        return { success: true, data: { results: data.results ?? [] } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(_payload: WebhookPayload, _secret: string): boolean {
    return false;
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = payload.body ? (() => { try { return JSON.parse(payload.body); } catch { return {}; } })() : {};
    return {
      integrationId: 'confluence',
      eventType: (body as Record<string, unknown>).eventType as string ?? 'unknown',
      payload: body as Record<string, unknown>,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://api.atlassian.com/oauth/token/accessible-resources', {
        headers: {
          Authorization: `Bearer ${credentials.accessToken}`,
          Accept: 'application/json',
        },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
