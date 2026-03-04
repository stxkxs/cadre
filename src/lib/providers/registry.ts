import type { ModelProvider } from '@/lib/engine/types';
import type { BaseProvider } from './base';
import { AnthropicProvider } from './anthropic';
import { OpenAIProvider } from './openai';
import { GroqProvider } from './groq';
import { ClaudeCodeProvider } from './claude-code';

class ProviderRegistry {
  private providers = new Map<ModelProvider, BaseProvider>();

  constructor() {
    this.register(new AnthropicProvider());
    this.register(new OpenAIProvider());
    this.register(new GroqProvider());
    this.register(new ClaudeCodeProvider());
  }

  private register(provider: BaseProvider) {
    this.providers.set(provider.id, provider);
  }

  get(id: ModelProvider): BaseProvider {
    const provider = this.providers.get(id);
    if (!provider) throw new Error(`Unknown provider: ${id}`);
    return provider;
  }

  has(id: string): boolean {
    return this.providers.has(id as ModelProvider);
  }

  getAll(): BaseProvider[] {
    return Array.from(this.providers.values());
  }

  getIds(): ModelProvider[] {
    return Array.from(this.providers.keys());
  }
}

export const providerRegistry = new ProviderRegistry();
