import { pgTable, text, timestamp, jsonb, boolean, uuid, index } from 'drizzle-orm/pg-core';

export const users = pgTable('users', {
  id: uuid('id').primaryKey().defaultRandom(),
  email: text('email').notNull().unique(),
  name: text('name'),
  image: text('image'),
  githubId: text('github_id').unique(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});

export const userApiKeys = pgTable('user_api_keys', {
  id: uuid('id').primaryKey().defaultRandom(),
  userId: uuid('user_id').notNull().references(() => users.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(), // 'anthropic' | 'openai' | 'groq' | 'claude-code' | 'bedrock'
  encryptedKey: text('encrypted_key').notNull(),
  iv: text('iv').notNull(),
  authTag: text('auth_tag').notNull(),
  isValid: boolean('is_valid').default(false),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

export const workflows = pgTable('workflows', {
  id: uuid('id').primaryKey().defaultRandom(),
  userId: uuid('user_id').notNull().references(() => users.id, { onDelete: 'cascade' }),
  name: text('name').notNull(),
  description: text('description').default(''),
  graphData: jsonb('graph_data').notNull().default({ nodes: [], edges: [] }),
  variables: jsonb('variables').default({}),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

export const runs = pgTable('runs', {
  id: uuid('id').primaryKey().defaultRandom(),
  workflowId: uuid('workflow_id').notNull().references(() => workflows.id, { onDelete: 'cascade' }),
  userId: uuid('user_id').notNull().references(() => users.id, { onDelete: 'cascade' }),
  status: text('status').notNull().default('pending'),
  context: jsonb('context').default({}),
  nodeStates: jsonb('node_states').default({}),
  tokenUsage: jsonb('token_usage').default({ input: 0, output: 0, cost: 0 }),
  startedAt: timestamp('started_at').defaultNow().notNull(),
  completedAt: timestamp('completed_at'),
});

export const integrationConnections = pgTable('integration_connections', {
  id: uuid('id').primaryKey().defaultRandom(),
  userId: uuid('user_id').notNull().references(() => users.id, { onDelete: 'cascade' }),
  integrationId: text('integration_id').notNull(),
  encryptedAccessToken: text('encrypted_access_token').notNull(),
  encryptedRefreshToken: text('encrypted_refresh_token'),
  iv: text('iv').notNull(),
  authTag: text('auth_tag').notNull(),
  tokenExpiresAt: timestamp('token_expires_at'),
  metadata: jsonb('metadata').default({}),
  isActive: boolean('is_active').default(true),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
}, (table) => [
  index('idx_integration_connections_user').on(table.userId),
  index('idx_integration_connections_user_integration').on(table.userId, table.integrationId),
]);

export const webhookEvents = pgTable('webhook_events', {
  id: uuid('id').primaryKey().defaultRandom(),
  integrationId: text('integration_id').notNull(),
  eventType: text('event_type').notNull(),
  payload: jsonb('payload').notNull(),
  sourceId: text('source_id'),
  status: text('status').notNull().default('received'),
  processedAt: timestamp('processed_at'),
  error: text('error'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
}, (table) => [
  index('idx_webhook_events_integration').on(table.integrationId),
  index('idx_webhook_events_status').on(table.status),
]);

export const webhookTriggers = pgTable('webhook_triggers', {
  id: uuid('id').primaryKey().defaultRandom(),
  userId: uuid('user_id').notNull().references(() => users.id, { onDelete: 'cascade' }),
  workflowId: uuid('workflow_id').notNull().references(() => workflows.id, { onDelete: 'cascade' }),
  integrationId: text('integration_id').notNull(),
  eventType: text('event_type').notNull(),
  filter: jsonb('filter').default({}),
  isActive: boolean('is_active').default(true),
  createdAt: timestamp('created_at').defaultNow().notNull(),
}, (table) => [
  index('idx_webhook_triggers_user').on(table.userId),
  index('idx_webhook_triggers_workflow').on(table.workflowId),
  index('idx_webhook_triggers_integration_event').on(table.integrationId, table.eventType),
]);

export const agentTemplates = pgTable('agent_templates', {
  id: uuid('id').primaryKey().defaultRandom(),
  name: text('name').notNull(),
  description: text('description').default(''),
  category: text('category').notNull(),
  nodeData: jsonb('node_data').notNull(),
  isBuiltIn: boolean('is_built_in').default(true),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});
