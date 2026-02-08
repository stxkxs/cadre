import { useState, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { Plus, Trash2, MessageSquare, TestTube, Pencil, Search, Bot } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogDescription,
} from '@/components/ui/dialog'
import { Pagination } from '@/components/ui/pagination'
import { ChatPanel } from '@/components/agents/ChatPanel'
import { useAgents, useCreateAgent, useUpdateAgent, useDeleteAgent } from '@/hooks/useAgents'
import { useProviders, useClaudeCodeStatus } from '@/hooks/useProviders'
import { usePagination } from '@/hooks/usePagination'
import { toolsApi } from '@/api/tools'
import type { Agent, Tool } from '@/types'

export function AgentsPage() {
  const { name } = useParams<{ name: string }>()
  const { data: agents, isLoading } = useAgents()
  const [chatAgent, setChatAgent] = useState<string | null>(name || null)
  const [showCreate, setShowCreate] = useState(false)
  const [editAgent, setEditAgent] = useState<Agent | null>(null)
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!agents) return []
    if (!search) return agents
    const q = search.toLowerCase()
    return agents.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.role.toLowerCase().includes(q) ||
        a.goal.toLowerCase().includes(q),
    )
  }, [agents, search])

  const pagination = usePagination(filtered, { pageSize: 12 })

  if (chatAgent) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" size="sm" onClick={() => setChatAgent(null)}>
          Back to agents
        </Button>
        <ChatPanel agentName={chatAgent} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold font-header tracking-tight">Agents</h1>
          {agents && (
            <Badge variant="secondary" className="text-xs font-mono">
              {agents.length}
            </Badge>
          )}
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button variant="accent" size="sm">
              <Plus className="h-4 w-4" /> New Agent
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create Agent</DialogTitle>
              <DialogDescription>Define a new AI agent with a role, goal, and tools.</DialogDescription>
            </DialogHeader>
            <AgentForm onSuccess={() => setShowCreate(false)} />
          </DialogContent>
        </Dialog>
      </div>

      {agents && agents.length > 0 && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search agents..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      )}

      {isLoading ? (
        <p className="text-muted-foreground">Loading agents...</p>
      ) : !agents?.length ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Bot className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No agents configured. Create one to get started.</p>
            </div>
          </CardContent>
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Search className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No agents matching &apos;{search}&apos;</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {pagination.pageItems.map((agent) => (
              <AgentCard
                key={agent.name}
                agent={agent}
                onChat={() => setChatAgent(agent.name)}
                onEdit={() => setEditAgent(agent)}
              />
            ))}
          </div>
          <Pagination
            page={pagination.page}
            totalPages={pagination.totalPages}
            totalItems={pagination.totalItems}
            startIndex={pagination.startIndex}
            endIndex={pagination.endIndex}
            canPrev={pagination.canPrev}
            canNext={pagination.canNext}
            onPrev={pagination.prevPage}
            onNext={pagination.nextPage}
            onPage={pagination.setPage}
          />
        </>
      )}

      <Dialog open={!!editAgent} onOpenChange={(open) => !open && setEditAgent(null)}>
        <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Agent</DialogTitle>
            <DialogDescription>Update agent configuration. Name cannot be changed.</DialogDescription>
          </DialogHeader>
          {editAgent && (
            <AgentEditForm agent={editAgent} onSuccess={() => setEditAgent(null)} />
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function AgentCard({
  agent,
  onChat,
  onEdit,
}: {
  agent: Agent
  onChat: () => void
  onEdit: () => void
}) {
  const deleteAgent = useDeleteAgent()

  return (
    <Card className="group hover:border-[var(--primary-accent)]/20 relative overflow-hidden">
      <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-[var(--primary-accent)]/40 opacity-0 group-hover:opacity-100 transition-opacity" />
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-sm font-semibold">{agent.name}</CardTitle>
            <p className="text-xs text-muted-foreground">{agent.role}</p>
          </div>
          <div className="flex gap-0.5">
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onChat} title="Chat">
              <MessageSquare className="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity" onClick={onEdit} title="Edit">
              <Pencil className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
              onClick={() => deleteAgent.mutate(agent.name)}
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-2.5">
        <p className="text-[11px] text-muted-foreground line-clamp-2">{agent.goal}</p>
        <div className="flex flex-wrap gap-1">
          {agent.tools?.map((tool) => (
            <Badge key={tool} variant="outline" className="text-[10px]">
              {tool}
            </Badge>
          ))}
        </div>
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
          <TestTube className="h-3 w-3" />
          {agent.memory?.type || 'conversation'}
          {agent.provider && agent.provider !== '' && (
            <Badge variant="accent" className="text-[10px] ml-auto">
              {agent.provider === 'claudecode' ? 'CLI' : 'API'}
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function AgentForm({ onSuccess }: { onSuccess: () => void }) {
  const createAgent = useCreateAgent()
  const { data: tools } = useQuery({ queryKey: ['tools'], queryFn: toolsApi.list })
  const { data: providers } = useProviders()
  const { data: ccStatus } = useClaudeCodeStatus()
  const [form, setForm] = useState({
    name: '',
    role: '',
    goal: '',
    backstory: '',
    selectedTools: ['file_read', 'file_write', 'bash', 'grep'] as string[],
    memoryType: 'conversation',
    maxTokens: '100000',
    provider: 'claudecode',
    providerModel: '',
    apiKey: '',
    workDir: '',
  })

  const toggleTool = (toolName: string) => {
    setForm((f) => ({
      ...f,
      selectedTools: f.selectedTools.includes(toolName)
        ? f.selectedTools.filter((t) => t !== toolName)
        : [...f.selectedTools, toolName],
    }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createAgent.mutate(
      {
        name: form.name,
        role: form.role,
        goal: form.goal,
        backstory: form.backstory,
        tools: form.selectedTools,
        memory: { type: form.memoryType, max_tokens: parseInt(form.maxTokens) || 100000 },
        provider: form.provider || undefined,
        provider_model: form.providerModel || undefined,
        api_key: form.apiKey || undefined,
        work_dir: form.workDir || undefined,
      },
      { onSuccess },
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
      <Input placeholder="Role" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} required />
      <Input placeholder="Goal" value={form.goal} onChange={(e) => setForm({ ...form, goal: e.target.value })} required />
      <Textarea placeholder="Backstory" value={form.backstory} onChange={(e) => setForm({ ...form, backstory: e.target.value })} rows={3} />
      <div>
        <label className="text-sm font-medium mb-2 block">Tools</label>
        <ToolSelector tools={tools} selected={form.selectedTools} onToggle={toggleTool} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-sm font-medium mb-1 block">Memory</label>
          <Select
            value={form.memoryType}
            onChange={(e) => setForm({ ...form, memoryType: e.target.value })}
          >
            <option value="conversation">conversation</option>
            <option value="long_term">long_term</option>
            <option value="shared">shared</option>
          </Select>
        </div>
        <div>
          <label className="text-sm font-medium mb-1 block">Max Tokens</label>
          <Input
            type="number"
            value={form.maxTokens}
            onChange={(e) => setForm({ ...form, maxTokens: e.target.value })}
          />
        </div>
      </div>
      <ProviderFields
        provider={form.provider}
        providerModel={form.providerModel}
        apiKey={form.apiKey}
        workDir={form.workDir}
        providers={providers}
        ccStatus={ccStatus}
        onChange={(fields) => setForm({ ...form, ...fields })}
      />
      <Button type="submit" variant="accent" disabled={createAgent.isPending} className="w-full">
        {createAgent.isPending ? 'Creating...' : 'Create Agent'}
      </Button>
    </form>
  )
}

function AgentEditForm({ agent, onSuccess }: { agent: Agent; onSuccess: () => void }) {
  const updateAgent = useUpdateAgent()
  const { data: tools } = useQuery({ queryKey: ['tools'], queryFn: toolsApi.list })
  const { data: providers } = useProviders()
  const { data: ccStatus } = useClaudeCodeStatus()
  const [form, setForm] = useState({
    role: agent.role,
    goal: agent.goal,
    backstory: agent.backstory,
    compact_backstory: agent.compact_backstory || '',
    selectedTools: agent.tools || [],
    memoryType: agent.memory?.type || 'conversation',
    maxTokens: String(agent.memory?.max_tokens || 100000),
    provider: agent.provider || '',
    providerModel: agent.provider_model || '',
    apiKey: agent.api_key || '',
    workDir: agent.work_dir || '',
  })

  const toggleTool = (toolName: string) => {
    setForm((f) => ({
      ...f,
      selectedTools: f.selectedTools.includes(toolName)
        ? f.selectedTools.filter((t) => t !== toolName)
        : [...f.selectedTools, toolName],
    }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateAgent.mutate(
      {
        name: agent.name,
        agent: {
          name: agent.name,
          role: form.role,
          goal: form.goal,
          backstory: form.backstory,
          compact_backstory: form.compact_backstory || undefined,
          tools: form.selectedTools,
          memory: { type: form.memoryType, max_tokens: parseInt(form.maxTokens) || 100000 },
          provider: form.provider || undefined,
          provider_model: form.providerModel || undefined,
          api_key: form.apiKey || undefined,
          work_dir: form.workDir || undefined,
        },
      },
      { onSuccess },
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="text-sm font-medium mb-1 block">Name</label>
        <Input value={agent.name} disabled className="opacity-60" />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Role</label>
        <Input value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} required />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Goal</label>
        <Input value={form.goal} onChange={(e) => setForm({ ...form, goal: e.target.value })} required />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Backstory</label>
        <Textarea value={form.backstory} onChange={(e) => setForm({ ...form, backstory: e.target.value })} rows={3} />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Compact Backstory</label>
        <Textarea
          value={form.compact_backstory}
          onChange={(e) => setForm({ ...form, compact_backstory: e.target.value })}
          rows={2}
          placeholder="Shorter backstory for context-limited scenarios"
        />
      </div>
      <div>
        <label className="text-sm font-medium mb-2 block">Tools</label>
        <ToolSelector tools={tools} selected={form.selectedTools} onToggle={toggleTool} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-sm font-medium mb-1 block">Memory</label>
          <Select
            value={form.memoryType}
            onChange={(e) => setForm({ ...form, memoryType: e.target.value })}
          >
            <option value="conversation">conversation</option>
            <option value="long_term">long_term</option>
            <option value="shared">shared</option>
          </Select>
        </div>
        <div>
          <label className="text-sm font-medium mb-1 block">Max Tokens</label>
          <Input
            type="number"
            value={form.maxTokens}
            onChange={(e) => setForm({ ...form, maxTokens: e.target.value })}
          />
        </div>
      </div>
      <ProviderFields
        provider={form.provider}
        providerModel={form.providerModel}
        apiKey={form.apiKey}
        workDir={form.workDir}
        providers={providers}
        ccStatus={ccStatus}
        onChange={(fields) => setForm({ ...form, ...fields })}
      />
      <Button type="submit" variant="accent" disabled={updateAgent.isPending} className="w-full">
        {updateAgent.isPending ? 'Saving...' : 'Save Changes'}
      </Button>
    </form>
  )
}

function ToolSelector({
  tools,
  selected,
  onToggle,
}: {
  tools: Tool[] | undefined
  selected: string[]
  onToggle: (name: string) => void
}) {
  const allTools = tools?.length
    ? tools
    : selected.map((name) => ({ name, description: '', provider: '' }))

  const unique = Array.from(new Set([...allTools.map((t) => t.name), ...selected]))

  return (
    <div className="flex flex-wrap gap-1.5">
      {unique.map((name) => (
        <button
          key={name}
          type="button"
          onClick={() => onToggle(name)}
          className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors cursor-pointer ${
            selected.includes(name)
              ? 'bg-[var(--primary-accent)] text-white border-transparent'
              : 'bg-background text-muted-foreground border-input hover:border-[var(--primary-accent)]/50'
          }`}
        >
          {name}
        </button>
      ))}
    </div>
  )
}

function ProviderFields({
  provider,
  providerModel,
  apiKey,
  workDir,
  providers,
  ccStatus,
  onChange,
}: {
  provider: string
  providerModel: string
  apiKey: string
  workDir: string
  providers: import('@/types').ProviderInfo[] | undefined
  ccStatus: import('@/types').ClaudeCodeStatus | undefined
  onChange: (fields: { provider?: string; providerModel?: string; apiKey?: string; workDir?: string }) => void
}) {
  return (
    <div className="space-y-3">
      <div>
        <label className="text-sm font-medium mb-1 block">Provider</label>
        <Select
          value={provider}
          onChange={(e) => onChange({ provider: e.target.value, apiKey: '' })}
        >
          <option value="">Default (Anthropic API)</option>
          {providers?.map((p) => (
            <option key={p.name} value={p.name}>
              {p.label}
            </option>
          ))}
        </Select>
        {provider === 'claudecode' && ccStatus && !ccStatus.available && (
          <p className="text-xs text-destructive mt-1">
            Claude CLI not found. Install it to use this provider.
          </p>
        )}
        {provider === 'claudecode' && ccStatus?.available && (
          <p className="text-xs text-muted-foreground mt-1">
            Found at {ccStatus.path}
          </p>
        )}
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Model Override</label>
        <Input
          placeholder="Leave empty for default"
          value={providerModel}
          onChange={(e) => onChange({ providerModel: e.target.value })}
        />
      </div>
      {(provider === '' || provider === 'anthropic') && (
        <div>
          <label className="text-sm font-medium mb-1 block">API Key Override</label>
          <Input
            type="password"
            placeholder="Leave empty for global key"
            value={apiKey}
            onChange={(e) => onChange({ apiKey: e.target.value })}
          />
        </div>
      )}
      {provider === 'claudecode' && (
        <div>
          <label className="text-sm font-medium mb-1 block">Working Directory</label>
          <Input
            placeholder="Leave empty for global default"
            value={workDir}
            onChange={(e) => onChange({ workDir: e.target.value })}
          />
        </div>
      )}
    </div>
  )
}
