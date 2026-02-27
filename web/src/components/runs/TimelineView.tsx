import { useMemo } from 'react'
import { formatDuration, durationBetween } from '@/lib/time-utils'
import type { TaskState } from '@/types'

interface TimelineViewProps {
  tasks: TaskState[]
  runStartedAt: string
}

const statusColors: Record<string, string> = {
  completed: 'var(--accent-green)',
  failed: 'var(--destructive)',
  running: 'var(--primary-accent)',
  pending: 'var(--muted-foreground)',
  skipped: 'var(--muted-foreground)',
}

export function TimelineView({ tasks, runStartedAt }: TimelineViewProps) {
  const { rows, totalDuration } = useMemo(() => {
    const runStart = new Date(runStartedAt).getTime()
    let maxEnd = runStart

    const rows = tasks.map((task) => {
      const start = task.started_at ? new Date(task.started_at).getTime() : runStart
      const end = task.completed_at ? new Date(task.completed_at).getTime() : Date.now()
      if (end > maxEnd) maxEnd = end
      return { task, start, end }
    })

    return { rows, totalDuration: maxEnd - runStart || 1 }
  }, [tasks, runStartedAt])

  const runStart = new Date(runStartedAt).getTime()
  const rowHeight = 40
  const labelWidth = 140
  const chartWidth = 600
  const svgHeight = rows.length * rowHeight + 20

  return (
    <div className="overflow-x-auto">
      <svg width={labelWidth + chartWidth + 20} height={svgHeight} className="font-mono">
        {rows.map(({ task, start, end }, i) => {
          const y = i * rowHeight + 10
          const barStart = ((start - runStart) / totalDuration) * chartWidth
          const barWidth = Math.max(((end - start) / totalDuration) * chartWidth, 4)
          const color = statusColors[task.status] || statusColors.pending
          const duration = task.started_at ? formatDuration(durationBetween(task.started_at, task.completed_at)) : ''

          return (
            <g key={task.name}>
              {/* Label */}
              <text
                x={labelWidth - 8}
                y={y + 20}
                textAnchor="end"
                className="fill-foreground text-[11px]"
              >
                {task.name.length > 16 ? task.name.slice(0, 16) + '\u2026' : task.name}
              </text>

              {/* Bar */}
              <rect
                x={labelWidth + barStart}
                y={y + 6}
                width={barWidth}
                height={22}
                rx={4}
                fill={color}
                opacity={0.85}
                data-running={task.status === 'running' ? 'true' : undefined}
              >
                <title>{`${task.name} (${task.agent}) \u2014 ${duration || task.status}`}</title>
              </rect>

              {/* Duration label */}
              {duration && barWidth > 40 && (
                <text
                  x={labelWidth + barStart + barWidth / 2}
                  y={y + 21}
                  textAnchor="middle"
                  className="fill-white text-[9px] font-semibold"
                  style={{ paintOrder: 'stroke', stroke: color, strokeWidth: 2 }}
                >
                  {duration}
                </text>
              )}

              {/* Row separator */}
              <line
                x1={labelWidth}
                x2={labelWidth + chartWidth}
                y1={y + rowHeight}
                y2={y + rowHeight}
                stroke="var(--border)"
                strokeWidth={0.5}
                opacity={0.5}
              />
            </g>
          )
        })}

        {/* Running pulse animation */}
        {rows.some(({ task }) => task.status === 'running') && (
          <defs>
            <style>{`
              @keyframes pulse-bar { 0%, 100% { opacity: 0.85; } 50% { opacity: 0.5; } }
              rect[data-running="true"] { animation: pulse-bar 1.5s ease-in-out infinite; }
            `}</style>
          </defs>
        )}
      </svg>
    </div>
  )
}
