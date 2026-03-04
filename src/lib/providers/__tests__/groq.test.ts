import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GroqProvider } from '../groq';
import type { ProviderMessage } from '../base';

const mockCreate = vi.fn();
const mockModelsList = vi.fn().mockResolvedValue({ data: [] });

vi.mock('groq-sdk', () => {
  return {
    default: class MockGroq {
      chat = { completions: { create: mockCreate } };
      models = { list: mockModelsList };
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      constructor(_opts: Record<string, unknown>) {}
    },
  };
});

describe('GroqProvider', () => {
  let provider: GroqProvider;

  beforeEach(() => {
    vi.clearAllMocks();
    mockCreate.mockImplementation((opts: { stream?: boolean }) => {
      if (opts.stream) {
        return {
          [Symbol.asyncIterator]: async function* () {
            yield { choices: [{ delta: { content: 'Fast' } }] };
            yield { choices: [{ delta: { content: ' reply' } }] };
          },
        };
      }
      return Promise.resolve({
        choices: [{ message: { content: 'Hello from Groq' }, finish_reason: 'stop' }],
        usage: { prompt_tokens: 5, completion_tokens: 10 },
        model: 'llama-3.3-70b-versatile',
      });
    });
    provider = new GroqProvider();
  });

  it('has correct id and name', () => {
    expect(provider.id).toBe('groq');
    expect(provider.name).toBe('Groq');
  });

  it('chat returns content and tokens', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const result = await provider.chat(messages, { model: 'llama-3.3-70b-versatile' }, 'test-key');
    expect(result.content).toBe('Hello from Groq');
    expect(result.tokens.input).toBe(5);
    expect(result.tokens.output).toBe(10);
  });

  it('stream yields text chunks then done', async () => {
    const messages: ProviderMessage[] = [{ role: 'user', content: 'Hello' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: 'llama-3.3-70b-versatile' }, 'test-key')) {
      chunks.push(chunk);
    }
    expect(chunks).toHaveLength(3);
    expect(chunks[0]).toEqual({ type: 'text', content: 'Fast' });
    expect(chunks[1]).toEqual({ type: 'text', content: ' reply' });
    expect(chunks[2].type).toBe('done');
  });

  it('uses default model when empty string provided', async () => {
    await provider.chat([{ role: 'user', content: 'Hi' }], { model: '' }, 'test-key');
    const call = mockCreate.mock.calls[0][0];
    expect(call.model).toBe('llama-3.3-70b-versatile');
  });
});
