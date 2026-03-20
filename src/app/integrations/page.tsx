'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Plug, Check, Loader2 } from 'lucide-react';
import { toast } from '@/components/ui/use-toast';

interface IntegrationInfo {
  id: string;
  name: string;
  config: { color: string; capabilities: string[] };
  connected: boolean;
  connectedAt: string | null;
}

export default function IntegrationsPage() {
  const [integrations, setIntegrations] = useState<IntegrationInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [disconnecting, setDisconnecting] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/integrations')
      .then(r => r.json())
      .then(data => {
        setIntegrations(data.integrations || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleConnect = (id: string) => {
    window.location.href = `/api/integrations/${id}/connect`;
  };

  const handleDisconnect = async (id: string) => {
    setDisconnecting(id);
    try {
      const res = await fetch(`/api/integrations/${id}/disconnect`, { method: 'DELETE' });
      if (res.ok) {
        setIntegrations(prev =>
          prev.map(i => i.id === id ? { ...i, connected: false, connectedAt: null } : i)
        );
        toast({ title: 'Disconnected successfully' });
      }
    } catch {
      toast({ title: 'Failed to disconnect', variant: 'destructive' });
    } finally {
      setDisconnecting(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-dim" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Integrations</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Connect external services to use in your workflows
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {integrations.map((integration) => (
          <Card key={integration.id} className="overflow-hidden">
            <div
              className="h-1"
              style={{ backgroundColor: integration.config.color }}
            />
            <CardContent className="p-4">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className="flex items-center justify-center w-10 h-10 rounded-lg"
                    style={{ backgroundColor: `${integration.config.color}15` }}
                  >
                    <Plug className="w-5 h-5" style={{ color: integration.config.color }} />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-foreground">{integration.name}</h3>
                    <div className="flex gap-1 mt-1">
                      {integration.config.capabilities.map(cap => (
                        <Badge key={cap} variant="secondary" className="text-[10px] px-1.5 py-0">
                          {cap}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {integration.connected ? (
                    <>
                      <Badge variant="success" className="gap-1">
                        <Check className="h-3 w-3" />
                        Connected
                      </Badge>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDisconnect(integration.id)}
                        disabled={disconnecting === integration.id}
                      >
                        {disconnecting === integration.id ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          'Disconnect'
                        )}
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleConnect(integration.id)}
                    >
                      Connect
                    </Button>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
