export function computePercentages(values: number[]): number[] {
  const total = values.reduce((sum, v) => sum + v, 0)
  if (total === 0) return values.map(() => 0)
  return values.map((v) => (v / total) * 100)
}

const chartColors = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
]

export function getChartColor(index: number): string {
  return chartColors[index % chartColors.length]
}
