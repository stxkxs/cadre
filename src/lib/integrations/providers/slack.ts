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

export class SlackIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'slack';
  readonly name = 'Slack';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'slack')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.SLACK_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(','),
      state,
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', Accept: 'application/json' },
      body: new URLSearchParams({
        client_id: process.env.SLACK_CLIENT_ID || '',
        client_secret: process.env.SLACK_CLIENT_SECRET || '',
        code,
        redirect_uri: redirectUri,
      }),
    });
    const data = await response.json();
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      metadata: { teamId: data.team?.id, teamName: data.team?.name, botUserId: data.bot_user_id },
    };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    if (!credentials.refreshToken) throw new Error('No refresh token available');
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', Accept: 'application/json' },
      body: new URLSearchParams({
        client_id: process.env.SLACK_CLIENT_ID || '',
        client_secret: process.env.SLACK_CLIENT_SECRET || '',
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }),
    });
    const data = await response.json();
    return { accessToken: data.access_token, refreshToken: data.refresh_token };
  }

  getActions(): IntegrationAction[] {
    return [
      { id: 'post-message', name: 'Post Message', direction: 'write', inputSchema: { channel: 'string', text: 'string', blocks: 'array' }, outputSchema: { message: 'object', ts: 'string' } },
      { id: 'read-history', name: 'Read History', direction: 'read', inputSchema: { channel: 'string', limit: 'number', oldest: 'string', latest: 'string' }, outputSchema: { messages: 'array', hasMore: 'boolean' } },
      { id: 'list-channels', name: 'List Channels', direction: 'read', inputSchema: { limit: 'number', cursor: 'string' }, outputSchema: { channels: 'array', nextCursor: 'string' } },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      'Content-Type': 'application/json; charset=utf-8',
    };

    switch (action.actionId) {
      case 'post-message': {
        const { channel, text, blocks } = action.params as { channel: string; text: string; blocks?: unknown[] };
        const res = await fetch('https://slack.com/api/chat.postMessage', {
          method: 'POST',
          headers,
          body: JSON.stringify({ channel, text, ...(blocks ? { blocks } : {}) }),
        });
        const data = await res.json();
        if (!data.ok) return { success: false, data: null, error: data.error };
        return { success: true, data: { message: data.message, ts: data.ts } };
      }
      case 'read-history': {
        const { channel, limit = 100, oldest, latest } = action.params as { channel: string; limit?: number; oldest?: string; latest?: string };
        const params = new URLSearchParams({ channel, limit: String(limit) });
        if (oldest) params.set('oldest', oldest);
        if (latest) params.set('latest', latest);
        const res = await fetch(`https://slack.com/api/conversations.history?${params}`, { headers });
        const data = await res.json();
        if (!data.ok) return { success: false, data: null, error: data.error };
        return { success: true, data: { messages: data.messages || [], hasMore: data.has_more ?? false } };
      }
      case 'list-channels': {
        const { limit = 200, cursor } = action.params as { limit?: number; cursor?: string };
        const params = new URLSearchParams({ limit: String(limit) });
        if (cursor) params.set('cursor', cursor);
        const res = await fetch(`https://slack.com/api/conversations.list?${params}`, { headers });
        const data = await res.json();
        if (!data.ok) return { success: false, data: null, error: data.error };
        return { success: true, data: { channels: data.channels || [], nextCursor: data.response_metadata?.next_cursor ?? '' } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['x-slack-signature'];
    const timestamp = payload.headers['x-slack-request-timestamp'];
    if (!signature || !timestamp) return false;
    const signingPayload = `v0:${timestamp}:${payload.body}`;
    return this.verifyHmacSha256(signingPayload, signature, secret, 'v0=');
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body);
    return {
      integrationId: 'slack',
      eventType: body.event?.type || body.type || 'unknown',
      payload: body,
      sourceId: body.team_id,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://slack.com/api/auth.test', {
        method: 'POST',
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      const data = await res.json();
      return res.ok && data.ok === true;
    } catch {
      return false;
    }
  }
}
