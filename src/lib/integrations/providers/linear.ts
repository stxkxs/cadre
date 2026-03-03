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

const LINEAR_API_URL = 'https://api.linear.app/graphql';

export class LinearIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'linear';
  readonly name = 'Linear';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'linear')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.LINEAR_CLIENT_ID || '',
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
        client_id: process.env.LINEAR_CLIENT_ID,
        client_secret: process.env.LINEAR_CLIENT_SECRET,
        code,
        redirect_uri: redirectUri,
        grant_type: 'authorization_code',
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
        client_id: process.env.LINEAR_CLIENT_ID,
        client_secret: process.env.LINEAR_CLIENT_SECRET,
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
      }),
    });
    const data = await response.json();
    return { accessToken: data.access_token, refreshToken: data.refresh_token };
  }

  getActions(): IntegrationAction[] {
    return [
      { id: 'read-issues', name: 'Read Issues', direction: 'read', inputSchema: { teamId: 'string', state: 'string', limit: 'number' }, outputSchema: { issues: 'array' } },
      { id: 'create-issue', name: 'Create Issue', direction: 'write', inputSchema: { teamId: 'string', title: 'string', description: 'string', priority: 'number' }, outputSchema: { issue: 'object' } },
      { id: 'update-issue', name: 'Update Issue', direction: 'write', inputSchema: { issueId: 'string', title: 'string', description: 'string', stateId: 'string', priority: 'number' }, outputSchema: { issue: 'object' } },
      { id: 'search-issues', name: 'Search Issues', direction: 'read', inputSchema: { query: 'string', limit: 'number' }, outputSchema: { issues: 'array' } },
    ];
  }

  private async graphql(query: string, variables: Record<string, unknown>, accessToken: string): Promise<Record<string, unknown>> {
    const res = await fetch(LINEAR_API_URL, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${accessToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ query, variables }),
    });
    const json = await res.json();
    return json as Record<string, unknown>;
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const { accessToken } = credentials;

    switch (action.actionId) {
      case 'read-issues': {
        const { teamId, state, limit = 50 } = action.params as { teamId?: string; state?: string; limit?: number };
        const filter: Record<string, unknown> = {};
        if (teamId) filter.team = { id: { eq: teamId } };
        if (state) filter.state = { name: { eq: state } };
        const query = `
          query ReadIssues($filter: IssueFilter, $first: Int) {
            issues(filter: $filter, first: $first) {
              nodes {
                id
                identifier
                title
                description
                priority
                state { id name }
                team { id name }
                assignee { id name email }
                createdAt
                updatedAt
                url
              }
            }
          }
        `;
        const data = await this.graphql(query, { filter, first: limit }, accessToken);
        const issues = (data.data as Record<string, unknown>)?.issues as { nodes: unknown[] } | undefined;
        return { success: true, data: { issues: issues?.nodes ?? [] } };
      }

      case 'create-issue': {
        const { teamId, title, description, priority } = action.params as { teamId: string; title: string; description?: string; priority?: number };
        const query = `
          mutation CreateIssue($input: IssueCreateInput!) {
            issueCreate(input: $input) {
              success
              issue {
                id
                identifier
                title
                description
                priority
                state { id name }
                url
              }
            }
          }
        `;
        const input: Record<string, unknown> = { teamId, title };
        if (description !== undefined) input.description = description;
        if (priority !== undefined) input.priority = priority;
        const data = await this.graphql(query, { input }, accessToken);
        const result = (data.data as Record<string, unknown>)?.issueCreate as { success: boolean; issue: unknown } | undefined;
        return { success: result?.success ?? false, data: { issue: result?.issue ?? null } };
      }

      case 'update-issue': {
        const { issueId, title, description, stateId, priority } = action.params as { issueId: string; title?: string; description?: string; stateId?: string; priority?: number };
        const query = `
          mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
            issueUpdate(id: $id, input: $input) {
              success
              issue {
                id
                identifier
                title
                description
                priority
                state { id name }
                url
              }
            }
          }
        `;
        const input: Record<string, unknown> = {};
        if (title !== undefined) input.title = title;
        if (description !== undefined) input.description = description;
        if (stateId !== undefined) input.stateId = stateId;
        if (priority !== undefined) input.priority = priority;
        const data = await this.graphql(query, { id: issueId, input }, accessToken);
        const result = (data.data as Record<string, unknown>)?.issueUpdate as { success: boolean; issue: unknown } | undefined;
        return { success: result?.success ?? false, data: { issue: result?.issue ?? null } };
      }

      case 'search-issues': {
        const { query: searchQuery, limit = 50 } = action.params as { query: string; limit?: number };
        const query = `
          query SearchIssues($query: String!, $first: Int) {
            issueSearch(query: $query, first: $first) {
              nodes {
                id
                identifier
                title
                description
                priority
                state { id name }
                team { id name }
                assignee { id name email }
                createdAt
                updatedAt
                url
              }
            }
          }
        `;
        const data = await this.graphql(query, { query: searchQuery, first: limit }, accessToken);
        const issues = (data.data as Record<string, unknown>)?.issueSearch as { nodes: unknown[] } | undefined;
        return { success: true, data: { issues: issues?.nodes ?? [] } };
      }

      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean {
    const signature = payload.headers['linear-signature'];
    if (!signature) return false;
    return this.verifyHmacSha256(payload.body, signature, secret);
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body);
    return {
      integrationId: 'linear',
      eventType: body.type || 'unknown',
      payload: body,
      sourceId: body.organizationId,
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const data = await this.graphql('{ viewer { id } }', {}, credentials.accessToken);
      return !!(data.data as Record<string, unknown>)?.viewer;
    } catch {
      return false;
    }
  }
}
