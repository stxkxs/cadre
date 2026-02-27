import { useEffect } from 'react'

interface Shortcut {
  key: string
  meta?: boolean
  shift?: boolean
  action: () => void
  when?: () => boolean
  allowInInput?: boolean
}

export function useKeyboardShortcuts(shortcuts: Shortcut[]) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable

      for (const shortcut of shortcuts) {
        const metaMatch = shortcut.meta ? (e.metaKey || e.ctrlKey) : !(e.metaKey || e.ctrlKey)
        const shiftMatch = shortcut.shift ? e.shiftKey : !e.shiftKey
        const keyMatch = e.key.toLowerCase() === shortcut.key.toLowerCase()

        if (!metaMatch || !shiftMatch || !keyMatch) continue
        if (isInput && !shortcut.allowInInput) continue
        if (shortcut.when && !shortcut.when()) continue

        e.preventDefault()
        shortcut.action()
        return
      }
    }

    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [shortcuts])
}
