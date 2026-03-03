import { createHmac, timingSafeEqual } from 'crypto';
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

export abstract class BaseIntegration {
  abstract readonly id: IntegrationId;
  abstract readonly name: string;
  abstract readonly config: IntegrationConfig;

  // OAuth
  abstract getAuthorizationUrl(state: string, redirectUri: string): string;
  abstract exchangeCode(code: string, redirectUri: string): Promise<IntegrationCredentials>;
  abstract refreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials>;

  // Actions
  abstract getActions(): IntegrationAction[];
  abstract execute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput>;

  // Webhooks
  abstract verifyWebhookSignature(payload: WebhookPayload, secret: string): boolean;
  abstract parseWebhookEvent(payload: WebhookPayload): WebhookEvent;

  // Health check
  abstract testConnection(credentials: IntegrationCredentials): Promise<boolean>;

  async safeExecute(action: IntegrationActionInput, credentials: IntegrationCredentials): Promise<IntegrationActionOutput> {
    try {
      return await this.execute(action, credentials);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Integration action failed';
      return { success: false, data: null, error: message };
    }
  }

  async safeRefreshToken(credentials: IntegrationCredentials): Promise<IntegrationCredentials> {
    try {
      return await this.refreshToken(credentials);
    } catch {
      return credentials;
    }
  }

  protected buildOAuthUrl(baseUrl: string, params: Record<string, string>): string {
    const url = new URL(baseUrl);
    for (const [key, value] of Object.entries(params)) {
      url.searchParams.set(key, value);
    }
    return url.toString();
  }

  protected verifyHmacSha256(payload: string, signature: string, secret: string, prefix?: string): boolean {
    const expected = createHmac('sha256', secret).update(payload).digest('hex');
    const expectedWithPrefix = prefix ? `${prefix}${expected}` : expected;
    return timingSafeEqual(
      Buffer.from(expectedWithPrefix),
      Buffer.from(signature)
    );
  }
}
