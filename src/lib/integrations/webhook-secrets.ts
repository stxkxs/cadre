import type { IntegrationId } from '@/types/integration';

const envKeyMap: Record<IntegrationId, string> = {
  github: 'WEBHOOK_SECRET_GITHUB',
  linear: 'WEBHOOK_SECRET_LINEAR',
  notion: 'WEBHOOK_SECRET_NOTION',
  slack: 'WEBHOOK_SECRET_SLACK',
  figma: 'WEBHOOK_SECRET_FIGMA',
  jira: 'WEBHOOK_SECRET_JIRA',
  confluence: 'WEBHOOK_SECRET_CONFLUENCE',
  'google-docs': 'WEBHOOK_SECRET_GOOGLE_DOCS',
  loom: 'WEBHOOK_SECRET_LOOM',
  coda: 'WEBHOOK_SECRET_CODA',
};

export function getWebhookSecret(integrationId: IntegrationId): string {
  const envKey = envKeyMap[integrationId];
  return process.env[envKey] || process.env.WEBHOOK_SECRET || '';
}
