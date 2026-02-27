interface DonutChartProps {
  segments: { label: string; value: number; color: string }[]
  size?: number
  strokeWidth?: number
}

export function DonutChart({ segments, size = 160, strokeWidth = 24 }: DonutChartProps) {
  const total = segments.reduce((sum, s) => sum + s.value, 0)
  if (total === 0) {
    return (
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={(size - strokeWidth) / 2}
          fill="none"
          stroke="var(--muted)"
          strokeWidth={strokeWidth}
        />
        <text
          x={size / 2}
          y={size / 2}
          textAnchor="middle"
          dominantBaseline="central"
          className="fill-muted-foreground text-sm font-mono"
        >
          0
        </text>
      </svg>
    )
  }

  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  let offset = 0

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      {segments.filter((s) => s.value > 0).map((segment, i) => {
        const segLength = (segment.value / total) * circumference
        const currentOffset = offset
        offset += segLength
        return (
          <circle
            key={i}
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={segment.color}
            strokeWidth={strokeWidth}
            strokeDasharray={`${segLength} ${circumference - segLength}`}
            strokeDashoffset={-currentOffset}
            strokeLinecap="round"
            transform={`rotate(-90 ${size / 2} ${size / 2})`}
          >
            <title>{`${segment.label}: ${segment.value}`}</title>
          </circle>
        )
      })}
      <text
        x={size / 2}
        y={size / 2}
        textAnchor="middle"
        dominantBaseline="central"
        className="fill-foreground text-2xl font-bold font-mono"
      >
        {total}
      </text>
    </svg>
  )
}
