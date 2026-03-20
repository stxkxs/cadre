import type { IntegrationId } from '@/types/integration';
import type { BaseIntegration } from './base';
import { GitHubIntegration } from './providers/github';
import { LinearIntegration } from './providers/linear';
import { NotionIntegration } from './providers/notion';
import { SlackIntegration } from './providers/slack';
import { FigmaIntegration } from './providers/figma';
import { JiraIntegration } from './providers/jira';
import { ConfluenceIntegration } from './providers/confluence';
import { GoogleDocsIntegration } from './providers/google-docs';
import { LoomIntegration } from './providers/loom';
import { CodaIntegration } from './providers/coda';

class IntegrationRegistry {
  private integrations = new Map<IntegrationId, BaseIntegration>();

  constructor() {
    this.register(new GitHubIntegration());
    this.register(new LinearIntegration());
    this.register(new NotionIntegration());
    this.register(new SlackIntegration());
    this.register(new FigmaIntegration());
    this.register(new JiraIntegration());
    this.register(new ConfluenceIntegration());
    this.register(new GoogleDocsIntegration());
    this.register(new LoomIntegration());
    this.register(new CodaIntegration());
  }

  private register(integration: BaseIntegration) {
    this.integrations.set(integration.id, integration);
  }

  get(id: IntegrationId): BaseIntegration {
    const integration = this.integrations.get(id);
    if (!integration) throw new Error(`Unknown integration: ${id}`);
    return integration;
  }

  has(id: string): boolean {
    return this.integrations.has(id as IntegrationId);
  }

  getAll(): BaseIntegration[] {
    return Array.from(this.integrations.values());
  }

  getIds(): IntegrationId[] {
    return Array.from(this.integrations.keys());
  }
}

export const integrationRegistry = new IntegrationRegistry();
