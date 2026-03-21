import type { IntegrationId } from '@/types/integration';
import { logger } from '@/lib/logger';

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
  const secret = process.env[envKey] || process.env.WEBHOOK_SECRET || '';

  if (!secret) {
    const env = process.env.CADRE_ENV || 'local';
    if (env === 'prod' || env === 'staging') {
      throw new Error(`Webhook secret not configured for ${integrationId} in ${env}. Set ${envKey} or WEBHOOK_SECRET.`);
    }
    logger.warn('Webhook secret not configured, skipping verification', { integrationId, env });
  }

  return secret;
}
