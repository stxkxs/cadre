import type { SSEEvent } from '@/types'

export function connectSSE(
  path: string,
  onEvent: (event: SSEEvent) => void,
  onError?: (error: Event) => void,
): EventSource {
  const es = new EventSource(`/api${path}`)

  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data) as SSEEvent
      onEvent(data)
    } catch {
      // ignore parse errors
    }
  }

  es.onerror = (e) => {
    onError?.(e)
  }

  return es
}

export interface StreamChunk {
  t: 'c' | 'done' | 'error'
  c?: string
  error?: string
}

export function streamChat(
  agentName: string,
  message: string,
  onChunk: (chunk: StreamChunk) => void,
  onDone: () => void,
  onError?: (error: string) => void,
): AbortController {
  const controller = new AbortController()

  fetch(`/api/agents/${agentName}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
    signal: controller.signal,
  })
    .then(async (res) => {
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        onError?.(body.error || res.statusText)
        return
      }

      const reader = res.body?.getReader()
      if (!reader) return

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const chunk = JSON.parse(line.slice(6)) as StreamChunk
              onChunk(chunk)
              if (chunk.t === 'done') onDone()
              if (chunk.t === 'error') onError?.(chunk.error || 'Unknown error')
            } catch {
              // ignore
            }
          }
        }
      }
    })
    .catch((err) => {
      if (err.name !== 'AbortError') {
        onError?.(err.message)
      }
    })

  return controller
}
