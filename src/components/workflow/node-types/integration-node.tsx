'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Plug } from 'lucide-react';
import { cn } from '@/lib/utils';
import { INTEGRATION_CONFIGS } from '@/types/integration-configs';

interface IntegrationNodeData {
  label: string;
  integrationId?: string;
  integrationAction?: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  isFailed?: boolean;
  [key: string]: unknown;
}

function IntegrationNode({ data, selected }: NodeProps & { data: IntegrationNodeData }) {
  const integrationConfig = INTEGRATION_CONFIGS.find(c => c.id === data.integrationId);
  const color = integrationConfig?.color || '#6366f1';
  const integrationName = integrationConfig?.name || 'Integration';

  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && 'ring-2 ring-indigo-500/50',
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60',
        data.isFailed && 'border-rose-500/60'
      )}
    >
      {/* Top accent bar */}
      <div
        className="h-[3px] w-full rounded-t-lg"
        style={{ backgroundColor: color }}
      />

      <Handle type="target" position={Position.Top} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div
            className="flex items-center justify-center w-6 h-6 rounded-full"
            style={{ backgroundColor: `${color}20` }}
          >
            <Plug className="w-3.5 h-3.5" style={{ color }} />
          </div>
          <p className="text-sm font-medium text-foreground font-display">{data.label || 'Integration'}</p>
        </div>
        <p className="text-xs text-dim mt-1">{integrationName}</p>
        {data.integrationAction && (
          <p className="text-xs text-dim mt-0.5 font-mono">{data.integrationAction}</p>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
    </div>
  );
}

export default memo(IntegrationNode);
