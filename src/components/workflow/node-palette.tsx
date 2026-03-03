'use client';

import React, { useState } from 'react';
import { Brain, GitBranch, ArrowDownToLine, ArrowUpFromLine, Layers, RotateCcw, Plug, PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useWorkflowStore } from '@/lib/store/workflow-store';

const paletteItems = [
  { type: 'agent', label: 'Agent', icon: Brain, description: 'AI model execution', accentColor: 'border-l-orange-500', textColor: 'text-orange-400', borderColor: 'border-orange-500/20' },
  { type: 'condition', label: 'Condition', icon: GitBranch, description: 'Branch logic', accentColor: 'border-l-amber-500', textColor: 'text-amber-400', borderColor: 'border-amber-500/20' },
  { type: 'input', label: 'Input', icon: ArrowDownToLine, description: 'Workflow input', accentColor: 'border-l-cyan-500', textColor: 'text-cyan-400', borderColor: 'border-cyan-500/20' },
  { type: 'output', label: 'Output', icon: ArrowUpFromLine, description: 'Workflow output', accentColor: 'border-l-[#0AEFB7]', textColor: 'text-[#0AEFB7]', borderColor: 'border-[#0AEFB7]/20' },
  { type: 'parallel', label: 'Parallel', icon: Layers, description: 'Concurrent execution', accentColor: 'border-l-indigo-500', textColor: 'text-indigo-400', borderColor: 'border-indigo-500/20' },
  { type: 'loop', label: 'Loop', icon: RotateCcw, description: 'Repeat until condition', accentColor: 'border-l-pink-500', textColor: 'text-pink-400', borderColor: 'border-pink-500/20' },
  { type: 'integration', label: 'Integration', icon: Plug, description: 'External service action', accentColor: 'border-l-violet-500', textColor: 'text-violet-400', borderColor: 'border-violet-500/20' },
];

export function NodePalette() {
  const nodes = useWorkflowStore((s) => s.nodes);
  const [collapsed, setCollapsed] = useState(false);

  const onDragStart = (event: React.DragEvent, nodeType: string, label: string) => {
    event.dataTransfer.setData('application/reactflow-type', nodeType);
    event.dataTransfer.setData('application/reactflow-label', label);
    event.dataTransfer.effectAllowed = 'move';
  };

  if (collapsed) {
    return (
      <div className="w-10 border-r border-border bg-card flex flex-col items-center pt-3">
        <button
          onClick={() => setCollapsed(false)}
          className="text-dim hover:text-foreground transition-colors cursor-pointer"
          title="Show node palette"
        >
          <PanelLeftOpen className="h-4 w-4" />
        </button>
      </div>
    );
  }

  const typeCounts = new Map<string, number>();
  for (const n of nodes) {
    typeCounts.set(n.type, (typeCounts.get(n.type) || 0) + 1);
  }

  return (
    <div className="w-56 border-r border-border bg-card p-3 flex flex-col gap-1.5">
      <div className="flex items-center justify-between px-2 mb-2">
        <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-dim font-display">
          Nodes
        </h3>
        <button
          onClick={() => setCollapsed(true)}
          className="text-dim hover:text-foreground transition-colors cursor-pointer"
          title="Collapse palette"
        >
          <PanelLeftClose className="h-3.5 w-3.5" />
        </button>
      </div>
      {paletteItems.map((node) => {
        const count = typeCounts.get(node.type) || 0;
        return (
          <div
            key={node.type}
            draggable
            onDragStart={(e) => onDragStart(e, node.type, node.label)}
            className={cn(
              'flex items-center gap-3 rounded-lg border border-l-[3px] bg-card px-3 py-2.5 cursor-grab active:cursor-grabbing transition-all hover:bg-hover',
              node.accentColor,
              node.borderColor
            )}
          >
            <node.icon className={cn('w-4 h-4 shrink-0', node.textColor)} />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">{node.label}</p>
              <p className="text-xs text-dim truncate">{node.description}</p>
            </div>
            {count > 0 && (
              <span className="text-[10px] font-medium bg-input text-muted-foreground rounded-md w-5 h-5 flex items-center justify-center shrink-0">
                {count}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
