import OpenAI from 'openai';
import { BaseProvider, type ProviderMessage, type ProviderOptions, type ProviderResponse, type StreamChunk } from './base';
import type { ModelProvider } from '@/lib/engine/types';

export class OpenAIProvider extends BaseProvider {
  readonly id: ModelProvider = 'openai';
  readonly name = 'OpenAI';

  async chat(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): Promise<ProviderResponse> {
    const client = new OpenAI({ apiKey });
    const response = await client.chat.completions.create({
      model: options.model || 'gpt-4o',
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
      model: response.model,
      finishReason: choice?.finish_reason || 'stop',
    };
  }

  async *stream(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): AsyncGenerator<StreamChunk> {
    const client = new OpenAI({ apiKey });
    const stream = await client.chat.completions.create({
      model: options.model || 'gpt-4o',
      temperature: options.temperature,
      max_tokens: options.maxTokens || 4096,
      messages: messages.map(m => ({ role: m.role, content: m.content })),
      stream: true,
      stream_options: { include_usage: true },
    });

    let inputTokens = 0;
    let outputTokens = 0;

    for await (const chunk of stream) {
      const delta = chunk.choices[0]?.delta;
      if (delta?.content) {
        yield { type: 'text', content: delta.content };
      }
      if (chunk.usage) {
        inputTokens = chunk.usage.prompt_tokens || 0;
        outputTokens = chunk.usage.completion_tokens || 0;
      }
    }

    yield { type: 'done', content: '', tokens: { input: inputTokens, output: outputTokens } };
  }

  async validateKey(apiKey: string): Promise<boolean> {
    try {
      const client = new OpenAI({ apiKey });
      await client.models.list();
      return true;
    } catch {
      return false;
    }
  }
}
