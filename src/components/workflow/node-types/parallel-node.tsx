'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Layers } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ParallelNodeData {
  label: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  [key: string]: unknown;
}

function ParallelNode({ data, selected }: NodeProps & { data: ParallelNodeData }) {
  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && 'node-glow-indigo',
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60'
      )}
    >
      {/* Top accent bar */}
      <div className="h-[3px] w-full bg-indigo-500 rounded-t-lg" />

      <Handle type="target" position={Position.Top} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-center w-6 h-6 rounded-full bg-indigo-500/20">
            <Layers className="w-3.5 h-3.5 text-indigo-400" />
          </div>
          <div>
            <p className="text-sm font-medium text-foreground font-display">{data.label || 'Parallel'}</p>
            <p className="text-xs text-dim">Execute branches concurrently</p>
          </div>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-handle !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
    </div>
  );
}

export default memo(ParallelNode);
