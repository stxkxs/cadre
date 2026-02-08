import { api } from './client'
import type { ProviderInfo, ClaudeCodeStatus } from '@/types'

export const providersApi = {
  list: () => api.get<ProviderInfo[]>('/providers'),
  claudeCodeStatus: () => api.get<ClaudeCodeStatus>('/providers/claudecode/status'),
}
