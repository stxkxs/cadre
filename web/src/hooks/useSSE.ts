import { useEffect, useRef, useState } from 'react'
import { connectSSE } from '@/api/sse'
import type { SSEEvent } from '@/types'

export function useSSE(path: string, enabled = true) {
  const [events, setEvents] = useState<SSEEvent[]>([])
  const [connected, setConnected] = useState(false)
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!enabled) return

    const es = connectSSE(
      path,
      (event) => {
        if (event.type === 'connected') {
          setConnected(true)
          return
        }
        setEvents((prev) => [...prev.slice(-200), event])
      },
      () => setConnected(false),
    )
    esRef.current = es

    return () => {
      es.close()
      esRef.current = null
      setConnected(false)
    }
  }, [path, enabled])

  return { events, connected, clear: () => setEvents([]) }
}
