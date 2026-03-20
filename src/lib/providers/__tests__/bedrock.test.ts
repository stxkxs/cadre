import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BedrockProvider } from '../bedrock';
import type { ProviderMessage } from '../base';

// Mock AWS SDK clients
const mockSend = vi.fn();

vi.mock('@aws-sdk/client-bedrock-runtime', () => {
  return {
    BedrockRuntimeClient: class {
      send = mockSend;
    },
    ConverseCommand: class {
      constructor(public input: Record<string, unknown>) {}
    },
    ConverseStreamCommand: class {
      constructor(public input: Record<string, unknown>) {}
    },
  };
});

const mockBedrockSend = vi.fn();

vi.mock('@aws-sdk/client-bedrock', () => {
  return {
    BedrockClient: class {
      send = mockBedrockSend;
    },
    ListFoundationModelsCommand: class {
      constructor(public input: Record<string, unknown>) {}
    },
  };
});

describe('BedrockProvider', () => {
  let provider: BedrockProvider;

  beforeEach(() => {
    vi.clearAllMocks();
    provider = new BedrockProvider();
  });

  it('has correct id and name', () => {
    expect(provider.id).toBe('bedrock');
    expect(provider.name).toBe('AWS Bedrock');
  });

  it('chat returns content and tokens', async () => {
    mockSend.mockResolvedValue({
      output: {
        message: {
          content: [{ text: 'Hello from Bedrock' }],
        },
      },
      usage: { inputTokens: 10, outputTokens: 20 },
      stopReason: 'end_turn',
    });

    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const result = await provider.chat(messages, { model: 'anthropic.claude-3-sonnet' }, '');

    expect(result.content).toBe('Hello from Bedrock');
    expect(result.tokens.input).toBe(10);
    expect(result.tokens.output).toBe(20);
    expect(result.model).toBe('anthropic.claude-3-sonnet');
    expect(result.finishReason).toBe('end_turn');
  });

  it('extracts system messages for Bedrock API format', async () => {
    mockSend.mockResolvedValue({
      output: { message: { content: [{ text: 'ok' }] } },
      usage: { inputTokens: 5, outputTokens: 5 },
      stopReason: 'end_turn',
    });

    const messages: ProviderMessage[] = [
      { role: 'system', content: 'You are helpful' },
      { role: 'user', content: 'Hi' },
    ];
    await provider.chat(messages, { model: 'anthropic.claude-3-sonnet' }, '');

    const command = mockSend.mock.calls[0][0];
    expect(command.input.system).toEqual([{ text: 'You are helpful' }]);
    expect(command.input.messages).toHaveLength(1);
    expect(command.input.messages[0].role).toBe('user');
  });

  it('handles empty response content gracefully', async () => {
    mockSend.mockResolvedValue({
      output: { message: { content: [] } },
      usage: { inputTokens: 0, outputTokens: 0 },
      stopReason: 'end_turn',
    });

    const result = await provider.chat([{ role: 'user', content: 'Hi' }], { model: 'test' }, '');
    expect(result.content).toBe('');
  });

  it('stream yields text chunks then done with tokens', async () => {
    const streamEvents = [
      { contentBlockDelta: { delta: { text: 'Hello' } } },
      { contentBlockDelta: { delta: { text: ' world' } } },
      { metadata: { usage: { inputTokens: 15, outputTokens: 30 } } },
    ];

    mockSend.mockResolvedValue({
      stream: (async function* () {
        for (const event of streamEvents) yield event;
      })(),
    });

    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: 'test' }, '')) {
      chunks.push(chunk);
    }

    expect(chunks).toHaveLength(3);
    expect(chunks[0]).toEqual({ type: 'text', content: 'Hello' });
    expect(chunks[1]).toEqual({ type: 'text', content: ' world' });
    expect(chunks[2].type).toBe('done');
    expect(chunks[2].tokens).toEqual({ input: 15, output: 30 });
  });

  it('stream handles no stream in response', async () => {
    mockSend.mockResolvedValue({});

    const chunks = [];
    for await (const chunk of provider.stream([{ role: 'user', content: 'Hi' }], { model: 'test' }, '')) {
      chunks.push(chunk);
    }

    expect(chunks).toHaveLength(1);
    expect(chunks[0]).toEqual({ type: 'done', content: '', tokens: { input: 0, output: 0 } });
  });

  it('validateKey returns true on successful list call', async () => {
    mockBedrockSend.mockResolvedValue({ modelSummaries: [] });
    const result = await provider.validateKey('');
    expect(result).toBe(true);
  });

  it('validateKey returns false on error', async () => {
    mockBedrockSend.mockRejectedValue(new Error('AccessDenied'));
    const result = await provider.validateKey('');
    expect(result).toBe(false);
  });

  it('listModels returns formatted model list', async () => {
    mockBedrockSend.mockResolvedValue({
      modelSummaries: [
        { modelId: 'anthropic.claude-3-sonnet', modelName: 'Claude 3 Sonnet', providerName: 'Anthropic' },
        { modelId: 'meta.llama3-70b', modelName: 'Llama 3 70B', providerName: 'Meta' },
        { modelId: null, modelName: null }, // should be filtered out
      ],
    });

    const models = await provider.listModels();
    expect(models).toHaveLength(2);
    expect(models[0]).toEqual({ id: 'anthropic.claude-3-sonnet', name: 'Claude 3 Sonnet', provider: 'Anthropic' });
    expect(models[1]).toEqual({ id: 'meta.llama3-70b', name: 'Llama 3 70B', provider: 'Meta' });
  });

  it('listModels handles empty response', async () => {
    mockBedrockSend.mockResolvedValue({});
    const models = await provider.listModels();
    expect(models).toEqual([]);
  });
});
