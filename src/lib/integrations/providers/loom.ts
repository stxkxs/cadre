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

export class LoomIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'loom';
  readonly name = 'Loom';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'loom')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.LOOM_CLIENT_ID || '',
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
        client_id: process.env.LOOM_CLIENT_ID,
        client_secret: process.env.LOOM_CLIENT_SECRET,
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
        client_id: process.env.LOOM_CLIENT_ID,
        client_secret: process.env.LOOM_CLIENT_SECRET,
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
        id: 'list-videos',
        name: 'List Videos',
        direction: 'read',
        inputSchema: { limit: 'number', offset: 'number' },
        outputSchema: { videos: 'array' },
      },
      {
        id: 'get-transcript',
        name: 'Get Transcript',
        direction: 'read',
        inputSchema: { videoId: 'string' },
        outputSchema: { transcript: 'array' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      Accept: 'application/json',
    };

    switch (action.actionId) {
      case 'list-videos': {
        const { limit = 25, offset = 0 } = action.params as { limit?: number; offset?: number };
        const res = await fetch(
          `https://developer.loom.com/api/v1/videos?limit=${limit}&offset=${offset}`,
          { headers }
        );
        const data = await res.json();
        return { success: res.ok, data: { videos: data.videos ?? data } };
      }
      case 'get-transcript': {
        const { videoId } = action.params as { videoId: string };
        const res = await fetch(
          `https://developer.loom.com/api/v1/videos/${videoId}/transcript`,
          { headers }
        );
        const data = await res.json();
        return { success: res.ok, data: { transcript: data.transcript ?? data } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['loom-signature'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret);
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body);
    return {
      integrationId: 'loom',
      eventType: body.event_type || body.type || 'unknown',
      payload: body,
      sourceId: body.video?.id ?? body.id,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://developer.loom.com/api/v1/me', {
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
