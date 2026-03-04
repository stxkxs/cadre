'use client';

import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { ArrowUpFromLine } from 'lucide-react';
import { cn } from '@/lib/utils';

interface OutputNodeData {
  label: string;
  isRunning?: boolean;
  isCompleted?: boolean;
  [key: string]: unknown;
}

function OutputNode({ data, selected }: NodeProps & { data: OutputNodeData }) {
  return (
    <div
      className={cn(
        'relative rounded-lg border border-border bg-card min-w-[220px] shadow-lg transition-all',
        selected && 'node-glow-teal',
        data.isRunning && 'node-scan',
        data.isCompleted && 'border-emerald-500/60'
      )}
    >
      {/* Top accent bar */}
      <div className="h-[3px] w-full bg-[#0AEFB7] rounded-t-lg" />

      <Handle type="target" position={Position.Top} className="!bg-[#0AEFB7] !w-3.5 !h-3.5 !border-2 !border-card !ring-2 !ring-border" />

      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-center w-6 h-6 rounded-full bg-[#0AEFB7]/20">
            <ArrowUpFromLine className="w-3.5 h-3.5 text-[#0AEFB7]" />
          </div>
          <p className="text-sm font-medium text-foreground font-display">{data.label || 'Output'}</p>
        </div>
      </div>
    </div>
  );
}

export default memo(OutputNode);
