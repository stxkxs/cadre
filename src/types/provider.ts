export interface ProviderMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface ProviderOptions {
  model?: string;
  workspacePath?: string;
  workspace?: 'off' | 'safe' | 'full';
  maxTurns?: number;
  signal?: AbortSignal;
}

export interface StreamChunk {
  type: 'text' | 'done';
  content: string;
  tokens?: { input: number; output: number };
}

export interface ProviderConfig {
  id: string;
  name: string;
  color: string;
}

export const PROVIDERS: ProviderConfig[] = [
  { id: 'claude-code', name: 'Claude Code', color: '#6366f1' },
  { id: 'codex', name: 'Codex', color: '#10a37f' },
  { id: 'gemini', name: 'Gemini', color: '#4285f4' },
];
