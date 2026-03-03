'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { GitBranch } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ConditionNodeData {
  label: string;
  condition?: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  isFailed?: boolean;
  [key: string]: unknown;
}

function ConditionNode({ data, selected }: NodeProps & { data: ConditionNodeData }) {
  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && 'node-glow-amber',
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60',
        data.isFailed && 'border-rose-500/60'
      )}
    >
      {/* Top accent bar */}
      <div className="h-[3px] w-full bg-amber-500 rounded-t-lg" />

      <Handle type="target" position={Position.Top} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-center w-6 h-6 rounded-full bg-amber-500/20">
            <GitBranch className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <p className="text-sm font-medium text-foreground font-display truncate">{data.label || 'Condition'}</p>
        </div>

        {data.condition && (
          <div className="mt-2 rounded-md bg-input px-2 py-1">
            <code className="text-xs text-amber-300 font-mono">{data.condition}</code>
          </div>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} id="true" style={{ left: '30%' }} className="!bg-emerald-500 !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
      <Handle type="source" position={Position.Bottom} id="false" style={{ left: '70%' }} className="!bg-rose-500 !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
    </div>
  );
}

export default memo(ConditionNode);
