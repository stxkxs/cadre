import { useState } from 'react'
import { Rocket, ArrowRight, Sparkles } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { StepIndicator } from './StepIndicator'
import { useTemplateCrews } from '@/hooks/useTemplates'
import { useImportTemplate } from '@/hooks/useTemplates'
import { useStartRun } from '@/hooks/useRuns'
import { toast } from 'sonner'

interface GettingStartedProps {
  onDismiss: () => void
}

export function GettingStarted({ onDismiss }: GettingStartedProps) {
  const [step, setStep] = useState(0)
  const [selectedCrew, setSelectedCrew] = useState<string | null>(null)
  const { data: crews } = useTemplateCrews()
  const importTemplate = useImportTemplate()
  const startRun = useStartRun()

  const handleImportAndNext = () => {
    if (!selectedCrew) return
    importTemplate.mutate(
      { type: 'crew', name: selectedCrew },
      {
        onSuccess: () => {
          toast.success(`Imported crew: ${selectedCrew}`)
          setStep(2)
        },
        onError: (err) => toast.error(`Import failed: ${err.message}`),
      },
    )
  }

  const handleRun = () => {
    if (!selectedCrew) return
    startRun.mutate(
      { crew: selectedCrew },
      {
        onSuccess: (data) => {
          toast.success(`Run started: ${data.id.slice(0, 8)}`)
          onDismiss()
        },
        onError: (err) => toast.error(`Run failed: ${err.message}`),
      },
    )
  }

  return (
    <div className="space-y-8">
      <div className="text-center space-y-3">
        <div className="h-14 w-14 rounded-2xl bg-[var(--primary-accent)]/10 flex items-center justify-center mx-auto">
          <Rocket className="h-7 w-7 text-[var(--primary-accent)]" />
        </div>
        <div>
          <h2 className="text-lg font-bold font-header tracking-tight">Welcome to cadre</h2>
          <p className="text-xs text-muted-foreground mt-1">Get started by importing a crew template</p>
        </div>
        <StepIndicator steps={3} current={step} />
      </div>

      {step === 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-semibold text-center">Pick a Crew</h3>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {(crews || []).slice(0, 4).map((crew) => (
              <Card
                key={crew.name}
                className={`cursor-pointer transition-all ${
                  selectedCrew === crew.name
                    ? 'border-[var(--primary-accent)] ring-1 ring-[var(--primary-accent)]'
                    : 'hover:border-[var(--primary-accent)]/30'
                }`}
                onClick={() => setSelectedCrew(crew.name)}
              >
                <CardContent className="p-4 space-y-2">
                  <div className="flex items-center gap-2">
                    <Sparkles className="h-4 w-4 text-[var(--primary-accent)]" />
                    <span className="text-sm font-semibold">{crew.name}</span>
                  </div>
                  <p className="text-[11px] text-muted-foreground line-clamp-2">{crew.description}</p>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-[10px]">{crew.process}</Badge>
                    <span className="text-[10px] text-muted-foreground">{crew.agents?.length || 0} agents</span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
          <div className="flex items-center justify-between">
            <Link to="/templates" className="text-xs text-muted-foreground hover:text-foreground transition-colors">
              Browse all templates <ArrowRight className="inline h-3 w-3" />
            </Link>
            <Button variant="accent" size="sm" onClick={() => setStep(1)} disabled={!selectedCrew}>
              Next <ArrowRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      )}

      {step === 1 && (
        <div className="space-y-4">
          <h3 className="text-sm font-semibold text-center">Review & Import</h3>
          {selectedCrew && (
            <Card>
              <CardContent className="p-4 space-y-3">
                <p className="text-sm">
                  Importing <span className="font-semibold font-mono">{selectedCrew}</span> will add the crew along with its agents and tasks to your project.
                </p>
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  {crews?.find((c) => c.name === selectedCrew)?.agents?.map((a) => (
                    <Badge key={a} variant="outline" className="text-[10px]">{a}</Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
          <div className="flex items-center justify-between">
            <Button variant="ghost" size="sm" onClick={() => setStep(0)}>Back</Button>
            <Button variant="accent" size="sm" onClick={handleImportAndNext} disabled={importTemplate.isPending}>
              {importTemplate.isPending ? 'Importing...' : 'Import'}
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="space-y-4">
          <h3 className="text-sm font-semibold text-center">Configure & Run</h3>
          <Card>
            <CardContent className="p-4 space-y-3">
              <p className="text-sm">
                <span className="font-semibold font-mono">{selectedCrew}</span> has been imported. You can run it now or configure it first in the composer.
              </p>
            </CardContent>
          </Card>
          <div className="flex items-center justify-center gap-2">
            <Link to={`/composer/${selectedCrew}`}>
              <Button variant="ghost" size="sm">Open in Composer</Button>
            </Link>
            <Button variant="accent" size="sm" onClick={handleRun} disabled={startRun.isPending}>
              {startRun.isPending ? 'Starting...' : 'Run Now'}
            </Button>
          </div>
        </div>
      )}

      <div className="text-center">
        <button onClick={onDismiss} className="text-[11px] text-muted-foreground hover:text-foreground transition-colors">
          Skip setup
        </button>
      </div>
    </div>
  )
}
