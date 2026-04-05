import type { ProviderMessage, ProviderOptions, StreamChunk } from '@/types/provider';

export interface CodingAgentProvider {
  readonly id: string;
  readonly name: string;
  stream(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): AsyncGenerator<StreamChunk>;
  validateCli(): Promise<boolean>;
}
