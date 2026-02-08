import { useState, useCallback, useRef } from 'react'
import { streamChat } from '@/api/sse'
import type { ChatMessage } from '@/types'

export function useAgentChat(agentName: string) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streaming, setStreaming] = useState(false)
  const controllerRef = useRef<AbortController | null>(null)

  const send = useCallback(
    (message: string) => {
      if (streaming || !message.trim()) return

      const userMsg: ChatMessage = { role: 'user', content: message, timestamp: new Date() }
      setMessages((prev) => [...prev, userMsg])
      setStreaming(true)

      let assistantContent = ''
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: '', timestamp: new Date() },
      ])

      controllerRef.current = streamChat(
        agentName,
        message,
        (chunk) => {
          if (chunk.t === 'c' && chunk.c) {
            assistantContent += chunk.c
            setMessages((prev) => {
              const updated = [...prev]
              updated[updated.length - 1] = {
                ...updated[updated.length - 1],
                content: assistantContent,
              }
              return updated
            })
          }
        },
        () => setStreaming(false),
        (error) => {
          assistantContent += `\n\n[Error: ${error}]`
          setMessages((prev) => {
            const updated = [...prev]
            updated[updated.length - 1] = {
              ...updated[updated.length - 1],
              content: assistantContent,
            }
            return updated
          })
          setStreaming(false)
        },
      )
    },
    [agentName, streaming],
  )

  const stop = useCallback(() => {
    controllerRef.current?.abort()
    setStreaming(false)
  }, [])

  const clear = useCallback(() => {
    stop()
    setMessages([])
  }, [stop])

  return { messages, streaming, send, stop, clear }
}
