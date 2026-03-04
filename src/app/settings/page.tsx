'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import { Brain, Sparkles, Zap, Terminal, Check, X, Loader2, Eye, EyeOff } from 'lucide-react';
import { useSettingsStore } from '@/lib/store/settings-store';
import { toast } from '@/components/ui/use-toast';
import type { ModelProvider } from '@/lib/engine/types';

const providers: { id: ModelProvider; name: string; icon: React.ElementType; placeholder: string; color: string; noApiKey?: boolean }[] = [
  { id: 'anthropic', name: 'Anthropic (Claude)', icon: Brain, placeholder: 'sk-ant-...', color: 'text-orange-400' },
  { id: 'openai', name: 'OpenAI (ChatGPT)', icon: Sparkles, placeholder: 'sk-...', color: 'text-green-400' },
  { id: 'groq', name: 'Groq (Llama)', icon: Zap, placeholder: 'gsk_...', color: 'text-purple-400' },
  { id: 'claude-code', name: 'Claude Code', icon: Terminal, placeholder: '', color: 'text-blue-400', noApiKey: true },
];

export default function SettingsPage() {
  const { theme, setTheme, apiKeyStatuses, setApiKeyStatus } = useSettingsStore();
  const [keys, setKeys] = useState<Record<string, string>>({});
  const [showKeys, setShowKeys] = useState<Record<string, boolean>>({});
  const [validating, setValidating] = useState<Record<string, boolean>>({});

  const handleValidate = async (provider: ModelProvider) => {
    const isCliProvider = providers.find(p => p.id === provider)?.noApiKey;
    const apiKey = keys[provider];
    if (!apiKey && !isCliProvider) return;

    setValidating((v) => ({ ...v, [provider]: true }));
    try {
      const res = await fetch('/api/models/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, apiKey: apiKey || '__cli__' }),
      });
      const data = await res.json();

      setApiKeyStatus(provider, {
        isConfigured: true,
        isValid: data.valid,
        lastValidated: new Date(),
      });

      if (data.valid) {
        toast({ title: `${provider} API key validated`, variant: 'default' });
      } else {
        toast({ title: `Invalid ${provider} API key`, variant: 'destructive' });
      }
    } catch {
      toast({ title: 'Validation failed', variant: 'destructive' });
    } finally {
      setValidating((v) => ({ ...v, [provider]: false }));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">Manage API keys and preferences</p>
      </div>

      {/* API Keys */}
      <Card>
        <CardHeader>
          <CardTitle>API Keys</CardTitle>
          <CardDescription>
            Your keys are encrypted and stored securely. They are only used server-side during workflow execution.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {providers.map((provider) => {
            const status = apiKeyStatuses[provider.id];
            const Icon = provider.icon;

            return (
              <div key={provider.id} className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Icon className={`h-4 w-4 ${provider.color}`} />
                    <Label>{provider.name}</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    {status.lastValidated && (
                      <span className="text-xs text-dim">
                        Checked {new Date(status.lastValidated).toLocaleDateString()}
                      </span>
                    )}
                    {status.isConfigured && (
                      <Badge variant={status.isValid ? 'success' : 'destructive'} className="gap-1">
                        {status.isValid ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                        {status.isValid ? 'Valid' : 'Invalid'}
                      </Badge>
                    )}
                  </div>
                </div>
                {provider.noApiKey ? (
                  <div className="flex items-center gap-2">
                    <p className="text-xs text-dim flex-1">
                      Uses its own authentication. Make sure the <code className="bg-input px-1 rounded">claude</code> CLI is installed and authenticated.
                    </p>
                    <Button
                      variant="secondary"
                      onClick={() => handleValidate(provider.id)}
                      disabled={validating[provider.id]}
                    >
                      {validating[provider.id] ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        'Check CLI'
                      )}
                    </Button>
                  </div>
                ) : (
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <Input
                        type={showKeys[provider.id] ? 'text' : 'password'}
                        placeholder={provider.placeholder}
                        value={keys[provider.id] || ''}
                        onChange={(e) => setKeys((k) => ({ ...k, [provider.id]: e.target.value }))}
                      />
                      <button
                        type="button"
                        onClick={() => setShowKeys((s) => ({ ...s, [provider.id]: !s[provider.id] }))}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-dim hover:text-foreground cursor-pointer"
                      >
                        {showKeys[provider.id] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                    <Button
                      variant="secondary"
                      onClick={() => handleValidate(provider.id)}
                      disabled={!keys[provider.id] || validating[provider.id]}
                    >
                      {validating[provider.id] ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        'Validate'
                      )}
                    </Button>
                  </div>
                )}
                {provider.id !== 'claude-code' && <Separator className="mt-4" />}
              </div>
            );
          })}
        </CardContent>
      </Card>

      {/* Preferences */}
      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <Label>Dark Mode</Label>
              <p className="text-xs text-dim">Use dark theme</p>
            </div>
            <Switch
              checked={theme === 'dark'}
              onCheckedChange={(checked) => setTheme(checked ? 'dark' : 'light')}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
