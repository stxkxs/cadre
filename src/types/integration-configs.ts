import type { IntegrationConfig } from './integration';

export const INTEGRATION_CONFIGS: IntegrationConfig[] = [
  {
    id: 'github',
    name: 'GitHub',
    icon: 'github',
    color: '#24292e',
    capabilities: ['read', 'write', 'search', 'webhook'],
    oauth: {
      authorizationUrl: 'https://github.com/login/oauth/authorize',
      tokenUrl: 'https://github.com/login/oauth/access_token',
      scopes: ['repo', 'read:org', 'read:user'],
    },
    webhook: {
      signatureHeader: 'x-hub-signature-256',
      signatureMethod: 'hmac-sha256',
      signaturePrefix: 'sha256=',
    },
  },
  {
    id: 'linear',
    name: 'Linear',
    icon: 'target',
    color: '#5E6AD2',
    capabilities: ['read', 'write', 'search', 'webhook'],
    oauth: {
      authorizationUrl: 'https://linear.app/oauth/authorize',
      tokenUrl: 'https://api.linear.app/oauth/token',
      scopes: ['read', 'write', 'issues:create'],
    },
    webhook: {
      signatureHeader: 'linear-signature',
      signatureMethod: 'hmac-sha256',
    },
  },
  {
    id: 'notion',
    name: 'Notion',
    icon: 'file-text',
    color: '#000000',
    capabilities: ['read', 'write', 'search'],
    oauth: {
      authorizationUrl: 'https://api.notion.com/v1/oauth/authorize',
      tokenUrl: 'https://api.notion.com/v1/oauth/token',
      scopes: [],
      responseType: 'code',
    },
  },
  {
    id: 'slack',
    name: 'Slack',
    icon: 'message-square',
    color: '#4A154B',
    capabilities: ['read', 'write', 'webhook'],
    oauth: {
      authorizationUrl: 'https://slack.com/oauth/v2/authorize',
      tokenUrl: 'https://slack.com/api/oauth.v2.access',
      scopes: ['chat:write', 'channels:read', 'channels:history'],
    },
    webhook: {
      signatureHeader: 'x-slack-signature',
      signatureMethod: 'hmac-sha256',
      signaturePrefix: 'v0=',
    },
  },
  {
    id: 'figma',
    name: 'Figma',
    icon: 'pen-tool',
    color: '#F24E1E',
    capabilities: ['read', 'webhook'],
    oauth: {
      authorizationUrl: 'https://www.figma.com/oauth',
      tokenUrl: 'https://api.figma.com/v1/oauth/token',
      scopes: ['files:read'],
    },
    webhook: {
      signatureHeader: 'x-figma-signature',
      signatureMethod: 'passcode',
    },
  },
  {
    id: 'jira',
    name: 'Jira',
    icon: 'ticket',
    color: '#0052CC',
    capabilities: ['read', 'write', 'search', 'webhook'],
    oauth: {
      authorizationUrl: 'https://auth.atlassian.com/authorize',
      tokenUrl: 'https://auth.atlassian.com/oauth/token',
      scopes: ['read:jira-work', 'write:jira-work', 'read:jira-user'],
    },
    webhook: {
      signatureHeader: 'x-hub-signature',
      signatureMethod: 'hmac-sha256',
    },
  },
  {
    id: 'confluence',
    name: 'Confluence',
    icon: 'book-open',
    color: '#172B4D',
    capabilities: ['read', 'write', 'search'],
    oauth: {
      authorizationUrl: 'https://auth.atlassian.com/authorize',
      tokenUrl: 'https://auth.atlassian.com/oauth/token',
      scopes: ['read:confluence-content.all', 'write:confluence-content'],
    },
  },
  {
    id: 'google-docs',
    name: 'Google Docs',
    icon: 'file-edit',
    color: '#4285F4',
    capabilities: ['read', 'write'],
    oauth: {
      authorizationUrl: 'https://accounts.google.com/o/oauth2/v2/auth',
      tokenUrl: 'https://oauth2.googleapis.com/token',
      scopes: ['https://www.googleapis.com/auth/documents', 'https://www.googleapis.com/auth/drive.file'],
    },
  },
  {
    id: 'loom',
    name: 'Loom',
    icon: 'video',
    color: '#625DF5',
    capabilities: ['read', 'list'],
    oauth: {
      authorizationUrl: 'https://www.loom.com/oauth/authorize',
      tokenUrl: 'https://www.loom.com/oauth/token',
      scopes: ['content:read'],
    },
    webhook: {
      signatureHeader: 'loom-signature',
      signatureMethod: 'hmac-sha256',
    },
  },
  {
    id: 'coda',
    name: 'Coda',
    icon: 'table',
    color: '#F46A54',
    capabilities: ['read', 'write', 'list', 'webhook'],
    oauth: {
      authorizationUrl: 'https://coda.io/account/authorize',
      tokenUrl: 'https://coda.io/apis/v1/oauth/token',
      scopes: ['read', 'write'],
    },
    webhook: {
      signatureHeader: 'x-coda-signature',
      signatureMethod: 'hmac-sha256',
    },
  },
];
