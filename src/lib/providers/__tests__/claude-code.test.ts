import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ClaudeCodeProvider } from '../claude-code';
import type { ProviderMessage } from '../base';
import { EventEmitter } from 'events';

// Mock child_process.spawn
vi.mock('child_process', () => ({
  spawn: vi.fn(),
}));

function createMockProc(stdout = '', exitCode: number | null = 0, signal: string | null = null) {
  const proc = new EventEmitter();
  const stdoutEmitter = new EventEmitter();
  const stderrEmitter = new EventEmitter();
  const stdinData: string[] = [];

  Object.assign(proc, {
    stdout: stdoutEmitter,
    stderr: stderrEmitter,
    stdin: {
      write: (data: string) => stdinData.push(data),
      end: () => {},
    },
    kill: vi.fn(),
    pid: 1234,
    __stdinData: stdinData,
  });

  // Schedule stdout data + close after tick
  setTimeout(() => {
    if (stdout) stdoutEmitter.emit('data', Buffer.from(stdout));
    proc.emit('close', exitCode, signal);
  }, 10);

  return proc;
}

describe('ClaudeCodeProvider', () => {
  let provider: ClaudeCodeProvider;
  let spawnMock: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    vi.clearAllMocks();
    provider = new ClaudeCodeProvider();
    const cp = await import('child_process');
    spawnMock = cp.spawn as unknown as ReturnType<typeof vi.fn>;
  });

  it('has correct id and name', () => {
    expect(provider.id).toBe('claude-code');
    expect(provider.name).toBe('Claude Code');
  });

  it('builds correct CLI args', async () => {
    const jsonOutput = JSON.stringify({ result: 'done' });
    spawnMock.mockReturnValue(createMockProc(jsonOutput));

    const messages: ProviderMessage[] = [{ role: 'user', content: 'test' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: '', maxTurns: 5 }, '')) {
      chunks.push(chunk);
    }

    expect(spawnMock).toHaveBeenCalledWith(
      'claude',
      ['-p', '--output-format', 'json', '--max-turns', '5'],
      expect.objectContaining({ env: process.env }),
    );
  });

  it('sets workspace cwd and permissions flag', async () => {
    const jsonOutput = JSON.stringify({ result: 'done' });
    spawnMock.mockReturnValue(createMockProc(jsonOutput));

    const messages: ProviderMessage[] = [{ role: 'user', content: 'test' }];
    const chunks = [];
    for await (const chunk of provider.stream(messages, {
      model: '',
      workspace: 'full',
      workspacePath: '/tmp/workspace',
    }, '')) {
      chunks.push(chunk);
    }

    const args = spawnMock.mock.calls[0][1];
    expect(args).toContain('--dangerously-skip-permissions');
    expect(spawnMock.mock.calls[0][2].cwd).toBe('/tmp/workspace');
  });

  it('pipes prompt via stdin', async () => {
    const proc = createMockProc(JSON.stringify({ result: 'ok' }));
    spawnMock.mockReturnValue(proc);

    const messages: ProviderMessage[] = [
      { role: 'system', content: 'Be helpful' },
      { role: 'user', content: 'Hello' },
    ];

    const chunks = [];
    for await (const chunk of provider.stream(messages, { model: '' }, '')) {
      chunks.push(chunk);
    }

    expect((proc as unknown as { __stdinData: string[] }).__stdinData.join('')).toBe('Be helpful\n\nHello');
  });

  describe('parseOutput', () => {
    it('parses JSON with result field', async () => {
      spawnMock.mockReturnValue(createMockProc(JSON.stringify({ result: 'Hello world' })));
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
      expect(chunks[0].content).toBe('Hello world');
    });

    it('parses JSON with text field', async () => {
      spawnMock.mockReturnValue(createMockProc(JSON.stringify({ text: 'From text' })));
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
      expect(chunks[0].content).toBe('From text');
    });

    it('parses JSON with content string field', async () => {
      spawnMock.mockReturnValue(createMockProc(JSON.stringify({ content: 'From content' })));
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
      expect(chunks[0].content).toBe('From content');
    });

    it('parses array content blocks', async () => {
      const output = JSON.stringify({
        content: [
          { type: 'text', text: 'Part 1' },
          { type: 'text', text: 'Part 2' },
        ],
      });
      spawnMock.mockReturnValue(createMockProc(output));
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
      expect(chunks[0].content).toBe('Part 1Part 2');
    });

    it('falls back to plain text for non-JSON', async () => {
      spawnMock.mockReturnValue(createMockProc('Just plain text'));
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
      expect(chunks[0].content).toBe('Just plain text');
    });
  });

  it('sends SIGTERM on abort signal', async () => {
    const proc = new EventEmitter();
    const stdoutEmitter = new EventEmitter();
    const stderrEmitter = new EventEmitter();
    const killFn = vi.fn(() => {
      // Simulate process being killed
      setTimeout(() => {
        proc.emit('close', null, 'SIGTERM');
      }, 5);
    });

    Object.assign(proc, {
      stdout: stdoutEmitter,
      stderr: stderrEmitter,
      stdin: { write: vi.fn(), end: vi.fn() },
      kill: killFn,
      pid: 1234,
    });

    spawnMock.mockReturnValue(proc);

    const controller = new AbortController();
    const messages: ProviderMessage[] = [{ role: 'user', content: 'test' }];

    const promise = (async () => {
      const chunks = [];
      for await (const chunk of provider.stream(messages, { model: '', signal: controller.signal }, '')) {
        chunks.push(chunk);
      }
      return chunks;
    })();

    // Abort after a tick
    setTimeout(() => controller.abort(), 15);

    await expect(promise).rejects.toThrow('cancelled');
    expect(killFn).toHaveBeenCalledWith('SIGTERM');
  });

  it('throws on non-zero exit code', async () => {
    const proc = createMockProc('', 1);
    // Emit stderr before close
    setTimeout(() => {
      (proc as unknown as { stderr: EventEmitter }).stderr.emit('data', Buffer.from('Something failed'));
    }, 5);
    spawnMock.mockReturnValue(proc);

    const promise = (async () => {
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
    })();

    await expect(promise).rejects.toThrow();
  });

  it('throws with exit code message when no stderr', async () => {
    spawnMock.mockReturnValue(createMockProc('', 1));

    const promise = (async () => {
      const chunks = [];
      for await (const chunk of provider.stream([{ role: 'user', content: 'test' }], { model: '' }, '')) {
        chunks.push(chunk);
      }
    })();

    await expect(promise).rejects.toThrow('exited with code 1');
  });

  it('chat delegates to stream', async () => {
    spawnMock.mockReturnValue(createMockProc(JSON.stringify({ result: 'chat result' })));
    const result = await provider.chat([{ role: 'user', content: 'test' }], { model: '' }, '');
    expect(result.content).toBe('chat result');
    expect(result.model).toBe('claude-code');
  });
});
