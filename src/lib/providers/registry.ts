import type { CodingAgentProvider } from './base';
import { ClaudeCodeProvider } from './claude-code';
import { CodexProvider } from './codex';
import { GeminiProvider } from './gemini';

const providers = new Map<string, CodingAgentProvider>();

function register(provider: CodingAgentProvider): void {
  providers.set(provider.id, provider);
}

register(new ClaudeCodeProvider());
register(new CodexProvider());
register(new GeminiProvider());

export function getProvider(id: string): CodingAgentProvider {
  const provider = providers.get(id);
  if (!provider) {
    // Fall back to claude-code for unknown/missing provider IDs
    return providers.get('claude-code')!;
  }
  return provider;
}

export function listProviders(): CodingAgentProvider[] {
  return [...providers.values()];
}
