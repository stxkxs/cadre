'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Brain } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ModelProvider } from '@/lib/engine/types';

const providerStyles: Record<ModelProvider, { barColor: string; iconBg: string; iconText: string; glowClass: string }> = {
  anthropic: { barColor: 'bg-orange-500', iconBg: 'bg-orange-500/20', iconText: 'text-orange-400', glowClass: 'node-glow-orange' },
  openai: { barColor: 'bg-green-500', iconBg: 'bg-green-500/20', iconText: 'text-green-400', glowClass: 'node-glow-green' },
  groq: { barColor: 'bg-purple-500', iconBg: 'bg-purple-500/20', iconText: 'text-purple-400', glowClass: 'node-glow-purple' },
  'claude-code': { barColor: 'bg-blue-500', iconBg: 'bg-blue-500/20', iconText: 'text-blue-400', glowClass: 'node-glow-blue' },
  bedrock: { barColor: 'bg-amber-500', iconBg: 'bg-amber-500/20', iconText: 'text-amber-400', glowClass: 'node-glow-amber' },
};

interface AgentNodeData {
  label: string;
  provider?: ModelProvider;
  model?: string;
  systemPrompt?: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  isFailed?: boolean;
  [key: string]: unknown;
}

function AgentNode({ data, selected }: NodeProps & { data: AgentNodeData }) {
  const provider = data.provider || 'anthropic';
  const style = providerStyles[provider];

  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && style.glowClass,
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60',
        data.isFailed && 'border-rose-500/60'
      )}
    >
      {/* Top accent bar */}
      <div className={cn('h-[3px] w-full rounded-t-lg', style.barColor)} />

      <Handle type="target" position={Position.Top} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className={cn('flex items-center justify-center w-6 h-6 rounded-full', style.iconBg)}>
            <Brain className={cn('w-3.5 h-3.5', style.iconText)} />
          </div>
          <p className="text-sm font-medium text-foreground font-display">{data.label || 'Agent'}</p>
        </div>
        <p className="text-xs text-dim mt-1">{data.model || 'No model selected'}</p>

        {data.systemPrompt && (
          <p className="text-xs text-dim mt-1 line-clamp-2">{data.systemPrompt}</p>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
    </div>
  );
}

export default memo(AgentNode);
