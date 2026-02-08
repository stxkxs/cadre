import { useState, useRef, useEffect } from 'react'
import { Send, Square, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { useAgentChat } from '@/hooks/useAgentChat'
import { cn } from '@/lib/utils'

export function ChatPanel({ agentName }: { agentName: string }) {
  const { messages, streaming, send, stop, clear } = useAgentChat(agentName)
  const [input, setInput] = useState('')
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages])

  const handleSend = () => {
    if (input.trim()) {
      send(input)
      setInput('')
    }
  }

  return (
    <Card className="flex flex-col h-[calc(100vh-12rem)] overflow-hidden">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold">{agentName}</h2>
          <p className="text-[10px] font-mono text-muted-foreground">chat session</p>
        </div>
        <Button variant="ghost" size="sm" onClick={clear}>
          <RotateCcw className="h-3 w-3 mr-1" /> Clear
        </Button>
      </div>

      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && (
          <p className="text-xs text-muted-foreground text-center py-12">
            Send a message to start chatting with {agentName}
          </p>
        )}
        {messages.map((msg, i) => (
          <div
            key={i}
            className={cn(
              'max-w-[80%] rounded-xl px-4 py-3',
              msg.role === 'user'
                ? 'ml-auto bg-[var(--primary-accent)] text-white'
                : 'bg-muted border-l-2 border-[var(--primary-accent)]/30',
            )}
          >
            <div className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">
              {msg.content}
              {streaming && i === messages.length - 1 && msg.role === 'assistant' && (
                <span className="inline-flex gap-0.5 ml-1">
                  <span className="h-1 w-1 rounded-full bg-foreground/50 animate-pulse" style={{ animationDelay: '0ms' }} />
                  <span className="h-1 w-1 rounded-full bg-foreground/50 animate-pulse" style={{ animationDelay: '150ms' }} />
                  <span className="h-1 w-1 rounded-full bg-foreground/50 animate-pulse" style={{ animationDelay: '300ms' }} />
                </span>
              )}
            </div>
          </div>
        ))}
      </div>

      <div className="border-t p-3">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), handleSend())}
            placeholder="Type a message..."
            disabled={streaming}
            rows={1}
            className="flex-1 resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm font-mono shadow-sm placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:border-[var(--primary-accent)] focus-visible:shadow-[0_0_0_3px_var(--primary-accent)]/10 disabled:cursor-not-allowed disabled:opacity-50"
          />
          {streaming ? (
            <Button variant="destructive" size="icon" onClick={stop}>
              <Square className="h-4 w-4" />
            </Button>
          ) : (
            <Button variant="accent" size="icon" onClick={handleSend} disabled={!input.trim()}>
              <Send className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}
