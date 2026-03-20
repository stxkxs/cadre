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

export class GoogleDocsIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'google-docs';
  readonly name = 'Google Docs';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'google-docs')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.GOOGLE_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(' '),
      state,
      response_type: 'code',
      access_type: 'offline',
      prompt: 'consent',
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        client_id: process.env.GOOGLE_CLIENT_ID || '',
        client_secret: process.env.GOOGLE_CLIENT_SECRET || '',
        grant_type: 'authorization_code',
        code,
        redirect_uri: redirectUri,
      }).toString(),
    });
    const data = await response.json();
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : undefined,
    };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    if (!credentials.refreshToken) throw new Error('No refresh token available');
    const response = await fetch('https://oauth2.googleapis.com/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        client_id: process.env.GOOGLE_CLIENT_ID || '',
        client_secret: process.env.GOOGLE_CLIENT_SECRET || '',
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }).toString(),
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
        id: 'read-document',
        name: 'Read Document',
        direction: 'read',
        inputSchema: { documentId: 'string' },
        outputSchema: { document: 'object' },
      },
      {
        id: 'create-document',
        name: 'Create Document',
        direction: 'write',
        inputSchema: { title: 'string', content: 'string', folderId: 'string' },
        outputSchema: { document: 'object' },
      },
      {
        id: 'update-document',
        name: 'Update Document',
        direction: 'write',
        inputSchema: { documentId: 'string', requests: 'array' },
        outputSchema: { result: 'object' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      'Content-Type': 'application/json',
    };

    switch (action.actionId) {
      case 'read-document': {
        const { documentId } = action.params as { documentId: string };
        const res = await fetch(`https://docs.googleapis.com/v1/documents/${documentId}`, { headers });
        const document = await res.json();
        if (!res.ok) return { success: false, data: null, error: document.error?.message ?? 'Failed to read document' };
        return { success: true, data: { document } };
      }
      case 'create-document': {
        const { title, content, folderId } = action.params as {
          title: string;
          content?: string;
          folderId?: string;
        };

        // Create the document via Docs API
        const createRes = await fetch('https://docs.googleapis.com/v1/documents', {
          method: 'POST',
          headers,
          body: JSON.stringify({ title }),
        });
        const document = await createRes.json();
        if (!createRes.ok) return { success: false, data: null, error: document.error?.message ?? 'Failed to create document' };

        // If initial content was supplied, insert it
        if (content) {
          await fetch(`https://docs.googleapis.com/v1/documents/${document.documentId}:batchUpdate`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
              requests: [{ insertText: { location: { index: 1 }, text: content } }],
            }),
          });
        }

        // Move to folder if specified (Drive API)
        if (folderId) {
          const metaRes = await fetch(
            `https://www.googleapis.com/drive/v3/files/${document.documentId}?addParents=${folderId}&fields=id,parents`,
            { method: 'PATCH', headers },
          );
          if (!metaRes.ok) {
            // Non-fatal: document was created, just couldn't move it
          }
        }

        return { success: true, data: { document } };
      }
      case 'update-document': {
        const { documentId, requests } = action.params as {
          documentId: string;
          requests: unknown[];
        };
        const res = await fetch(`https://docs.googleapis.com/v1/documents/${documentId}:batchUpdate`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ requests }),
        });
        const result = await res.json();
        if (!res.ok) return { success: false, data: null, error: result.error?.message ?? 'Failed to update document' };
        return { success: true, data: { result } };
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
      integrationId: 'google-docs',
      eventType: (body as Record<string, unknown>).kind as string ?? 'unknown',
      payload: body as Record<string, unknown>,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://www.googleapis.com/oauth2/v1/userinfo', {
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
