import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Sun, Moon, Circle } from 'lucide-react'
import { Button } from '@/components/ui/button'

const pageNames: Record<string, string> = {
  '/': 'Dashboard',
  '/agents': 'Agents',
  '/tasks': 'Tasks',
  '/crews': 'Crews',
  '/composer': 'Composer',
  '/runs': 'Runs',
  '/templates': 'Templates',
}

export function Topbar() {
  const [dark, setDark] = useState(true)
  const [healthy, setHealthy] = useState<boolean | null>(null)
  const location = useLocation()

  const pageName = pageNames[location.pathname] || location.pathname.split('/').filter(Boolean).pop() || ''

  useEffect(() => {
    fetch('/api/health')
      .then((r) => r.ok && setHealthy(true))
      .catch(() => setHealthy(false))
    const interval = setInterval(() => {
      fetch('/api/health')
        .then((r) => setHealthy(r.ok))
        .catch(() => setHealthy(false))
    }, 30000)
    return () => clearInterval(interval)
  }, [])

  const toggleTheme = () => {
    setDark((d) => {
      const next = !d
      document.documentElement.classList.toggle('dark', next)
      return next
    })
  }

  return (
    <header className="flex h-12 items-center justify-between border-b px-6">
      <div className="flex items-center gap-2">
        <span className="text-xs font-mono text-muted-foreground">~</span>
        <span className="text-xs font-mono text-muted-foreground">/</span>
        <span className="text-xs font-mono text-foreground">{pageName.toLowerCase()}</span>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1.5 rounded-full border px-2.5 py-1">
          <Circle
            className={`h-1.5 w-1.5 fill-current ${
              healthy === true
                ? 'text-[var(--accent-green)]'
                : healthy === false
                  ? 'text-destructive'
                  : 'text-muted-foreground animate-pulse'
            }`}
          />
          <span className="text-[10px] font-mono text-muted-foreground">
            {healthy === true ? 'live' : healthy === false ? 'offline' : '...'}
          </span>
        </div>
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={toggleTheme}>
          {dark ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
        </Button>
      </div>
    </header>
  )
}
