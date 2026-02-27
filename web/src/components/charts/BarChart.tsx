interface BarChartProps {
  items: { label: string; value: number; color?: string }[]
  maxValue?: number
}

export function BarChart({ items, maxValue }: BarChartProps) {
  const max = maxValue || Math.max(...items.map((i) => i.value), 1)

  return (
    <div className="space-y-2">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-3">
          <span className="text-[11px] text-muted-foreground w-24 truncate text-right font-mono">
            {item.label}
          </span>
          <div className="flex-1 h-5 bg-muted/50 rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-500"
              style={{
                width: `${Math.max((item.value / max) * 100, 2)}%`,
                backgroundColor: item.color || 'var(--primary-accent)',
              }}
            />
          </div>
          <span className="text-[11px] font-mono text-foreground w-12 text-right">
            {item.value < 1000 ? item.value.toFixed(0) : `${(item.value / 1000).toFixed(1)}k`}
          </span>
        </div>
      ))}
    </div>
  )
}
