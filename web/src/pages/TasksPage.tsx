import { useState, useMemo } from 'react'
import { Plus, Trash2, Pencil, Search, X, ListTodo } from 'lucide-react'
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
import { useTasks, useCreateTask, useUpdateTask, useDeleteTask } from '@/hooks/useTasks'
import { useAgents } from '@/hooks/useAgents'
import { usePagination } from '@/hooks/usePagination'
import type { Task, TaskInput, TaskOutput } from '@/types'

export function TasksPage() {
  const { data: tasks, isLoading } = useTasks()
  const [showCreate, setShowCreate] = useState(false)
  const [editTask, setEditTask] = useState<Task | null>(null)
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!tasks) return []
    if (!search) return tasks
    const q = search.toLowerCase()
    return tasks.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.agent.toLowerCase().includes(q),
    )
  }, [tasks, search])

  const pagination = usePagination(filtered, { pageSize: 12 })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold font-header tracking-tight">Tasks</h1>
          {tasks && (
            <Badge variant="secondary" className="text-xs font-mono">
              {tasks.length}
            </Badge>
          )}
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button variant="accent" size="sm">
              <Plus className="h-4 w-4" /> New Task
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Create Task</DialogTitle>
              <DialogDescription>Define a new task with an agent assignment and configuration.</DialogDescription>
            </DialogHeader>
            <TaskForm onSuccess={() => setShowCreate(false)} />
          </DialogContent>
        </Dialog>
      </div>

      {tasks && tasks.length > 0 && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search tasks..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      )}

      {isLoading ? (
        <p className="text-muted-foreground">Loading tasks...</p>
      ) : !tasks?.length ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <ListTodo className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No tasks configured. Create one to get started.</p>
            </div>
          </CardContent>
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Search className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No tasks matching &apos;{search}&apos;</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {pagination.pageItems.map((task) => (
              <TaskCard key={task.name} task={task} onEdit={() => setEditTask(task)} />
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

      <Dialog open={!!editTask} onOpenChange={(open) => !open && setEditTask(null)}>
        <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Task</DialogTitle>
            <DialogDescription>Update task configuration. Name cannot be changed.</DialogDescription>
          </DialogHeader>
          {editTask && (
            <TaskEditForm task={editTask} onSuccess={() => setEditTask(null)} />
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function TaskCard({ task, onEdit }: { task: Task; onEdit: () => void }) {
  const deleteTask = useDeleteTask()

  return (
    <Card className="group hover:border-[var(--primary-accent)]/20 relative overflow-hidden">
      <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-[var(--primary-accent)]/40 opacity-0 group-hover:opacity-100 transition-opacity" />
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <CardTitle className="text-sm font-semibold">{task.name}</CardTitle>
          <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onEdit} title="Edit">
              <Pencil className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive"
              onClick={() => deleteTask.mutate(task.name)}
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-2.5">
        <p className="text-[11px] text-muted-foreground line-clamp-2">{task.description}</p>
        <div className="flex items-center gap-2">
          <Badge variant="accent">{task.agent}</Badge>
          {task.timeout && (
            <Badge variant="secondary" className="text-[10px]">{task.timeout}</Badge>
          )}
        </div>
        {task.dependencies?.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {task.dependencies.map((dep) => (
              <Badge key={dep} variant="outline" className="text-[10px]">
                {dep}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function TaskForm({ onSuccess }: { onSuccess: () => void }) {
  const createTask = useCreateTask()
  const { data: agents } = useAgents()
  const { data: allTasks } = useTasks()
  const [form, setForm] = useState({
    name: '',
    description: '',
    agent: '',
    timeout: '30m',
    dependencies: [] as string[],
    inputs: [] as TaskInput[],
    outputs: [] as TaskOutput[],
    maxAttempts: '3',
    backoff: 'exponential',
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createTask.mutate(
      {
        name: form.name,
        description: form.description,
        agent: form.agent,
        inputs: form.inputs,
        outputs: form.outputs,
        dependencies: form.dependencies,
        timeout: form.timeout,
        retry: { max_attempts: parseInt(form.maxAttempts) || 3, backoff: form.backoff },
      },
      { onSuccess },
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
      <Textarea placeholder="Description" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={3} required />
      <div>
        <label className="text-sm font-medium mb-1 block">Agent</label>
        <Select
          value={form.agent}
          onChange={(e) => setForm({ ...form, agent: e.target.value })}
          required
        >
          <option value="">Select agent...</option>
          {agents?.map((a) => (
            <option key={a.name} value={a.name}>{a.name}</option>
          ))}
        </Select>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-sm font-medium mb-1 block">Timeout</label>
          <Input value={form.timeout} onChange={(e) => setForm({ ...form, timeout: e.target.value })} />
        </div>
        <div>
          <label className="text-sm font-medium mb-1 block">Retry</label>
          <div className="flex gap-1">
            <Input
              type="number"
              value={form.maxAttempts}
              onChange={(e) => setForm({ ...form, maxAttempts: e.target.value })}
              className="w-16"
              min={1}
            />
            <Select
              className="flex-1"
              value={form.backoff}
              onChange={(e) => setForm({ ...form, backoff: e.target.value })}
            >
              <option value="exponential">exponential</option>
              <option value="fixed">fixed</option>
            </Select>
          </div>
        </div>
      </div>
      {allTasks && allTasks.length > 0 && (
        <div>
          <label className="text-sm font-medium mb-1 block">Dependencies</label>
          <DependencySelector
            tasks={allTasks}
            selected={form.dependencies}
            exclude={form.name}
            onChange={(deps) => setForm({ ...form, dependencies: deps })}
          />
        </div>
      )}
      <div>
        <label className="text-sm font-medium mb-1 block">Inputs</label>
        <IOFieldList items={form.inputs} onChange={(inputs) => setForm({ ...form, inputs })} type="input" />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Outputs</label>
        <IOFieldList items={form.outputs} onChange={(outputs) => setForm({ ...form, outputs })} type="output" />
      </div>
      <Button type="submit" variant="accent" disabled={createTask.isPending} className="w-full">
        {createTask.isPending ? 'Creating...' : 'Create Task'}
      </Button>
    </form>
  )
}

function TaskEditForm({ task, onSuccess }: { task: Task; onSuccess: () => void }) {
  const updateTask = useUpdateTask()
  const { data: agents } = useAgents()
  const { data: allTasks } = useTasks()
  const [form, setForm] = useState({
    description: task.description,
    agent: task.agent,
    timeout: task.timeout || '30m',
    dependencies: task.dependencies || [],
    inputs: task.inputs || [],
    outputs: task.outputs || [],
    maxAttempts: String(task.retry?.max_attempts || 3),
    backoff: task.retry?.backoff || 'exponential',
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateTask.mutate(
      {
        name: task.name,
        task: {
          name: task.name,
          description: form.description,
          agent: form.agent,
          inputs: form.inputs,
          outputs: form.outputs,
          dependencies: form.dependencies,
          timeout: form.timeout,
          retry: { max_attempts: parseInt(form.maxAttempts) || 3, backoff: form.backoff },
        },
      },
      { onSuccess },
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="text-sm font-medium mb-1 block">Name</label>
        <Input value={task.name} disabled className="opacity-60" />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Description</label>
        <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={3} required />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Agent</label>
        <Select
          value={form.agent}
          onChange={(e) => setForm({ ...form, agent: e.target.value })}
          required
        >
          <option value="">Select agent...</option>
          {agents?.map((a) => (
            <option key={a.name} value={a.name}>{a.name}</option>
          ))}
        </Select>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-sm font-medium mb-1 block">Timeout</label>
          <Input value={form.timeout} onChange={(e) => setForm({ ...form, timeout: e.target.value })} />
        </div>
        <div>
          <label className="text-sm font-medium mb-1 block">Retry</label>
          <div className="flex gap-1">
            <Input
              type="number"
              value={form.maxAttempts}
              onChange={(e) => setForm({ ...form, maxAttempts: e.target.value })}
              className="w-16"
              min={1}
            />
            <Select
              className="flex-1"
              value={form.backoff}
              onChange={(e) => setForm({ ...form, backoff: e.target.value })}
            >
              <option value="exponential">exponential</option>
              <option value="fixed">fixed</option>
            </Select>
          </div>
        </div>
      </div>
      {allTasks && allTasks.length > 0 && (
        <div>
          <label className="text-sm font-medium mb-1 block">Dependencies</label>
          <DependencySelector
            tasks={allTasks}
            selected={form.dependencies}
            exclude={task.name}
            onChange={(deps) => setForm({ ...form, dependencies: deps })}
          />
        </div>
      )}
      <div>
        <label className="text-sm font-medium mb-1 block">Inputs</label>
        <IOFieldList items={form.inputs} onChange={(inputs) => setForm({ ...form, inputs })} type="input" />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">Outputs</label>
        <IOFieldList items={form.outputs} onChange={(outputs) => setForm({ ...form, outputs })} type="output" />
      </div>
      <Button type="submit" variant="accent" disabled={updateTask.isPending} className="w-full">
        {updateTask.isPending ? 'Saving...' : 'Save Changes'}
      </Button>
    </form>
  )
}

function DependencySelector({
  tasks,
  selected,
  exclude,
  onChange,
}: {
  tasks: Task[]
  selected: string[]
  exclude: string
  onChange: (deps: string[]) => void
}) {
  const available = tasks.filter((t) => t.name !== exclude)

  return (
    <div className="flex flex-wrap gap-1.5">
      {available.map((t) => (
        <button
          key={t.name}
          type="button"
          onClick={() =>
            onChange(
              selected.includes(t.name)
                ? selected.filter((d) => d !== t.name)
                : [...selected, t.name],
            )
          }
          className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors cursor-pointer ${
            selected.includes(t.name)
              ? 'bg-[var(--primary-accent)] text-white border-transparent'
              : 'bg-background text-muted-foreground border-input hover:border-[var(--primary-accent)]/50'
          }`}
        >
          {t.name}
        </button>
      ))}
    </div>
  )
}

function IOFieldList<T extends TaskInput | TaskOutput>({
  items,
  onChange,
  type,
}: {
  items: T[]
  onChange: (items: T[]) => void
  type: 'input' | 'output'
}) {
  const addItem = () => {
    if (type === 'input') {
      onChange([...items, { name: '', type: 'string', required: true } as T])
    } else {
      onChange([...items, { name: '', type: 'string' } as T])
    }
  }

  const removeItem = (index: number) => {
    onChange(items.filter((_, i) => i !== index))
  }

  const updateItem = (index: number, field: string, value: string | boolean) => {
    onChange(items.map((item, i) => (i === index ? { ...item, [field]: value } : item)))
  }

  return (
    <div className="space-y-2">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-2">
          <Input
            placeholder="name"
            value={item.name}
            onChange={(e) => updateItem(i, 'name', e.target.value)}
            className="flex-1"
          />
          <Select
            className="w-24"
            value={item.type}
            onChange={(e) => updateItem(i, 'type', e.target.value)}
          >
            <option value="string">string</option>
            <option value="string[]">string[]</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
          </Select>
          {type === 'input' && (
            <label className="flex items-center gap-1 text-xs whitespace-nowrap">
              <input
                type="checkbox"
                checked={(item as TaskInput).required}
                onChange={(e) => updateItem(i, 'required', e.target.checked)}
                className="rounded"
              />
              req
            </label>
          )}
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => removeItem(i)}>
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="ghost" size="sm" onClick={addItem}>
        <Plus className="h-3.5 w-3.5" /> Add {type}
      </Button>
    </div>
  )
}
