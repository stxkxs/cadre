import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { AppShell } from '@/components/layout/AppShell'
import { Dashboard } from '@/pages/Dashboard'
import { AgentsPage } from '@/pages/AgentsPage'
import { TasksPage } from '@/pages/TasksPage'
import { CrewsPage } from '@/pages/CrewsPage'
import { ComposerPage } from '@/pages/ComposerPage'
import { RunsPage } from '@/pages/RunsPage'
import { RunDetailPage } from '@/pages/RunDetailPage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: 1,
    },
  },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<Dashboard />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="agents/:name" element={<AgentsPage />} />
            <Route path="tasks" element={<TasksPage />} />
            <Route path="crews" element={<CrewsPage />} />
            <Route path="composer" element={<ComposerPage />} />
            <Route path="composer/:name" element={<ComposerPage />} />
            <Route path="runs" element={<RunsPage />} />
            <Route path="runs/:id" element={<RunDetailPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
      <Toaster position="bottom-right" richColors theme="dark" />
    </QueryClientProvider>
  )
}
