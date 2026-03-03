import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AnthropicProvider } from '../anthropic';
import type { ProviderMessage } from '../base';

// Mock the Anthropic SDK
const mockCreate = vi.fn().mockResolvedValue({
  content: [{ type: 'text', text: 'Hello from Claude' }],
  usage: { input_tokens: 10, output_tokens: 25 },
  model: 'claude-sonnet-4-6',
  stop_reason: 'end_turn',
});

const mockStreamObj = {
  [Symbol.asyncIterator]: async function* () {
    yield { type: 'content_block_delta', delta: { type: 'text_delta', text: 'Hello' } };
    yield { type: 'content_block_delta', delta: { type: 'text_delta', text: ' world' } };
  },
  finalMessage: vi.fn().mockResolvedValue({
    usage: { input_tokens: 15, output_tokens: 30 },
  }),
};

const mockStream = vi.fn().mockReturnValue(mockStreamObj);

vi.mock('@anthropic-ai/sdk', () => {
  return {
    default: class MockAnthropic {
      messages = { create: mockCreate, stream: mockStream };
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      constructor(_opts: Record<string, unknown>) {}
    },
  };
});

describe('AnthropicProvider', () => {
  let provider: AnthropicProvider;

  beforeEach(() => {
    vi.clearAllMocks();
    mockCreate.mockResolvedValue({
      content: [{ type: 'text', text: 'Hello from Claude' }],
      usage: { input_tokens: 10, output_tokens: 25 },
      model: 'claude-sonnet-4-6',
      stop_reason: 'end_turn',
    });
    mockStreamObj.finalMessage.mockResolvedValue({
      usage: { input_tokens: 15, output_tokens: 30 },
    });
    provider = new AnthropicProvider();
  });

  it('has correct id and name', () => {
    expect(provider.id).toBe('anthropic');
    expect(provider.name).toBe('Anthropic');
  });

  it('chat returns content and tokens', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const result = await provider.chat(messages, { model: 'claude-sonnet-4-6' }, 'test-key');
    expect(result.content).toBe('Hello from Claude');
    expect(result.tokens.input).toBe(10);
    expect(result.tokens.output).toBe(25);
    expect(result.model).toBe('claude-sonnet-4-6');
  });

  it('extracts system messages for Anthropic API format', async () => {
    const messages: ProviderMessage[] = [
      { role: 'system', content: 'You are helpful' },
      { role: 'user', content: 'Hello' },
    ];
    await provider.chat(messages, { model: 'claude-sonnet-4-6' }, 'test-key');

    const createCall = mockCreate.mock.calls[0][0];
    expect(createCall.system).toBe('You are helpful');
    expect(createCall.messages).toEqual([{ role: 'user', content: 'Hello' }]);
  });

  it('stream yields text chunks then done with tokens', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: 'claude-sonnet-4-6' }, 'test-key')) {
      chunks.push(chunk);
    }
    expect(chunks).toHaveLength(3);
    expect(chunks[0]).toEqual({ type: 'text', content: 'Hello' });
    expect(chunks[1]).toEqual({ type: 'text', content: ' world' });
    expect(chunks[2].type).toBe('done');
    expect(chunks[2].tokens).toEqual({ input: 15, output: 30 });
  });

  it('uses default model when none specified', async () => {
    await provider.chat([{ role: 'user', content: 'Hi' }], { model: '' }, 'test-key');
    const createCall = mockCreate.mock.calls[0][0];
    expect(createCall.model).toBe('claude-sonnet-4-6');
  });
});
