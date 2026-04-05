'use client';

import React from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { useWorkflowStore } from '@/lib/store/workflow-store';

export function ConfigPanel() {
  const { selectedNodeId, nodes, edges, updateNode, selectNode, removeNode } = useWorkflowStore();
  const node = nodes.find((n) => n.id === selectedNodeId);

  if (!node) return null;

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

        {/* Agent config */}
        {node.type === 'agent' && (
          <>
            <Separator />

            {/* Provider */}
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select
                value={node.data.provider || 'claude-code'}
                onValueChange={(value) => updateNode(node.id, { provider: value } as Record<string, unknown>)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="claude-code">Claude Code</SelectItem>
                  <SelectItem value="codex">Codex</SelectItem>
                  <SelectItem value="gemini">Gemini</SelectItem>
                </SelectContent>
              </Select>
            </div>

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

            {/* Max Turns */}
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

            {/* Timeout */}
            <div className="space-y-2">
              <Label>Timeout (seconds)</Label>
              <Input
                type="number"
                min={5}
                max={3600}
                value={node.data.timeout || 600}
                onChange={(e) => {
                  updateNode(node.id, { timeout: Math.min(3600, Math.max(5, parseInt(e.target.value) || 600)) });
                }}
              />
              <p className="text-xs text-dim">Max execution time (5–3600s)</p>
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

            {/* Permission Mode */}
            <div className="space-y-2">
              <Label>Permission Mode</Label>
              <Select
                value={node.data.permissionMode || 'default'}
                onValueChange={(value) => updateNode(node.id, { permissionMode: value } as Record<string, unknown>)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select mode" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="default">Default</SelectItem>
                  <SelectItem value="accept-edits">Accept Edits</SelectItem>
                  <SelectItem value="full">Full Autonomy</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-dim">
                {node.data.permissionMode === 'full'
                  ? 'Skips all permission prompts. Use with caution.'
                  : node.data.permissionMode === 'accept-edits'
                  ? 'Auto-accepts file edits, prompts for other actions.'
                  : 'Prompts for all potentially destructive actions.'}
              </p>
            </div>

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
                  <SelectItem value="full">Full Autonomy</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-dim">
                {node.data.workspace === 'safe'
                  ? 'Node runs in workspace directory. Output saved to file.'
                  : node.data.workspace === 'full'
                  ? 'Full autonomy — skips permission prompts.'
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

        {/* Router-specific config */}
        {node.type === 'router' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Routes</Label>
              {(node.data.routes || []).map((route: { label: string; condition?: string }, i: number) => (
                <div key={i} className="space-y-1 p-2 rounded-md border border-border">
                  <div className="flex items-center gap-1.5">
                    <Input
                      value={route.label}
                      onChange={(e) => {
                        const routes = [...(node.data.routes || [])];
                        routes[i] = { ...routes[i], label: e.target.value };
                        updateNode(node.id, { routes } as Record<string, unknown>);
                      }}
                      placeholder="Route label"
                      className="text-xs"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0"
                      onClick={() => {
                        const routes = (node.data.routes || []).filter((_: unknown, j: number) => j !== i);
                        updateNode(node.id, { routes } as Record<string, unknown>);
                      }}
                    >
                      <span className="text-xs text-rose-400">×</span>
                    </Button>
                  </div>
                  <Textarea
                    value={route.condition || ''}
                    onChange={(e) => {
                      const routes = [...(node.data.routes || [])];
                      routes[i] = { ...routes[i], condition: e.target.value };
                      updateNode(node.id, { routes } as Record<string, unknown>);
                    }}
                    placeholder="Optional condition expression"
                    rows={1}
                    className="font-mono text-xs"
                  />
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => {
                  const routes = [...(node.data.routes || []), { label: `Route ${(node.data.routes || []).length + 1}` }];
                  updateNode(node.id, { routes } as Record<string, unknown>);
                }}
              >
                Add Route
              </Button>
              <p className="text-xs text-dim">
                Each route can have an optional condition. First matching route wins. Routes without conditions act as defaults.
              </p>
            </div>
          </>
        )}

        {/* Transform-specific config */}
        {node.type === 'transform' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Template</Label>
              <Textarea
                value={node.data.template || ''}
                onChange={(e) => updateNode(node.id, { template: e.target.value } as Record<string, unknown>)}
                placeholder="Summary: {{node_id_output}}"
                rows={4}
                className="font-mono text-xs"
              />
              <p className="text-xs text-dim">
                {"Use {{variable}} to interpolate context values. No LLM call — pure text transformation."}
              </p>
            </div>
          </>
        )}

        {/* Gate-specific config */}
        {node.type === 'gate' && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Approval Message</Label>
              <Textarea
                value={node.data.gateMessage || ''}
                onChange={(e) => updateNode(node.id, { gateMessage: e.target.value } as Record<string, unknown>)}
                placeholder="Review the changes before continuing..."
                rows={3}
              />
              <p className="text-xs text-dim">
                Pauses execution until manually approved or rejected in the run monitor.
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
