'use client';

import React, { useEffect, useState, use } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Loader2, Trash2, Copy } from 'lucide-react';
import { toast } from '@/components/ui/use-toast';

interface Trigger {
  id: string;
  workflowId: string;
  integrationId: string;
  eventType: string;
  isActive: boolean;
  createdAt: string;
}

export default function WebhooksPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [loading, setLoading] = useState(true);

  const webhookUrl = `${typeof window !== 'undefined' ? window.location.origin : ''}/api/webhooks/${id}`;

  useEffect(() => {
    fetch('/api/webhooks/triggers')
      .then(r => r.json())
      .then(data => {
        setTriggers((data.triggers || []).filter((t: Trigger) => t.integrationId === id));
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [id]);

  const toggleTrigger = async (triggerId: string, isActive: boolean) => {
    try {
      await fetch(`/api/webhooks/triggers/${triggerId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ isActive }),
      });
      setTriggers(prev =>
        prev.map(t => t.id === triggerId ? { ...t, isActive } : t)
      );
    } catch {
      toast({ title: 'Failed to update trigger', variant: 'destructive' });
    }
  };

  const deleteTrigger = async (triggerId: string) => {
    try {
      await fetch(`/api/webhooks/triggers/${triggerId}`, { method: 'DELETE' });
      setTriggers(prev => prev.filter(t => t.id !== triggerId));
      toast({ title: 'Trigger deleted' });
    } catch {
      toast({ title: 'Failed to delete trigger', variant: 'destructive' });
    }
  };

  const copyUrl = () => {
    navigator.clipboard.writeText(webhookUrl);
    toast({ title: 'Webhook URL copied' });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-dim" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Webhook Triggers</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Configure which workflows run when events arrive from {id}
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Webhook URL</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <code className="flex-1 text-xs bg-input rounded px-3 py-2 text-foreground break-all">
              {webhookUrl}
            </code>
            <Button variant="ghost" size="icon" onClick={copyUrl}>
              <Copy className="h-4 w-4" />
            </Button>
          </div>
          <p className="text-xs text-dim mt-2">
            Register this URL as a webhook endpoint in your {id} settings.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Active Triggers</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {triggers.length === 0 ? (
            <p className="text-sm text-dim">No triggers configured yet.</p>
          ) : (
            triggers.map(trigger => (
              <div key={trigger.id} className="flex items-center justify-between border border-border rounded-lg p-3">
                <div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="font-mono text-xs">
                      {trigger.eventType}
                    </Badge>
                    <span className="text-xs text-dim">→</span>
                    <span className="text-xs text-dim font-mono">{trigger.workflowId.slice(0, 8)}...</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={trigger.isActive}
                    onCheckedChange={(checked) => toggleTrigger(trigger.id, checked)}
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-dim hover:text-rose-400"
                    onClick={() => deleteTrigger(trigger.id)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
