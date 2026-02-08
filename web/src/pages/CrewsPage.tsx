import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Trash2, Play, Workflow, Search, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Pagination } from '@/components/ui/pagination'
import { useCrews, useDeleteCrew } from '@/hooks/useCrews'
import { useStartRun } from '@/hooks/useRuns'
import { usePagination } from '@/hooks/usePagination'
import type { Crew } from '@/types'

export function CrewsPage() {
  const { data: crews, isLoading } = useCrews()
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!crews) return []
    if (!search) return crews
    const q = search.toLowerCase()
    return crews.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.description?.toLowerCase().includes(q) ||
        c.process.toLowerCase().includes(q),
    )
  }, [crews, search])

  const pagination = usePagination(filtered, { pageSize: 12 })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold font-header tracking-tight">Crews</h1>
          {crews && (
            <Badge variant="secondary" className="text-xs font-mono">
              {crews.length}
            </Badge>
          )}
        </div>
        <Link to="/composer">
          <Button variant="accent" size="sm">
            <Plus className="h-4 w-4" /> New Crew
          </Button>
        </Link>
      </div>

      {crews && crews.length > 0 && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search crews..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      )}

      {isLoading ? (
        <p className="text-muted-foreground">Loading crews...</p>
      ) : !crews?.length ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Users className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No crews configured. Open the Composer to create one.</p>
            </div>
          </CardContent>
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Search className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No crews matching &apos;{search}&apos;</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {pagination.pageItems.map((crew) => (
              <CrewCard key={crew.name} crew={crew} />
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
    </div>
  )
}

function CrewCard({ crew }: { crew: Crew }) {
  const deleteCrew = useDeleteCrew()
  const startRun = useStartRun()
  const [running, setRunning] = useState(false)

  const handleRun = () => {
    setRunning(true)
    startRun.mutate(
      { crew: crew.name },
      { onSettled: () => setRunning(false) },
    )
  }

  const processColor = {
    sequential: 'accent-blue',
    parallel: 'accent-green',
    hierarchical: 'accent-purple',
  }[crew.process] || 'accent-blue'

  return (
    <Card className="group hover:border-[var(--primary-accent)]/20 relative overflow-hidden">
      <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-[var(--primary-accent)]/40 opacity-0 group-hover:opacity-100 transition-opacity" />
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-sm font-semibold">{crew.name}</CardTitle>
            {crew.description && (
              <p className="text-[11px] text-muted-foreground line-clamp-2">{crew.description}</p>
            )}
          </div>
          <div className="flex gap-0.5">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={handleRun}
              disabled={running}
              title="Run"
            >
              <Play className="h-3.5 w-3.5" />
            </Button>
            <Link to={`/composer/${crew.name}`}>
              <Button variant="ghost" size="icon" className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity" title="Edit in Composer">
                <Workflow className="h-3.5 w-3.5" />
              </Button>
            </Link>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
              onClick={() => deleteCrew.mutate(crew.name)}
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-2.5">
        <div className="flex items-center gap-2">
          <Badge
            variant="accent"
            className={`bg-[var(--${processColor})]/15 text-[var(--${processColor})] border-transparent`}
          >
            {crew.process}
          </Badge>
          <span className="text-[11px] font-mono text-muted-foreground">
            {crew.tasks?.length || 0} tasks
          </span>
        </div>
        <div className="flex flex-wrap gap-1">
          {crew.agents?.map((agent) => (
            <div key={agent} className="flex items-center gap-1.5">
              <div className="h-5 w-5 rounded-full bg-secondary flex items-center justify-center">
                <span className="text-[9px] font-mono font-medium uppercase">{agent.slice(0, 2)}</span>
              </div>
              <span className="text-[11px] text-muted-foreground">{agent}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
