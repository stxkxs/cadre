import { Save, CheckCircle, Play, Settings2, Undo2, Redo2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogDescription,
} from '@/components/ui/dialog'
import { useState } from 'react'

interface ComposerToolbarProps {
  crewName: string
  crewDescription: string
  process: string
  manager: string
  errorStrategy: string
  concurrency: number
  maxIterations: number
  onSave: () => void
  onValidate: () => void
  onRun: () => void
  onMetaChange: (meta: {
    name: string
    description: string
    process: string
    manager: string
    errorStrategy: string
    concurrency: number
    maxIterations: number
  }) => void
  saving: boolean
  validating: boolean
  canUndo?: boolean
  canRedo?: boolean
  onUndo?: () => void
  onRedo?: () => void
  isDirty?: boolean
}

export function ComposerToolbar({
  crewName,
  crewDescription,
  process,
  manager,
  errorStrategy,
  concurrency,
  maxIterations,
  onSave,
  onValidate,
  onRun,
  onMetaChange,
  saving,
  validating,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  isDirty,
}: ComposerToolbarProps) {
  const [open, setOpen] = useState(false)

  return (
    <div className="flex items-center gap-2 border-b bg-background/50 px-4 py-1.5">
      <Input
        value={crewName}
        onChange={(e) => onMetaChange({ name: e.target.value, description: crewDescription, process, manager, errorStrategy, concurrency, maxIterations })}
        placeholder="Crew name"
        className="h-8 w-48 text-sm font-mono"
      />
      {isDirty && <span className="text-[var(--accent-orange)] font-bold text-sm">*</span>}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="ghost" size="sm">
            <Settings2 className="h-4 w-4 mr-1" /> Settings
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Crew Settings</DialogTitle>
            <DialogDescription>Configure the crew process type and execution behavior.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">Description</label>
              <Input
                value={crewDescription}
                onChange={(e) => onMetaChange({ name: crewName, description: e.target.value, process, manager, errorStrategy, concurrency, maxIterations })}
              />
            </div>
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">Process</label>
              <Select
                value={process}
                onChange={(e) => onMetaChange({ name: crewName, description: crewDescription, process: e.target.value, manager, errorStrategy, concurrency, maxIterations })}
              >
                <option value="sequential">Sequential</option>
                <option value="parallel">Parallel</option>
                <option value="hierarchical">Hierarchical</option>
              </Select>
            </div>
            {process === 'hierarchical' && (
              <div>
                <label className="text-xs text-muted-foreground mb-1 block">Manager Agent</label>
                <Input
                  value={manager}
                  onChange={(e) => onMetaChange({ name: crewName, description: crewDescription, process, manager: e.target.value, errorStrategy, concurrency, maxIterations })}
                />
              </div>
            )}
            {process === 'parallel' && (
              <>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Error Strategy</label>
                  <Select
                    value={errorStrategy}
                    onChange={(e) => onMetaChange({ name: crewName, description: crewDescription, process, manager, errorStrategy: e.target.value, concurrency, maxIterations })}
                  >
                    <option value="fail-fast">Fail Fast</option>
                    <option value="complete-running">Complete Running</option>
                    <option value="continue-all">Continue All</option>
                  </Select>
                </div>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Concurrency (0 = auto)</label>
                  <Input
                    type="number"
                    value={concurrency}
                    onChange={(e) => onMetaChange({ name: crewName, description: crewDescription, process, manager, errorStrategy, concurrency: parseInt(e.target.value) || 0, maxIterations })}
                  />
                </div>
              </>
            )}
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">Max Iterations</label>
              <Input
                type="number"
                min={0}
                max={100}
                value={maxIterations}
                onChange={(e) => onMetaChange({ name: crewName, description: crewDescription, process, manager, errorStrategy, concurrency, maxIterations: parseInt(e.target.value) || 0 })}
              />
              <p className="text-xs text-muted-foreground mt-1">0 = single pass, &gt;0 for loop workflows</p>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Undo/Redo */}
      {onUndo && (
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onUndo} disabled={!canUndo} title="Undo (Cmd+Z)">
          <Undo2 className="h-4 w-4" />
        </Button>
      )}
      {onRedo && (
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onRedo} disabled={!canRedo} title="Redo (Cmd+Shift+Z)">
          <Redo2 className="h-4 w-4" />
        </Button>
      )}

      <div className="flex-1" />

      <Button variant="ghost" size="sm" onClick={onValidate} disabled={validating}>
        <CheckCircle className="h-4 w-4 mr-1" />
        {validating ? 'Validating...' : 'Validate'}
      </Button>

      <Button variant="default" size="sm" onClick={onSave} disabled={saving}>
        <Save className="h-4 w-4 mr-1" />
        {saving ? 'Saving...' : 'Save'}
      </Button>

      <Button variant="accent" size="sm" onClick={onRun}>
        <Play className="h-4 w-4 mr-1" /> Run
      </Button>
    </div>
  )
}
