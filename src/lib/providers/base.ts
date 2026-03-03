import type { ModelProvider } from '@/lib/engine/types';

export interface ProviderMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface ProviderOptions {
  model: string;
  temperature?: number;
  maxTokens?: number;
  systemPrompt?: string;
  workspacePath?: string;
  workspace?: 'off' | 'safe' | 'full';
  maxTurns?: number;
  signal?: AbortSignal;
}

export interface ProviderResponse {
  content: string;
  tokens: { input: number; output: number };
  model: string;
  finishReason: string;
}

export interface StreamChunk {
  type: 'text' | 'done' | 'error';
  content: string;
  tokens?: { input: number; output: number };
}

export abstract class BaseProvider {
  abstract readonly id: ModelProvider;
  abstract readonly name: string;

  abstract chat(
    messages: ProviderMessage[],
    options: ProviderOptions,
    apiKey: string
  ): Promise<ProviderResponse>;

  abstract stream(
    messages: ProviderMessage[],
    options: ProviderOptions,
    apiKey: string
  ): AsyncGenerator<StreamChunk>;

  abstract validateKey(apiKey: string): Promise<boolean>;
}
