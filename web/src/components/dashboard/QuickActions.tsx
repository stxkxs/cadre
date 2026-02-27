import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Play, Plus, Workflow } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useCrews } from '@/hooks/useCrews'
import { useStartRun } from '@/hooks/useRuns'
import { toast } from 'sonner'

export function QuickActions() {
  const [showRun, setShowRun] = useState(false)
  const [selectedCrew, setSelectedCrew] = useState('')
  const [inputs, setInputs] = useState('')
  const { data: crews } = useCrews()
  const startRun = useStartRun()

  const handleStartRun = () => {
    if (!selectedCrew) {
      toast.error('Select a crew')
      return
    }
    let parsed: Record<string, unknown> | undefined
    try {
      parsed = inputs.trim() ? JSON.parse(inputs) : undefined
    } catch {
      toast.error('Invalid JSON inputs')
      return
    }
    startRun.mutate(
      { crew: selectedCrew, inputs: parsed },
      {
        onSuccess: (data) => {
          toast.success(`Run started: ${data.id.slice(0, 8)}`)
          setShowRun(false)
          setInputs('')
        },
        onError: (err) => toast.error(`Run failed: ${err.message}`),
      },
    )
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Button variant="accent" size="sm" className="w-full justify-start gap-2" onClick={() => setShowRun(true)}>
            <Play className="h-4 w-4" /> Start Run
          </Button>
          <Link to="/agents">
            <Button variant="ghost" size="sm" className="w-full justify-start gap-2">
              <Plus className="h-4 w-4" /> New Agent
            </Button>
          </Link>
          <Link to="/composer">
            <Button variant="ghost" size="sm" className="w-full justify-start gap-2">
              <Workflow className="h-4 w-4" /> Open Composer
            </Button>
          </Link>
        </CardContent>
      </Card>

      <Dialog open={showRun} onOpenChange={setShowRun}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Start Run</DialogTitle>
            <DialogDescription>Select a crew and optionally provide inputs.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="text-xs font-mono text-muted-foreground mb-1 block">Crew</label>
              <Select value={selectedCrew} onChange={(e) => setSelectedCrew(e.target.value)}>
                <option value="">Select a crew...</option>
                {crews?.map((c) => (
                  <option key={c.name} value={c.name}>{c.name}</option>
                ))}
              </Select>
            </div>
            <div>
              <label className="text-xs font-mono text-muted-foreground mb-1 block">Inputs (JSON, optional)</label>
              <Textarea
                value={inputs}
                onChange={(e) => setInputs(e.target.value)}
                rows={4}
                className="font-mono text-xs"
                placeholder="{}"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setShowRun(false)}>Cancel</Button>
              <Button variant="accent" size="sm" onClick={handleStartRun} disabled={startRun.isPending}>
                {startRun.isPending ? 'Starting...' : 'Start'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
