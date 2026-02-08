import { useParams } from 'react-router-dom'
import { PipelineComposer } from '@/components/composer/PipelineComposer'
import { useCrew } from '@/hooks/useCrews'

export function ComposerPage() {
  const { name } = useParams<{ name: string }>()
  const { data: crew, isLoading } = useCrew(name || '')

  if (name && isLoading) {
    return <p className="text-muted-foreground">Loading crew...</p>
  }

  return <PipelineComposer initialCrew={crew} />
}
