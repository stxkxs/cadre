'use client';

import React from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Separator } from '@/components/ui/separator';
import { useWorkflowStore } from '@/lib/store/workflow-store';
import { PROVIDER_CONFIGS } from '@/types/provider';
import type { ModelProvider } from '@/lib/engine/types';

export function ConfigPanel() {
  const { selectedNodeId, nodes, edges, updateNode, selectNode, removeNode } = useWorkflowStore();

  const node = nodes.find((n) => n.id === selectedNodeId);
  if (!node) return null;

  const selectedProvider = PROVIDER_CONFIGS.find((p) => p.id === node.data.provider);
  const models = selectedProvider?.models || [];
  const incomingEdges = edges.filter((e) => e.target === node.id).length;
  const outgoingEdges = edges.filter((e) => e.source === node.id).length;

  return (
    <div className="w-80 border-l border-border bg-card flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <h3 className="text-sm font-semibold text-foreground font-display">Node Configuration</h3>
          <p className="text-xs text-dim font-mono mt-0.5">{node.id}</p>
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => selectNode(null)}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Config form */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Node info */}
        <div className="flex items-center gap-4 text-xs text-dim">
          <span className="capitalize">{node.type}</span>
          <span>{incomingEdges} in / {outgoingEdges} out</span>
        </div>

        {/* Label */}
        <div className="space-y-2">
          <Label>Label <span className="text-rose-400">*</span></Label>
          <Input
            value={node.data.label || ''}
            onChange={(e) => updateNode(node.id, { label: e.target.value.slice(0, 100) })}
            placeholder="Node label"
          />
          {!node.data.label?.trim() && (
            <p className="text-xs text-rose-400">Label is required</p>
          )}
        </div>

        {/* Agent-specific config */}
        {node.type === 'agent' && (
          <>
            <Separator />

            {/* Provider */}
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select
                value={node.data.provider || ''}
                onValueChange={(value) => updateNode(node.id, { provider: value as ModelProvider, model: '' })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent>
                  {PROVIDER_CONFIGS.map((p) => (
                    <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Model */}
            {node.data.provider && (
              <div className="space-y-2">
                <Label>Model</Label>
                <Select
                  value={node.data.model || ''}
                  onValueChange={(value) => updateNode(node.id, { model: value })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select model" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.map((m) => (
                      <SelectItem key={m.id} value={m.id}>{m.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* System Prompt */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>System Prompt</Label>
                <span className={`text-xs ${(node.data.systemPrompt || '').length > 10000 ? 'text-rose-400' : 'text-dim'}`}>
                  {(node.data.systemPrompt || '').length}/10000
                </span>
              </div>
              <Textarea
                value={node.data.systemPrompt || ''}
                onChange={(e) => {
                  if (e.target.value.length <= 10000) {
                    updateNode(node.id, { systemPrompt: e.target.value });
                  }
                }}
                placeholder="You are a helpful assistant..."
                rows={4}
              />
            </div>

            {/* Temperature */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Temperature</Label>
                <span className="text-xs text-dim">{node.data.temperature ?? 0.7}</span>
              </div>
              <Slider
                min={0}
                max={2}
                step={0.1}
                value={node.data.temperature ?? 0.7}
                onChange={(e) => updateNode(node.id, { temperature: parseFloat((e.target as HTMLInputElement).value) })}
              />
            </div>

            {/* Max Tokens */}
            <div className="space-y-2">
              <Label>Max Tokens</Label>
              <Input
                type="number"
                min={1}
                max={200000}
                value={node.data.maxTokens || 4096}
                onChange={(e) => {
                  const val = parseInt(e.target.value) || 4096;
                  updateNode(node.id, { maxTokens: Math.min(200000, Math.max(1, val)) });
                }}
              />
              <p className="text-xs text-dim">1 – 200,000</p>
            </div>

            {/* Retries */}
            <div className="space-y-2">
              <Label>Retries on Error</Label>
              <Input
                type="number"
                min={0}
                max={5}
                value={node.data.retries || 0}
                onChange={(e) => updateNode(node.id, { retries: parseInt(e.target.value) || 0 })}
              />
            </div>

            {/* Timeout */}
            <div className="space-y-2">
              <Label>Timeout (seconds)</Label>
              <Input
                type="number"
                min={5}
                max={node.data.provider === 'claude-code' ? 3600 : 600}
                value={node.data.timeout || (node.data.provider === 'claude-code' ? 600 : 120)}
                onChange={(e) => {
                  const max = node.data.provider === 'claude-code' ? 3600 : 600;
                  const defaultVal = node.data.provider === 'claude-code' ? 600 : 120;
                  updateNode(node.id, { timeout: Math.min(max, Math.max(5, parseInt(e.target.value) || defaultVal)) });
                }}
              />
              <p className="text-xs text-dim">
                {node.data.provider === 'claude-code'
                  ? 'Max execution time (5-3600s). Claude Code tasks can run up to 1 hour.'
                  : 'Max execution time per node (5-600s)'}
              </p>
            </div>

            {/* Max Turns (Claude Code only) */}
            {node.data.provider === 'claude-code' && (
              <div className="space-y-2">
                <Label>Max Turns</Label>
                <Input
                  type="number"
                  min={1}
                  max={50}
                  value={node.data.maxTurns || 10}
                  onChange={(e) => updateNode(node.id, { maxTurns: Math.min(50, Math.max(1, parseInt(e.target.value) || 10)) })}
                />
                <p className="text-xs text-dim">
                  Limits agentic turns (tool calls). Default 10. Higher = more thorough but slower.
                </p>
              </div>
            )}

            {/* Workspace */}
            <div className="space-y-2">
              <Label>Workspace</Label>
              <Select
                value={node.data.workspace || 'off'}
                onValueChange={(value) => updateNode(node.id, { workspace: value as 'off' | 'safe' | 'full' })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select workspace mode" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="off">Off</SelectItem>
                  <SelectItem value="safe">Safe</SelectItem>
                  {node.data.provider === 'claude-code' && (
                    <SelectItem value="full">Full Autonomy</SelectItem>
                  )}
                </SelectContent>
              </Select>
              <p className="text-xs text-dim">
                {node.data.workspace === 'safe'
                  ? 'Node runs in workspace directory. Output saved to file.'
                  : node.data.workspace === 'full'
                  ? 'Full autonomy — skips permission prompts (Claude Code only).'
                  : 'No file I/O. Text output only.'}
              </p>
            </div>
          </>
        )}

        {/* Loop-specific config */}
        {node.type === 'loop' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Loop Condition</Label>
              <Textarea
                value={node.data.condition || ''}
                onChange={(e) => updateNode(node.id, { condition: e.target.value })}
                placeholder="context.iteration < 5"
                rows={3}
                className="font-mono text-xs"
              />
              {node.data.condition && (() => {
                try { new Function('context', `return (${node.data.condition})`); return null; }
                catch { return <p className="text-xs text-rose-400">Invalid JavaScript expression</p>; }
              })()}
              <p className="text-xs text-dim">
                Loops while this expression is true. Max 10 iterations.
              </p>
            </div>
          </>
        )}

        {/* Input node config */}
        {node.type === 'input' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Input Variable Name</Label>
              <Input
                value={(node.data as Record<string, unknown>).variableName as string || ''}
                onChange={(e) => updateNode(node.id, { variableName: e.target.value } as Record<string, unknown>)}
                placeholder="userQuery"
                className="font-mono text-xs"
              />
              <p className="text-xs text-dim">The variable name used to reference this input in the workflow context.</p>
            </div>
            <div className="space-y-2">
              <Label>Default Value</Label>
              <Textarea
                value={(node.data as Record<string, unknown>).defaultValue as string || ''}
                onChange={(e) => updateNode(node.id, { defaultValue: e.target.value } as Record<string, unknown>)}
                placeholder="Default input value..."
                rows={3}
              />
            </div>
          </>
        )}

        {/* Output node config */}
        {node.type === 'output' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Output Format</Label>
              <Select
                value={(node.data as Record<string, unknown>).format as string || 'text'}
                onValueChange={(value) => updateNode(node.id, { format: value } as Record<string, unknown>)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select format" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Plain Text</SelectItem>
                  <SelectItem value="json">JSON</SelectItem>
                  <SelectItem value="markdown">Markdown</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Output Variable</Label>
              <Input
                value={(node.data as Record<string, unknown>).variableName as string || 'result'}
                onChange={(e) => updateNode(node.id, { variableName: e.target.value } as Record<string, unknown>)}
                placeholder="result"
                className="font-mono text-xs"
              />
              <p className="text-xs text-dim">The context variable to capture as the final output.</p>
            </div>
          </>
        )}

        {/* Parallel node config */}
        {node.type === 'parallel' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Max Concurrency</Label>
              <Input
                type="number"
                min={1}
                max={10}
                value={(node.data as Record<string, unknown>).maxConcurrency as number || 5}
                onChange={(e) => updateNode(node.id, { maxConcurrency: parseInt(e.target.value) || 5 } as Record<string, unknown>)}
              />
              <p className="text-xs text-dim">Maximum number of branches to execute simultaneously.</p>
            </div>
          </>
        )}

        {/* Condition-specific config */}
        {node.type === 'condition' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Condition Expression</Label>
              <Textarea
                value={node.data.condition || ''}
                onChange={(e) => updateNode(node.id, { condition: e.target.value })}
                placeholder="context.output.includes('yes')"
                rows={3}
                className="font-mono text-xs"
              />
              {node.data.condition && (() => {
                try { new Function('context', `return (${node.data.condition})`); return null; }
                catch { return <p className="text-xs text-rose-400">Invalid JavaScript expression</p>; }
              })()}
              <p className="text-xs text-dim">
                JavaScript expression evaluated against the run context.
              </p>
            </div>
          </>
        )}

        <Separator />

        {/* Delete button */}
        <Button
          variant="destructive"
          size="sm"
          className="w-full"
          onClick={() => {
            removeNode(node.id);
            selectNode(null);
          }}
        >
          Delete Node
        </Button>
      </div>
    </div>
  );
}
