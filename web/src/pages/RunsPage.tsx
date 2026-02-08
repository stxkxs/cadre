import { Inbox } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Pagination } from '@/components/ui/pagination'
import { useRuns } from '@/hooks/useRuns'
import { usePagination } from '@/hooks/usePagination'

export function RunsPage() {
  const { data: runs, isLoading } = useRuns()
  const pagination = usePagination(runs, { pageSize: 20 })

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-bold font-header tracking-tight">Runs</h1>
        {runs && (
          <Badge variant="secondary" className="text-xs font-mono">
            {runs.length}
          </Badge>
        )}
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">Loading runs...</p>
      ) : !runs?.length ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="flex flex-col items-center gap-2">
              <Inbox className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No runs yet. Start a crew to see runs here.</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>All Runs</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {pagination.pageItems.map((run) => (
                <Link
                  key={run.id}
                  to={`/runs/${run.id}`}
                  className="flex items-center justify-between rounded-md border p-3 transition-colors hover:bg-accent"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-medium">{run.crew_name}</span>
                    <StatusBadge status={run.status} />
                    <span className="text-xs text-muted-foreground font-mono">
                      {run.id.slice(0, 8)}
                    </span>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>{run.tasks?.length || 0} tasks</span>
                    <span>{new Date(run.started_at).toLocaleString()}</span>
                  </div>
                </Link>
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
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const variant = {
    running: 'accent' as const,
    completed: 'success' as const,
    failed: 'destructive' as const,
    pending: 'secondary' as const,
    cancelled: 'outline' as const,
  }[status] || ('secondary' as const)

  return <Badge variant={variant}>{status}</Badge>
}
