import { pgTable, text, timestamp, jsonb, boolean, uuid } from 'drizzle-orm/pg-core';

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
  provider: text('provider').notNull(), // 'anthropic' | 'openai' | 'groq' | 'claude-code'
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

export const agentTemplates = pgTable('agent_templates', {
  id: uuid('id').primaryKey().defaultRandom(),
  name: text('name').notNull(),
  description: text('description').default(''),
  category: text('category').notNull(),
  nodeData: jsonb('node_data').notNull(),
  isBuiltIn: boolean('is_built_in').default(true),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});
