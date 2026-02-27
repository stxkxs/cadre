import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Bot,
  ListTodo,
  Users,
  Workflow,
  Play,
  BookOpen,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/agents', icon: Bot, label: 'Agents' },
  { to: '/tasks', icon: ListTodo, label: 'Tasks' },
  { to: '/crews', icon: Users, label: 'Crews' },
  { to: '/composer', icon: Workflow, label: 'Composer' },
  { to: '/runs', icon: Play, label: 'Runs' },
  { to: '/templates', icon: BookOpen, label: 'Templates' },
]

export function Sidebar() {
  return (
    <aside className="flex h-full w-52 flex-col border-r bg-card/50">
      <div className="flex h-14 items-center gap-2.5 border-b px-5">
        <span className="font-header text-base font-bold tracking-tight">cadre</span>
        <span className="text-[10px] font-mono text-muted-foreground">/</span>
        <span className="text-[10px] font-mono text-[var(--primary-accent)]">ui</span>
      </div>
      <nav className="flex-1 space-y-0.5 px-3 py-3">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors relative',
                isActive
                  ? 'text-foreground bg-accent'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent/50',
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-4 rounded-full bg-[var(--primary-accent)]" />
                )}
                <Icon className="h-4 w-4" />
                {label}
              </>
            )}
          </NavLink>
        ))}
      </nav>
      <div className="border-t px-5 py-3">
        <p className="text-[10px] font-mono text-muted-foreground/60">v0.1.0</p>
      </div>
    </aside>
  )
}
