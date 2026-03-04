import { describe, it, expect, vi, beforeEach } from 'vitest';
import { OpenAIProvider } from '../openai';
import type { ProviderMessage } from '../base';

const mockCreate = vi.fn();
const mockModelsList = vi.fn().mockResolvedValue({ data: [] });

vi.mock('openai', () => {
  return {
    default: class MockOpenAI {
      chat = { completions: { create: mockCreate } };
      models = { list: mockModelsList };
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      constructor(_opts: Record<string, unknown>) {}
    },
  };
});

describe('OpenAIProvider', () => {
  let provider: OpenAIProvider;

  beforeEach(() => {
    vi.clearAllMocks();
    mockCreate.mockImplementation((opts: { stream?: boolean }) => {
      if (opts.stream) {
        return {
          [Symbol.asyncIterator]: async function* () {
            yield { choices: [{ delta: { content: 'Hello' } }], usage: null };
            yield { choices: [{ delta: { content: ' there' } }], usage: null };
            yield { choices: [{ delta: {} }], usage: { prompt_tokens: 12, completion_tokens: 20 } };
          },
        };
      }
      return Promise.resolve({
        choices: [{ message: { content: 'Hello from GPT' }, finish_reason: 'stop' }],
        usage: { prompt_tokens: 8, completion_tokens: 15 },
        model: 'gpt-4o',
      });
    });
    provider = new OpenAIProvider();
  });

  it('has correct id and name', () => {
    expect(provider.id).toBe('openai');
    expect(provider.name).toBe('OpenAI');
  });

  it('chat returns content and tokens', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const result = await provider.chat(messages, { model: 'gpt-4o' }, 'test-key');
    expect(result.content).toBe('Hello from GPT');
    expect(result.tokens.input).toBe(8);
    expect(result.tokens.output).toBe(15);
    expect(result.model).toBe('gpt-4o');
  });

  it('stream yields text chunks with usage tracking', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: 'gpt-4o' }, 'test-key')) {
      chunks.push(chunk);
    }
    expect(chunks).toHaveLength(3);
    expect(chunks[0]).toEqual({ type: 'text', content: 'Hello' });
    expect(chunks[1]).toEqual({ type: 'text', content: ' there' });
    expect(chunks[2].type).toBe('done');
    expect(chunks[2].tokens).toEqual({ input: 12, output: 20 });
  });

  it('uses default model when empty string provided', async () => {
    await provider.chat([{ role: 'user', content: 'Hi' }], { model: '' }, 'test-key');
    const call = mockCreate.mock.calls[0][0];
    expect(call.model).toBe('gpt-4o');
  });
});
