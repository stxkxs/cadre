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

export class FigmaIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'figma';
  readonly name = 'Figma';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'figma')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.FIGMA_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(' '),
      state,
      response_type: 'code',
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.FIGMA_CLIENT_ID,
        client_secret: process.env.FIGMA_CLIENT_SECRET,
        redirect_uri: redirectUri,
        code,
        grant_type: 'authorization_code',
      }),
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
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.FIGMA_CLIENT_ID,
        client_secret: process.env.FIGMA_CLIENT_SECRET,
        refresh_token: credentials.refreshToken,
        grant_type: 'refresh_token',
      }),
    });
    const data = await response.json();
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token ?? credentials.refreshToken,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : undefined,
    };
  }

  getActions(): IntegrationAction[] {
    return [
      {
        id: 'read-file',
        name: 'Read File',
        direction: 'read',
        inputSchema: { key: 'string' },
        outputSchema: { document: 'object', components: 'object', styles: 'object' },
      },
      {
        id: 'read-comments',
        name: 'Read Comments',
        direction: 'read',
        inputSchema: { key: 'string' },
        outputSchema: { comments: 'array' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
    };

    switch (action.actionId) {
      case 'read-file': {
        const { key } = action.params as { key: string };
        const res = await fetch(`https://api.figma.com/v1/files/${key}`, { headers });
        if (!res.ok) {
          const err = await res.json().catch(() => ({}));
          return { success: false, data: null, error: (err as { message?: string }).message || `HTTP ${res.status}` };
        }
        const data = await res.json();
        return { success: true, data };
      }
      case 'read-comments': {
        const { key } = action.params as { key: string };
        const res = await fetch(`https://api.figma.com/v1/files/${key}/comments`, { headers });
        if (!res.ok) {
          const err = await res.json().catch(() => ({}));
          return { success: false, data: null, error: (err as { message?: string }).message || `HTTP ${res.status}` };
        }
        const data = await res.json();
        return { success: true, data: { comments: (data as { comments?: unknown[] }).comments ?? [] } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    // Figma uses a passcode-based verification: the passcode is included in the
    // JSON body under the "passcode" key rather than a header signature.
    try {
      const body = JSON.parse(payload.body) as { passcode?: string };
      return body.passcode === secret;
    } catch {
      return false;
    }
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body) as Record<string, unknown>;
    return {
      integrationId: 'figma',
      eventType: (body.event_type as string | undefined) ?? 'unknown',
      payload: body,
      sourceId: (body.file_key as string | undefined) ?? undefined,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://api.figma.com/v1/me', {
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
