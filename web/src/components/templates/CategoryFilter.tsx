import { cn } from '@/lib/utils'

interface CategoryFilterProps {
  categories: { id: string; label: string }[]
  active: string
  onChange: (id: string) => void
}

export function CategoryFilter({ categories, active, onChange }: CategoryFilterProps) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <button
        onClick={() => onChange('')}
        className={cn(
          'rounded-full px-3 py-1 text-xs font-medium transition-colors',
          active === ''
            ? 'bg-[var(--primary-accent)] text-white'
            : 'bg-accent text-muted-foreground hover:text-foreground',
        )}
      >
        All
      </button>
      {categories.map((cat) => (
        <button
          key={cat.id}
          onClick={() => onChange(cat.id)}
          className={cn(
            'rounded-full px-3 py-1 text-xs font-medium transition-colors',
            active === cat.id
              ? 'bg-[var(--primary-accent)] text-white'
              : 'bg-accent text-muted-foreground hover:text-foreground',
          )}
        >
          {cat.label}
        </button>
      ))}
    </div>
  )
}
