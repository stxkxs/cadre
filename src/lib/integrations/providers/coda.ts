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

const CODA_API_BASE = 'https://coda.io/apis/v1';

export class CodaIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'coda';
  readonly name = 'Coda';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'coda')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.CODA_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(' '),
      response_type: 'code',
      state,
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.CODA_CLIENT_ID,
        client_secret: process.env.CODA_CLIENT_SECRET,
        grant_type: 'authorization_code',
        code,
        redirect_uri: redirectUri,
      }),
    });
    const data = await response.json();
    return { accessToken: data.access_token, refreshToken: data.refresh_token };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    if (!credentials.refreshToken) throw new Error('No refresh token available');
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.CODA_CLIENT_ID,
        client_secret: process.env.CODA_CLIENT_SECRET,
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }),
    });
    const data = await response.json();
    return { accessToken: data.access_token, refreshToken: data.refresh_token };
  }

  getActions(): IntegrationAction[] {
    return [
      {
        id: 'read-doc',
        name: 'Read Doc',
        direction: 'read',
        inputSchema: { docId: 'string' },
        outputSchema: { doc: 'object' },
      },
      {
        id: 'list-rows',
        name: 'List Rows',
        direction: 'read',
        inputSchema: { docId: 'string', tableId: 'string', limit: 'number', pageToken: 'string' },
        outputSchema: { rows: 'array', nextPageToken: 'string' },
      },
      {
        id: 'insert-row',
        name: 'Insert Row',
        direction: 'write',
        inputSchema: { docId: 'string', tableId: 'string', cells: 'array' },
        outputSchema: { requestId: 'string', addedRowIds: 'array' },
      },
      {
        id: 'update-doc',
        name: 'Update Doc',
        direction: 'write',
        inputSchema: { docId: 'string', title: 'string' },
        outputSchema: { doc: 'object' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      Accept: 'application/json',
      'Content-Type': 'application/json',
    };

    switch (action.actionId) {
      case 'read-doc': {
        const { docId } = action.params as { docId: string };
        const res = await fetch(`${CODA_API_BASE}/docs/${docId}`, { headers });
        const doc = await res.json();
        return { success: res.ok, data: { doc } };
      }
      case 'list-rows': {
        const { docId, tableId, limit = 25, pageToken } = action.params as {
          docId: string;
          tableId: string;
          limit?: number;
          pageToken?: string;
        };
        const url = new URL(`${CODA_API_BASE}/docs/${docId}/tables/${tableId}/rows`);
        url.searchParams.set('limit', String(limit));
        if (pageToken) url.searchParams.set('pageToken', pageToken);
        const res = await fetch(url.toString(), { headers });
        const data = await res.json();
        return { success: res.ok, data: { rows: data.items ?? [], nextPageToken: data.nextPageToken } };
      }
      case 'insert-row': {
        const { docId, tableId, cells } = action.params as {
          docId: string;
          tableId: string;
          cells: Array<{ column: string; value: unknown }>;
        };
        const res = await fetch(`${CODA_API_BASE}/docs/${docId}/tables/${tableId}/rows`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ rows: [{ cells }] }),
        });
        const data = await res.json();
        return { success: res.ok, data: { requestId: data.requestId, addedRowIds: data.addedRowIds ?? [] } };
      }
      case 'update-doc': {
        const { docId, title } = action.params as { docId: string; title: string };
        const res = await fetch(`${CODA_API_BASE}/docs/${docId}`, {
          method: 'PATCH',
          headers,
          body: JSON.stringify({ title }),
        });
        const doc = await res.json();
        return { success: res.ok, data: { doc } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['x-coda-signature'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret);
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body);
    return {
      integrationId: 'coda',
      eventType: body.type || body.event || 'unknown',
      payload: body,
      sourceId: body.doc?.id ?? body.docId,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch(`${CODA_API_BASE}/whoami`, {
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
