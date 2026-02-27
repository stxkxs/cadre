import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DonutChart } from '@/components/charts/DonutChart'

interface RunSuccessChartProps {
  statusCounts: { completed: number; failed: number; cancelled: number; running: number; pending: number }
}

export function RunSuccessChart({ statusCounts }: RunSuccessChartProps) {
  const segments = [
    { label: 'Completed', value: statusCounts.completed, color: 'var(--accent-green)' },
    { label: 'Failed', value: statusCounts.failed, color: 'var(--destructive)' },
    { label: 'Cancelled', value: statusCounts.cancelled, color: 'var(--muted-foreground)' },
    { label: 'Running', value: statusCounts.running, color: 'var(--accent-blue)' },
    { label: 'Pending', value: statusCounts.pending, color: 'var(--chart-4)' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>Run Results</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-6">
          <DonutChart segments={segments} />
          <div className="space-y-2">
            {segments.filter((s) => s.value > 0).map((seg) => (
              <div key={seg.label} className="flex items-center gap-2">
                <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: seg.color }} />
                <span className="text-xs text-muted-foreground">{seg.label}</span>
                <span className="text-xs font-mono font-semibold ml-auto">{seg.value}</span>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
