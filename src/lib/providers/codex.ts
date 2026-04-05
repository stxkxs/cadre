import { spawn, type ChildProcessWithoutNullStreams } from 'child_process';
import type { CodingAgentProvider } from './base';
import type { ProviderMessage, ProviderOptions, StreamChunk } from '@/types/provider';

export class CodexProvider implements CodingAgentProvider {
  readonly id = 'codex';
  readonly name = 'Codex';

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async *stream(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): AsyncGenerator<StreamChunk> {
    const prompt = this.buildPrompt(messages);

    const args = ['exec', '--json'];

    if (options.workspace === 'full') {
      args.push('--full-auto');
    }

    if (options.workspacePath) {
      args.push('--cd', options.workspacePath);
    }

    args.push(prompt);

    const proc = spawn('codex', args, { env: process.env });

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
        throw new Error('Codex CLI not found. Install it with: npm install -g @openai/codex');
      }
      throw err;
    });

    const result = await closePromise;
    cleanup();

    if (result.signal === 'SIGTERM' || options.signal?.aborted) {
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error('Codex was cancelled');
    }

    if (result.code !== 0) {
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error(stderr || `Codex exited with code ${result.code}`);
    }

    const parsed = this.parseOutput(stdout);
    yield { type: 'text', content: parsed };
    yield { type: 'done', content: '', tokens: this.extractTokens(stdout) };
  }

  async validateCli(): Promise<boolean> {
    return new Promise((resolve) => {
      const proc = spawn('codex', ['--version']);
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

  private parseOutput(raw: string): string {
    // Codex --json outputs JSONL events. Extract assistant message content.
    const lines = raw.trim().split('\n');
    const textParts: string[] = [];

    for (const line of lines) {
      try {
        const event = JSON.parse(line);
        // Look for completed turn events with assistant content
        if (event.type === 'turn.completed' && event.turn?.role === 'assistant') {
          for (const item of event.turn.items || []) {
            if (item.type === 'message' && typeof item.text === 'string') {
              textParts.push(item.text);
            }
          }
        }
        // Also capture direct message items
        if (event.type === 'item.created' && event.item?.role === 'assistant') {
          if (typeof event.item.text === 'string') {
            textParts.push(event.item.text);
          }
        }
      } catch {
        // Not JSON — accumulate as raw text
        if (line.trim()) textParts.push(line);
      }
    }

    return textParts.join('\n') || raw.trim();
  }

  private extractTokens(raw: string): { input: number; output: number } {
    const lines = raw.trim().split('\n');
    for (const line of lines) {
      try {
        const event = JSON.parse(line);
        if (event.usage) {
          return {
            input: event.usage.input_tokens || event.usage.prompt_tokens || 0,
            output: event.usage.output_tokens || event.usage.completion_tokens || 0,
          };
        }
      } catch { /* skip */ }
    }
    return { input: 0, output: 0 };
  }

  private attachAbortHandler(proc: ChildProcessWithoutNullStreams, signal?: AbortSignal): () => void {
    if (!signal) return () => {};
    if (signal.aborted) { proc.kill('SIGTERM'); return () => {}; }
    const onAbort = () => { proc.kill('SIGTERM'); };
    signal.addEventListener('abort', onAbort, { once: true });
    return () => { signal.removeEventListener('abort', onAbort); };
  }
}
