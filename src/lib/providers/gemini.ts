import { spawn, type ChildProcessWithoutNullStreams } from 'child_process';
import type { CodingAgentProvider } from './base';
import type { ProviderMessage, ProviderOptions, StreamChunk } from '@/types/provider';

export class GeminiProvider implements CodingAgentProvider {
  readonly id = 'gemini';
  readonly name = 'Gemini';

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async *stream(messages: ProviderMessage[], options: ProviderOptions, _apiKey: string): AsyncGenerator<StreamChunk> {
    const prompt = this.buildPrompt(messages);

    const args = ['-p', prompt, '--output-format', 'stream-json'];

    if (options.workspace === 'full') {
      args.push('--yolo');
    }

    const spawnOptions: { env: NodeJS.ProcessEnv; cwd?: string } = {
      env: process.env,
    };

    if (options.workspacePath) {
      spawnOptions.cwd = options.workspacePath;
    }

    const proc = spawn('gemini', args, spawnOptions);

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
        throw new Error('Gemini CLI not found. Install it with: npm install -g @google/gemini-cli');
      }
      throw err;
    });

    const result = await closePromise;
    cleanup();

    if (result.signal === 'SIGTERM' || options.signal?.aborted) {
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error('Gemini was cancelled');
    }

    // Exit code 53 = turn limit exceeded — still produce output
    if (result.code !== 0 && result.code !== 53) {
      if (stdout.trim()) {
        yield { type: 'text', content: this.parseOutput(stdout) };
      }
      throw new Error(stderr || `Gemini exited with code ${result.code}`);
    }

    const parsed = this.parseOutput(stdout);
    yield { type: 'text', content: parsed };
    yield { type: 'done', content: '', tokens: this.extractTokens(stdout) };
  }

  async validateCli(): Promise<boolean> {
    return new Promise((resolve) => {
      const proc = spawn('gemini', ['--version']);
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
    // Gemini stream-json outputs JSONL events. Extract assistant message content.
    const lines = raw.trim().split('\n');
    const textParts: string[] = [];

    for (const line of lines) {
      try {
        const event = JSON.parse(line);
        // Assistant message chunks
        if (event.type === 'message' && event.role === 'model' && typeof event.content === 'string') {
          textParts.push(event.content);
        }
        // Final result event may contain aggregated response
        if (event.type === 'result' && typeof event.response === 'string') {
          return event.response;
        }
      } catch {
        if (line.trim()) textParts.push(line);
      }
    }

    return textParts.join('') || raw.trim();
  }

  private extractTokens(raw: string): { input: number; output: number } {
    const lines = raw.trim().split('\n');
    for (const line of lines) {
      try {
        const event = JSON.parse(line);
        if (event.type === 'result' && event.stats) {
          return {
            input: event.stats.inputTokens || event.stats.input_tokens || 0,
            output: event.stats.outputTokens || event.stats.output_tokens || 0,
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
