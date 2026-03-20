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

const NOTION_API_BASE = 'https://api.notion.com/v1';
const NOTION_VERSION = '2022-06-28';

export class NotionIntegration extends BaseIntegration {
  readonly id: IntegrationId = 'notion';
  readonly name = 'Notion';
  readonly config: IntegrationConfig = INTEGRATION_CONFIGS.find(c => c.id === 'notion')!;

  getAuthorizationUrl(state: string, redirectUri: string): string {
    return this.buildOAuthUrl(this.config.oauth.authorizationUrl, {
      client_id: process.env.NOTION_CLIENT_ID || '',
      redirect_uri: redirectUri,
      response_type: this.config.oauth.responseType || 'code',
      owner: 'user',
      state,
    });
  }

  async exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials> {
    const credentials = Buffer.from(
      `${process.env.NOTION_CLIENT_ID}:${process.env.NOTION_CLIENT_SECRET}`
    ).toString('base64');

    const response = await fetch(this.config.oauth.tokenUrl, {
      method: 'POST',
      headers: {
        Authorization: `Basic ${credentials}`,
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify({
        grant_type: 'authorization_code',
        code,
        redirect_uri: redirectUri,
      }),
    });
    const data = await response.json();
    return {
      accessToken: data.access_token,
      metadata: {
        workspaceId: data.workspace_id,
        workspaceName: data.workspace_name,
        botId: data.bot_id,
      },
    };
  }

  async refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    // Notion tokens do not expire — return existing credentials unchanged
    return credentials;
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
        inputSchema: { parentId: 'string', parentType: 'string', title: 'string', content: 'string' },
        outputSchema: { page: 'object' },
      },
      {
        id: 'query-database',
        name: 'Query Database',
        direction: 'read',
        inputSchema: { databaseId: 'string', filter: 'object', sorts: 'array' },
        outputSchema: { results: 'array' },
      },
      {
        id: 'search',
        name: 'Search',
        direction: 'read',
        inputSchema: { query: 'string', filter: 'object' },
        outputSchema: { results: 'array' },
      },
    ];
  }

  async execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    const headers = {
      Authorization: `Bearer ${credentials.accessToken}`,
      'Notion-Version': NOTION_VERSION,
      'Content-Type': 'application/json',
    };

    switch (action.actionId) {
      case 'read-page': {
        const { pageId } = action.params as { pageId: string };
        const res = await fetch(`${NOTION_API_BASE}/pages/${pageId}`, { headers });
        const page = await res.json();
        return { success: res.ok, data: { page } };
      }
      case 'create-page': {
        const { parentId, parentType = 'page_id', title, content } = action.params as {
          parentId: string;
          parentType?: string;
          title: string;
          content?: string;
        };
        const body: Record<string, unknown> = {
          parent: { [parentType]: parentId },
          properties: {
            title: {
              title: [{ type: 'text', text: { content: title } }],
            },
          },
        };
        if (content) {
          body.children = [
            {
              object: 'block',
              type: 'paragraph',
              paragraph: {
                rich_text: [{ type: 'text', text: { content } }],
              },
            },
          ];
        }
        const res = await fetch(`${NOTION_API_BASE}/pages`, {
          method: 'POST',
          headers,
          body: JSON.stringify(body),
        });
        const page = await res.json();
        return { success: res.ok, data: { page } };
      }
      case 'query-database': {
        const { databaseId, filter, sorts } = action.params as {
          databaseId: string;
          filter?: Record<string, unknown>;
          sorts?: unknown[];
        };
        const body: Record<string, unknown> = {};
        if (filter) body.filter = filter;
        if (sorts) body.sorts = sorts;
        const res = await fetch(`${NOTION_API_BASE}/databases/${databaseId}/query`, {
          method: 'POST',
          headers,
          body: JSON.stringify(body),
        });
        const data = await res.json();
        return { success: res.ok, data: { results: data.results || [] } };
      }
      case 'search': {
        const { query, filter } = action.params as { query: string; filter?: Record<string, unknown> };
        const body: Record<string, unknown> = { query };
        if (filter) body.filter = filter;
        const res = await fetch(`${NOTION_API_BASE}/search`, {
          method: 'POST',
          headers,
          body: JSON.stringify(body),
        });
        const data = await res.json();
        return { success: res.ok, data: { results: data.results || [] } };
      }
      default:
        return { success: false, data: null, error: `Unknown action: ${action.actionId}` };
    }
  }

  verifyWebhookSignature(_payload: WebhookPayload, _secret: string): boolean {
    // Notion does not support webhook signature verification
    return false;
  }

  parseWebhookEvent(payload: WebhookPayload): WebhookEvent {
    const body = JSON.parse(payload.body) as Record<string, unknown>;
    return {
      integrationId: 'notion',
      eventType: (body.type as string) || 'unknown',
      payload: body,
      sourceId: (body.workspace_id as string | undefined),
    };
  }

  async testConnection(credentials: IntegrationCredentials): Promise<boolean> {
    try {
      const res = await fetch(`${NOTION_API_BASE}/users/me`, {
        headers: {
          Authorization: `Bearer ${credentials.accessToken}`,
          'Notion-Version': NOTION_VERSION,
        },
      });
      return res.ok;
    } catch {
      return false;
    }
  }
}
