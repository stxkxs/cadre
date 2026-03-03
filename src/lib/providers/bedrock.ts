import {
  BedrockRuntimeClient,
  ConverseCommand,
  ConverseStreamCommand,
  type Message,
  type ContentBlock,
} from '@aws-sdk/client-bedrock-runtime';
import { BedrockClient, ListFoundationModelsCommand } from '@aws-sdk/client-bedrock';
import { BaseProvider, type ProviderMessage, type ProviderOptions, type ProviderResponse, type StreamChunk } from './base';
import type { ModelProvider } from '@/lib/engine/types';

export class BedrockProvider extends BaseProvider {
  readonly id: ModelProvider = 'bedrock';
  readonly name = 'AWS Bedrock';

  private getClient(): BedrockRuntimeClient {
    return new BedrockRuntimeClient({
      region: process.env.AWS_REGION || 'us-east-1',
    });
  }

  private getBedrockClient(): BedrockClient {
    return new BedrockClient({
      region: process.env.AWS_REGION || 'us-east-1',
    });
  }

  private toBedrockMessages(messages: ProviderMessage[]): { system?: string; bedrockMessages: Message[] } {
    const systemMsg = messages.find(m => m.role === 'system');
    const bedrockMessages: Message[] = messages
      .filter(m => m.role !== 'system')
      .map(m => ({
        role: m.role as 'user' | 'assistant',
        content: [{ text: m.content } as ContentBlock],
      }));

    return {
      system: systemMsg?.content,
      bedrockMessages,
    };
  }

  async chat(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): Promise<ProviderResponse> {
    const client = this.getClient();
    const { system, bedrockMessages } = this.toBedrockMessages(messages);

    const command = new ConverseCommand({
      modelId: options.model,
      messages: bedrockMessages,
      system: system ? [{ text: system }] : undefined,
      inferenceConfig: {
        maxTokens: options.maxTokens || 4096,
        temperature: options.temperature,
      },
    });

    const response = await client.send(command);
    const outputContent = response.output?.message?.content;
    const text = outputContent?.map(c => c.text || '').join('') || '';

    return {
      content: text,
      tokens: {
        input: response.usage?.inputTokens || 0,
        output: response.usage?.outputTokens || 0,
      },
      model: options.model,
      finishReason: response.stopReason || 'end_turn',
    };
  }

  async *stream(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): AsyncGenerator<StreamChunk> {
    const client = this.getClient();
    const { system, bedrockMessages } = this.toBedrockMessages(messages);

    const command = new ConverseStreamCommand({
      modelId: options.model,
      messages: bedrockMessages,
      system: system ? [{ text: system }] : undefined,
      inferenceConfig: {
        maxTokens: options.maxTokens || 4096,
        temperature: options.temperature,
      },
    });

    const response = await client.send(command);

    let inputTokens = 0;
    let outputTokens = 0;

    if (response.stream) {
      for await (const event of response.stream) {
        if (event.contentBlockDelta?.delta?.text) {
          yield { type: 'text', content: event.contentBlockDelta.delta.text };
        }
        if (event.metadata?.usage) {
          inputTokens = event.metadata.usage.inputTokens || 0;
          outputTokens = event.metadata.usage.outputTokens || 0;
        }
      }
    }

    yield {
      type: 'done',
      content: '',
      tokens: { input: inputTokens, output: outputTokens },
    };
  }

  async validateKey(_apiKey: string): Promise<boolean> {
    try {
      const client = this.getBedrockClient();
      await client.send(new ListFoundationModelsCommand({}));
      return true;
    } catch {
      return false;
    }
  }

  async listModels(): Promise<{ id: string; name: string; provider: string }[]> {
    const client = this.getBedrockClient();
    const response = await client.send(new ListFoundationModelsCommand({
      byOutputModality: 'TEXT',
    }));

    return (response.modelSummaries || [])
      .filter(m => m.modelId && m.modelName)
      .map(m => ({
        id: m.modelId!,
        name: m.modelName!,
        provider: m.providerName || 'Unknown',
      }));
  }
}
