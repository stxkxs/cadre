import Groq from 'groq-sdk';
import { BaseProvider, type ProviderMessage, type ProviderOptions, type ProviderResponse, type StreamChunk } from './base';
import type { ModelProvider } from '@/lib/engine/types';

export class GroqProvider extends BaseProvider {
  readonly id: ModelProvider = 'groq';
  readonly name = 'Groq';

  async chat(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): Promise<ProviderResponse> {
    const client = new Groq({ apiKey });
    const response = await client.chat.completions.create({
      model: options.model || 'llama-3.3-70b-versatile',
      temperature: options.temperature,
      max_tokens: options.maxTokens || 4096,
      messages: messages.map(m => ({ role: m.role, content: m.content })),
    });

    const choice = response.choices[0];
    return {
      content: choice?.message?.content || '',
      tokens: {
        input: response.usage?.prompt_tokens || 0,
        output: response.usage?.completion_tokens || 0,
      },
      model: response.model || options.model || 'llama-3.3-70b-versatile',
      finishReason: choice?.finish_reason || 'stop',
    };
  }

  async *stream(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): AsyncGenerator<StreamChunk> {
    const client = new Groq({ apiKey });
    const stream = await client.chat.completions.create({
      model: options.model || 'llama-3.3-70b-versatile',
      temperature: options.temperature,
      max_tokens: options.maxTokens || 4096,
      messages: messages.map(m => ({ role: m.role, content: m.content })),
      stream: true,
    });

    for await (const chunk of stream) {
      const delta = chunk.choices[0]?.delta;
      if (delta?.content) {
        yield { type: 'text', content: delta.content };
      }
    }

    yield { type: 'done', content: '', tokens: { input: 0, output: 0 } };
  }

  async validateKey(apiKey: string): Promise<boolean> {
    try {
      const client = new Groq({ apiKey });
      await client.models.list();
      return true;
    } catch {
      return false;
    }
  }
}
