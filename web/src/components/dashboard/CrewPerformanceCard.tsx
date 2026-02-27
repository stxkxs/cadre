import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatDuration } from '@/lib/time-utils'

interface CrewStat {
  name: string
  total: number
  successRate: number
  avgDuration: number
}

interface CrewPerformanceCardProps {
  crewStats: CrewStat[]
}

export function CrewPerformanceCard({ crewStats }: CrewPerformanceCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Crew Performance</CardTitle>
      </CardHeader>
      <CardContent>
        {crewStats.length === 0 ? (
          <p className="text-xs text-muted-foreground">No crew data yet.</p>
        ) : (
          <div className="space-y-3">
            {crewStats.slice(0, 5).map((stat) => (
              <div key={stat.name} className="space-y-1">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-mono font-semibold">{stat.name}</span>
                  <div className="flex items-center gap-3 text-[10px] text-muted-foreground font-mono">
                    <span>{stat.total} runs</span>
                    {stat.avgDuration > 0 && <span>{formatDuration(stat.avgDuration)}</span>}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-2 bg-muted/50 rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full bg-[var(--accent-green)] transition-all duration-500"
                      style={{ width: `${stat.successRate}%` }}
                    />
                  </div>
                  <span className="text-[10px] font-mono text-muted-foreground w-10 text-right">
                    {stat.successRate.toFixed(0)}%
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
