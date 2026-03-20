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

export class GitHubIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'github';
  readonly name = 'GitHub';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'github')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.GITHUB_INTEGRATION_CLIENT_ID || '',
      redirect_uri: redirectUri,
      scope: this.config.oauth.scopes.join(' '),
      state,
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        client_id: process.env.GITHUB_INTEGRATION_CLIENT_ID,
        client_secret: process.env.GITHUB_INTEGRATION_CLIENT_SECRET,
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
        client_id: process.env.GITHUB_INTEGRATION_CLIENT_ID,
        client_secret: process.env.GITHUB_INTEGRATION_CLIENT_SECRET,
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }),
    });
    const data = await response.json();
    return { accessToken: data.access_token, refreshToken: data.refresh_token };
  }

  getActions(): IntegrationAction[] {
    return [
      { id: 'read-issues', name: 'Read Issues', direction: 'read', inputSchema: { repo: 'string', state: 'string' }, outputSchema: { issues: 'array' } },
      { id: 'create-issue', name: 'Create Issue', direction: 'write', inputSchema: { repo: 'string', title: 'string', body: 'string' }, outputSchema: { issue: 'object' } },
      { id: 'read-prs', name: 'Read Pull Requests', direction: 'read', inputSchema: { repo: 'string', state: 'string' }, outputSchema: { pullRequests: 'array' } },
      { id: 'create-comment', name: 'Create Comment', direction: 'write', inputSchema: { repo: 'string', issueNumber: 'number', body: 'string' }, outputSchema: { comment: 'object' } },
      { id: 'read-file', name: 'Read File', direction: 'read', inputSchema: { repo: 'string', path: 'string', ref: 'string' }, outputSchema: { content: 'string' } },
      { id: 'search-code', name: 'Search Code', direction: 'read', inputSchema: { query: 'string' }, outputSchema: { results: 'array' } },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      Accept: 'application/vnd.github+json',
      'X-GitHub-Api-Version': '2022-11-28',
    };

    switch (action.actionId) {
      case 'read-issues': {
        const { repo, state = 'open' } = action.params as { repo: string; state?: string };
        const [owner, name] = repo.split('/');
        const res = await fetch(`https://api.github.com/repos/${owner}/${name}/issues?state=${state}`, { headers });
        const issues = await res.json();
        return { success: true, data: { issues } };
      }
      case 'create-issue': {
        const { repo, title, body } = action.params as { repo: string; title: string; body: string };
        const [owner, name] = repo.split('/');
        const res = await fetch(`https://api.github.com/repos/${owner}/${name}/issues`, {
          method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' },
          body: JSON.stringify({ title, body }),
        });
        const issue = await res.json();
        return { success: true, data: { issue } };
      }
      case 'read-prs': {
        const { repo, state = 'open' } = action.params as { repo: string; state?: string };
        const [owner, name] = repo.split('/');
        const res = await fetch(`https://api.github.com/repos/${owner}/${name}/pulls?state=${state}`, { headers });
        const pullRequests = await res.json();
        return { success: true, data: { pullRequests } };
      }
      case 'create-comment': {
        const { repo, issueNumber, body } = action.params as { repo: string; issueNumber: number; body: string };
        const [owner, name] = repo.split('/');
        const res = await fetch(`https://api.github.com/repos/${owner}/${name}/issues/${issueNumber}/comments`, {
          method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' },
          body: JSON.stringify({ body }),
        });
        const comment = await res.json();
        return { success: true, data: { comment } };
      }
      case 'read-file': {
        const { repo, path, ref = 'main' } = action.params as { repo: string; path: string; ref?: string };
        const [owner, name] = repo.split('/');
        const res = await fetch(`https://api.github.com/repos/${owner}/${name}/contents/${path}?ref=${ref}`, { headers });
        const file = await res.json();
        const content = file.content ? Buffer.from(file.content, 'base64').toString('utf-8') : '';
        return { success: true, data: { content, sha: file.sha } };
      }
      case 'search-code': {
        const { query } = action.params as { query: string };
        const res = await fetch(`https://api.github.com/search/code?q=${encodeURIComponent(query)}`, { headers });
        const results = await res.json();
        return { success: true, data: { results: results.items || [] } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['x-hub-signature-256'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret, 'sha256=');
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body);
    return {
      integrationId: 'github',
      eventType: payload.headers['x-github-event'] || 'unknown',
      payload: body,
      sourceId: body.repository?.full_name,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch('https://api.github.com/user', {
        headers: { Authorization: `Bearer ${credentials.accessToken}` },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
