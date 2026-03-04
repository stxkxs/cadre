import { BaseProvider, type ProviderMessage, type ProviderOptions, type ProviderResponse, type StreamChunk } from './base';
import type { ModelProvider } from '@/lib/engine/types';
import { spawn, type ChildProcessWithoutNullStreams } from 'child_process';

export class ClaudeCodeProvider extends BaseProvider {
  readonly id: ModelProvider = 'claude-code';
  readonly name = 'Claude Code';

  async chat(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): Promise<ProviderResponse> {
    let content = '';
    let tokens = { input: 0, output: 0 };
    for await (const chunk of this.stream(messages, options, _apiKey)) {
      if (chunk.type === 'text') content += chunk.content;
      if (chunk.type === 'done' && chunk.tokens) tokens = chunk.tokens;
    }
    return { content, tokens, model: 'claude-code', finishReason: 'stop' };
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async *stream(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): AsyncGenerator<StreamChunk> {
    const prompt = this.buildPrompt(messages);
    const workspaceEnabled = options.workspace && options.workspace !== 'off' && options.workspacePath;

    const args = ['-p', '--output-format', 'json'];

    const maxTurns = options.maxTurns || 10;
    args.push('--max-turns', String(maxTurns));

    if (workspaceEnabled && options.workspace === 'full') {
      args.push('--dangerously-skip-permissions');
    }

    const spawnOptions: { env: NodeJS.ProcessEnv; cwd?: string } = {
      env: process.env,
    };

    if (workspaceEnabled) {
      spawnOptions.cwd = options.workspacePath;
    }

    const proc = spawn('claude', args, spawnOptions);

    // Pipe prompt via stdin to avoid ARG_MAX limits
    proc.stdin.write(prompt);
    proc.stdin.end();

    const cleanup = this.attachAbortHandler(proc, options.signal);

    let stdout = '';
    let stderr = '';
    proc.stdout.on('data', (data: Buffer) => { stdout += data.toString(); });
    proc.stderr.on('data', (data: Buffer) => { stderr += data.toString(); });

    const closePromise = new Promise<{ code: number | null; signal: NodeJS.Signals | null }>((resolve) => {
      proc.on('close', (code, signal) => resolve({ code, signal }));
    });

    proc.on('error', (err) => {
      cleanup();
      if ((err as NodeJS.ErrnoException).code === 'ENOENT') {
        throw new Error('Claude Code CLI not found. Install it with: npm install -g @anthropic-ai/claude-code');
      }
      throw err;
    });

    const result = await closePromise;
    cleanup();

    if (result.signal === 'SIGTERM' || options.signal?.aborted) {
      // Yield any partial output captured before cancellation
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error('Claude Code was cancelled');
    }

    if (result.code !== 0) {
      // Yield any partial output even on error
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error(stderr || `Claude Code exited with code ${result.code}`);
    }

    const parsed = this.parseOutput(stdout);
    yield { type: 'text', content: parsed };
    yield { type: 'done', content: '', tokens: { input: 0, output: 0 } };
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async validateKey(_apiKey: string): Promise<boolean> {
    return new Promise((resolve) => {
      const proc = spawn('claude', ['--version']);
      proc.on('close', (code) => resolve(code === 0));
      proc.on('error', () => resolve(false));
    });
  }

  private buildPrompt(messages: ProviderMessage[]): string {
    const system = messages.find(m => m.role === 'system')?.content;
    const user = messages
      .filter(m => m.role === 'user')
      .map(m => m.content)
      .join('\n');
    return system ? `${system}\n\n${user}` : user;
  }

  /** Parse Claude Code JSON output, falling back to raw text. */
  private parseOutput(raw: string): string {
    const trimmed = raw.trim();
    if (!trimmed) return '';

    try {
      const parsed = JSON.parse(trimmed);
      if (typeof parsed === 'object' && parsed !== null) {
        // Claude Code JSON: { result: "...", cost_usd: ..., duration_ms: ..., num_turns: ... }
        if (typeof parsed.result === 'string') return parsed.result;
        if (typeof parsed.text === 'string') return parsed.text;
        if (typeof parsed.content === 'string') return parsed.content;
        // Array of content blocks
        if (Array.isArray(parsed.content)) {
          return parsed.content
            .filter((c: { type?: string; text?: string }) => c.type === 'text' && c.text)
            .map((c: { text: string }) => c.text)
            .join('');
        }
      }
      // Parsed but no recognized field — return raw
      return trimmed;
    } catch {
      // Not JSON — return as plain text
      return trimmed;
    }
  }

  private attachAbortHandler(proc: ChildProcessWithoutNullStreams, signal?: AbortSignal): () => void {
    if (!signal) return () => {};

    if (signal.aborted) {
      proc.kill('SIGTERM');
      return () => {};
    }

    const onAbort = () => { proc.kill('SIGTERM'); };
    signal.addEventListener('abort', onAbort, { once: true });

    return () => {
      signal.removeEventListener('abort', onAbort);
    };
  }
}
