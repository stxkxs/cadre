import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { ModelProvider } from '@/lib/engine/types';

interface ApiKeyStatus {
  isConfigured: boolean;
  isValid: boolean;
  lastValidated?: Date;
}

interface SettingsStore {
  theme: 'dark' | 'light' | 'system';
  sidebarCollapsed: boolean;
  apiKeyStatuses: Record<ModelProvider, ApiKeyStatus>;

  setTheme: (theme: SettingsStore['theme']) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setApiKeyStatus: (provider: ModelProvider, status: ApiKeyStatus) => void;
}

const defaultApiKeyStatus: ApiKeyStatus = { isConfigured: false, isValid: false };

export const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      theme: 'dark',
      sidebarCollapsed: false,
      apiKeyStatuses: {
        anthropic: defaultApiKeyStatus,
        openai: defaultApiKeyStatus,
        groq: defaultApiKeyStatus,
        'claude-code': defaultApiKeyStatus,
        bedrock: defaultApiKeyStatus,
      },

      setTheme: (theme) => set({ theme }),
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      setApiKeyStatus: (provider, status) => set((s) => ({
        apiKeyStatuses: { ...s.apiKeyStatuses, [provider]: status },
      })),
    }),
    { name: 'cadre-settings' }
  )
);
