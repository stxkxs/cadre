import Anthropic from '@anthropic-ai/sdk';
import { BaseProvider, type ProviderMessage, type ProviderOptions, type ProviderResponse, type StreamChunk } from './base';
import type { ModelProvider } from '@/lib/engine/types';

export class AnthropicProvider extends BaseProvider {
  readonly id: ModelProvider = 'anthropic';
  readonly name = 'Anthropic';

  async chat(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): Promise<ProviderResponse> {
    const client = new Anthropic({ apiKey });
    const systemPrompt = options.systemPrompt || messages.find(m => m.role === 'system')?.content;
    const filteredMessages = messages.filter(m => m.role !== 'system');

    const response = await client.messages.create({
      model: options.model || 'claude-sonnet-4-6',
      max_tokens: options.maxTokens || 4096,
      temperature: options.temperature,
      system: systemPrompt,
      messages: filteredMessages.map(m => ({
        role: m.role as 'user' | 'assistant',
        content: m.content,
      })),
    });

    const textContent = response.content.find(c => c.type === 'text');
    return {
      content: textContent?.text || '',
      tokens: { input: response.usage.input_tokens, output: response.usage.output_tokens },
      model: response.model,
      finishReason: response.stop_reason || 'end_turn',
    };
  }

  async *stream(messages: ProviderMessage[], options: ProviderOptions, apiKey: string): AsyncGenerator<StreamChunk> {
    const client = new Anthropic({ apiKey });
    const systemPrompt = options.systemPrompt || messages.find(m => m.role === 'system')?.content;
    const filteredMessages = messages.filter(m => m.role !== 'system');

    const stream = client.messages.stream({
      model: options.model || 'claude-sonnet-4-6',
      max_tokens: options.maxTokens || 4096,
      temperature: options.temperature,
      system: systemPrompt,
      messages: filteredMessages.map(m => ({
        role: m.role as 'user' | 'assistant',
        content: m.content,
      })),
    });

    for await (const event of stream) {
      if (event.type === 'content_block_delta' && event.delta.type === 'text_delta') {
        yield { type: 'text', content: event.delta.text };
      }
    }

    const finalMessage = await stream.finalMessage();
    yield {
      type: 'done',
      content: '',
      tokens: { input: finalMessage.usage.input_tokens, output: finalMessage.usage.output_tokens },
    };
  }

  async validateKey(apiKey: string): Promise<boolean> {
    try {
      const client = new Anthropic({ apiKey });
      await client.messages.create({
        model: 'claude-haiku-4-5',
        max_tokens: 10,
        messages: [{ role: 'user', content: 'Hi' }],
      });
      return true;
    } catch {
      return false;
    }
  }
}
