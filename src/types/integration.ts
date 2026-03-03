export type IntegrationId =
  | 'github'
  | 'linear'
  | 'notion'
  | 'slack'
  | 'figma'
  | 'jira'
  | 'confluence'
  | 'google-docs'
  | 'loom'
  | 'coda';

export type IntegrationCapability = 'read' | 'write' | 'list' | 'search' | 'webhook';

export interface OAuthConfig {
  authorizationUrl: string;
  tokenUrl: string;
  scopes: string[];
  responseType?: string;
}

export interface WebhookConfig {
  signatureHeader: string;
  signatureMethod: 'hmac-sha256' | 'bearer' | 'passcode';
  signaturePrefix?: string;
}

export interface IntegrationConfig {
  id: IntegrationId;
  name: string;
  icon: string;
  color: string;
  capabilities: IntegrationCapability[];
  oauth: OAuthConfig;
  webhook?: WebhookConfig;
}

export interface IntegrationAction {
  id: string;
  name: string;
  direction: 'read' | 'write';
  inputSchema: Record<string, unknown>;
  outputSchema: Record<string, unknown>;
}

export interface IntegrationCredentials {
  accessToken: string;
  refreshToken?: string;
  tokenExpiresAt?: Date;
  metadata?: Record<string, unknown>;
}

export interface IntegrationActionInput {
  actionId: string;
  params: Record<string, unknown>;
}

export interface IntegrationActionOutput {
  success: boolean;
  data: unknown;
  error?: string;
}

export interface WebhookPayload {
  headers: Record<string, string>;
  body: string;
  rawBody: Buffer;
}

export interface WebhookEvent {
  integrationId: IntegrationId;
  eventType: string;
  payload: Record<string, unknown>;
  sourceId?: string;
}
