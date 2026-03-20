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

export class JiraIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'jira';
  readonly name = 'Jira';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'jira')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.JIRA_CLIENT_ID || '',
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
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.JIRA_CLIENT_ID,
        client_secret: process.env.JIRA_CLIENT_SECRET,
        code,
        redirect_uri: redirectUri,
        grant_type: 'authorization_code',
      }),
    });
    const data = await response.json() as {
      access_token: string;
      refresh_token?: string;
      expires_in?: number;
    };

    // Fetch accessible resources to get the cloudId up front and store it in metadata.
    const cloudId = await this.fetchCloudId(data.access_token);

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : undefined,
      metadata: cloudId ? { cloudId } : undefined,
    };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    if (!credentials.refreshToken) throw new Error('No refresh token available');
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.JIRA_CLIENT_ID,
        client_secret: process.env.JIRA_CLIENT_SECRET,
        refresh_token: credentials.refreshToken,
        grant_type: 'refresh_token',
      }),
    });
    const data = await response.json() as {
      access_token: string;
      refresh_token?: string;
      expires_in?: number;
    };
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token ?? credentials.refreshToken,
      tokenExpiresAt: data.expires_in ? new Date(Date.now() + data.expires_in * 1000) : undefined,
      metadata: credentials.metadata,
    };
  }

  getActions(): IntegrationAction[] {
    return [
      {
        id: 'read-issue',
        name: 'Read Issue',
        direction: 'read',
        inputSchema: { issueKey: 'string' },
        outputSchema: { issue: 'object' },
      },
      {
        id: 'create-issue',
        name: 'Create Issue',
        direction: 'write',
        inputSchema: { projectKey: 'string', summary: 'string', description: 'string', issueType: 'string' },
        outputSchema: { issue: 'object' },
      },
      {
        id: 'search-issues',
        name: 'Search Issues',
        direction: 'read',
        inputSchema: { jql: 'string', maxResults: 'number' },
        outputSchema: { issues: 'array', total: 'number' },
      },
      {
        id: 'add-comment',
        name: 'Add Comment',
        direction: 'write',
        inputSchema: { issueKey: 'string', body: 'string' },
        outputSchema: { comment: 'object' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const cloudId = credentials.metadata?.cloudId as string | undefined;
    if (!cloudId) {
      return { success: false, data: null, error: 'Missing cloudId in credentials metadata' };
    }

    const baseUrl = `https://api.atlassian.com/ex/jira/${cloudId}/rest/api/3`;
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      Accept: 'application/json',
      'Content-Type': 'application/json',
    };

    switch (action.actionId) {
      case 'read-issue': {
        const { issueKey } = action.params as { issueKey: string };
        const res = await fetch(`${baseUrl}/issue/${issueKey}`, { headers });
        if (!res.ok) {
          const err = await res.json().catch(() => ({})) as { message?: string };
          return { success: false, data: null, error: err.message || `HTTP ${res.status}` };
        }
        const issue = await res.json();
        return { success: true, data: { issue } };
      }
      case 'create-issue': {
        const { projectKey, summary, description, issueType = 'Task' } = action.params as {
          projectKey: string;
          summary: string;
          description: string;
          issueType?: string;
        };
        const res = await fetch(`${baseUrl}/issue`, {
          method: 'POST',
          headers,
          body: JSON.stringify({
            fields: {
              project: { key: projectKey },
              summary,
              description: {
                type: 'doc',
                version: 1,
                content: [{ type: 'paragraph', content: [{ type: 'text', text: description }] }],
              },
              issuetype: { name: issueType },
            },
          }),
        });
        if (!res.ok) {
          const err = await res.json().catch(() => ({})) as { message?: string };
          return { success: false, data: null, error: err.message || `HTTP ${res.status}` };
        }
        const issue = await res.json();
        return { success: true, data: { issue } };
      }
      case 'search-issues': {
        const { jql, maxResults = 50 } = action.params as { jql: string; maxResults?: number };
        const res = await fetch(`${baseUrl}/search`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ jql, maxResults }),
        });
        if (!res.ok) {
          const err = await res.json().catch(() => ({})) as { message?: string };
          return { success: false, data: null, error: err.message || `HTTP ${res.status}` };
        }
        const data = await res.json() as { issues?: unknown[]; total?: number };
        return { success: true, data: { issues: data.issues ?? [], total: data.total ?? 0 } };
      }
      case 'add-comment': {
        const { issueKey, body } = action.params as { issueKey: string; body: string };
        const res = await fetch(`${baseUrl}/issue/${issueKey}/comment`, {
          method: 'POST',
          headers,
          body: JSON.stringify({
            body: {
              type: 'doc',
              version: 1,
              content: [{ type: 'paragraph', content: [{ type: 'text', text: body }] }],
            },
          }),
        });
        if (!res.ok) {
          const err = await res.json().catch(() => ({})) as { message?: string };
          return { success: false, data: null, error: err.message || `HTTP ${res.status}` };
        }
        const comment = await res.json();
        return { success: true, data: { comment } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['x-hub-signature'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret, 'sha256=');
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body) as Record<string, unknown>;
    return {
      integrationId: 'jira',
      eventType: (body.webhookEvent as string | undefined) ?? 'unknown',
      payload: body,
      sourceId: ((body.issue as Record<string, unknown> | undefined)?.key as string | undefined) ?? undefined,
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

  // Fetch the first accessible Jira cloudId for the given access token.
  private async fetchCloudId(accessToken: string): Promise<string | null> {
    try {
      const res = await fetch('https://api.atlassian.com/oauth/token/accessible-resources', {
        headers: {
          Authorization: `Bearer ${accessToken}`,
          Accept: 'application/json',
        },
      });
      if (!res.ok) return null;
      const resources = await res.json() as Array<{ id: string; scopes?: string[] }>;
      return resources[0]?.id ?? null;
    } catch {
      return null;
    }
  }
}
