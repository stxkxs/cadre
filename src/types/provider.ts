import type { ModelProvider } from '@/lib/engine/types';

export interface ProviderConfig {
  id: ModelProvider;
  name: string;
  models: ModelConfig[];
  color: string;
  icon: string;
  dynamicModels?: boolean;
  noApiKey?: boolean;
}

export interface ModelConfig {
  id: string;
  name: string;
  maxTokens: number;
  supportsStreaming: boolean;
  supportsTools: boolean;
  inputCostPer1k: number;
  outputCostPer1k: number;
}

export interface ApiKeyConfig {
  provider: ModelProvider;
  isConfigured: boolean;
  isValid: boolean;
  lastValidated?: Date;
}

export const PROVIDER_CONFIGS: ProviderConfig[] = [
  {
    id: 'anthropic',
    name: 'Anthropic (Claude)',
    color: '#f97316',
    icon: 'brain',
    models: [
      { id: 'claude-sonnet-4-6', name: 'Claude Sonnet 4.6', maxTokens: 8192, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.003, outputCostPer1k: 0.015 },
      { id: 'claude-haiku-4-5', name: 'Claude Haiku 4.5', maxTokens: 8192, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.001, outputCostPer1k: 0.005 },
      { id: 'claude-opus-4-6', name: 'Claude Opus 4.6', maxTokens: 8192, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.015, outputCostPer1k: 0.075 },
    ],
  },
  {
    id: 'openai',
    name: 'OpenAI (ChatGPT)',
    color: '#22c55e',
    icon: 'sparkles',
    models: [
      { id: 'gpt-4o', name: 'GPT-4o', maxTokens: 4096, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.005, outputCostPer1k: 0.015 },
      { id: 'gpt-4o-mini', name: 'GPT-4o Mini', maxTokens: 4096, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.00015, outputCostPer1k: 0.0006 },
    ],
  },
  {
    id: 'groq',
    name: 'Groq (Llama)',
    color: '#a855f7',
    icon: 'zap',
    models: [
      { id: 'llama-3.3-70b-versatile', name: 'Llama 3.3 70B', maxTokens: 4096, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.00059, outputCostPer1k: 0.00079 },
      { id: 'llama-3.1-8b-instant', name: 'Llama 3.1 8B', maxTokens: 4096, supportsStreaming: true, supportsTools: false, inputCostPer1k: 0.00005, outputCostPer1k: 0.00008 },
    ],
  },
  {
    id: 'claude-code',
    name: 'Claude Code (Terminal)',
    color: '#3b82f6',
    icon: 'terminal',
    noApiKey: true,
    models: [
      { id: 'claude-code', name: 'Claude Code', maxTokens: 8192, supportsStreaming: true, supportsTools: true, inputCostPer1k: 0.003, outputCostPer1k: 0.015 },
    ],
  },
  {
    id: 'bedrock',
    name: 'AWS Bedrock',
    color: '#f59e0b',
    icon: 'cloud',
    dynamicModels: true,
    noApiKey: true,
    models: [],
  },
];
