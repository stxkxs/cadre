'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { ArrowDownToLine } from 'lucide-react';
import { cn } from '@/lib/utils';

interface InputNodeData {
  label: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  [key: string]: unknown;
}

function InputNode({ data, selected }: NodeProps & { data: InputNodeData }) {
  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && 'node-glow-cyan',
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60'
      )}
    >
      {/* Top accent bar */}
      <div className="h-[3px] w-full bg-cyan-500 rounded-t-lg" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-center w-6 h-6 rounded-full bg-cyan-500/20">
            <ArrowDownToLine className="w-3.5 h-3.5 text-cyan-400" />
          </div>
          <p className="text-sm font-medium text-foreground font-display">{data.label || 'Input'}</p>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-cyan-500 !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />
    </div>
  );
}

export default memo(InputNode);
