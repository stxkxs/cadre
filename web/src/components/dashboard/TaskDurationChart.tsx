import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { BarChart } from '@/components/charts/BarChart'
import { getChartColor } from '@/components/charts/chart-utils'

interface TaskDurationStat {
  name: string
  avgDuration: number
}

interface TaskDurationChartProps {
  taskDurations: TaskDurationStat[]
}

export function TaskDurationChart({ taskDurations }: TaskDurationChartProps) {
  const items = taskDurations.map((t, i) => ({
    label: t.name,
    value: Math.round(t.avgDuration / 1000), // convert to seconds
    color: getChartColor(i),
  }))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Avg Task Duration (s)</CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-xs text-muted-foreground">No task duration data yet.</p>
        ) : (
          <BarChart items={items} />
        )}
      </CardContent>
    </Card>
  )
}
