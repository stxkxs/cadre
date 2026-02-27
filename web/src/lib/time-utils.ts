export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) {
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`
  }
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`
}

export function timeAgo(date: string | Date): string {
  const now = Date.now()
  const then = new Date(date).getTime()
  const diff = now - then

  if (diff < 60_000) return 'just now'
  if (diff < 3_600_000) {
    const m = Math.floor(diff / 60_000)
    return `${m}m ago`
  }
  if (diff < 86_400_000) {
    const h = Math.floor(diff / 3_600_000)
    return `${h}h ago`
  }
  const d = Math.floor(diff / 86_400_000)
  return `${d}d ago`
}

export type TimeRange = '24h' | '7d' | '30d'

export function isWithinRange(date: string | Date, range: TimeRange): boolean {
  const now = Date.now()
  const then = new Date(date).getTime()
  const diff = now - then
  const limits: Record<TimeRange, number> = {
    '24h': 86_400_000,
    '7d': 604_800_000,
    '30d': 2_592_000_000,
  }
  return diff <= limits[range]
}

export function durationBetween(start: string, end?: string): number {
  if (!end) return Date.now() - new Date(start).getTime()
  return new Date(end).getTime() - new Date(start).getTime()
}
