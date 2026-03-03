'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import { Search, Brain, Sparkles, Zap, Terminal, Plus, Copy } from 'lucide-react';
import { toast } from '@/components/ui/use-toast';

interface Template {
  id: string;
  name: string;
  description: string;
  category: string;
  nodeData: {
    type: string;
    data: {
      label: string;
      provider: string;
      model: string;
      systemPrompt: string;
      temperature: number;
      maxTokens: number;
    };
  };
  isBuiltIn: boolean;
}

const providerIcons: Record<string, React.ElementType> = {
  anthropic: Brain,
  openai: Sparkles,
  groq: Zap,
  'claude-code': Terminal,
};

const providerColors: Record<string, string> = {
  anthropic: 'text-orange-500 bg-orange-500/10',
  openai: 'text-green-500 bg-green-500/10',
  groq: 'text-purple-500 bg-purple-500/10',
  'claude-code': 'text-blue-500 bg-blue-500/10',
};

export default function LibraryPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState('All');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function fetchTemplates() {
      try {
        const res = await fetch('/api/library');
        if (res.ok) {
          setTemplates(await res.json());
        }
      } catch {
        // Silently handle
      } finally {
        setIsLoading(false);
      }
    }
    fetchTemplates();
  }, []);

  const categories = ['All', ...Array.from(new Set(templates.map((t) => t.category)))];

  const filtered = templates.filter((t) => {
    const matchesSearch =
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.description.toLowerCase().includes(search.toLowerCase());
    const matchesCategory = activeCategory === 'All' || t.category === activeCategory;
    return matchesSearch && matchesCategory;
  });

  const handleCopy = (template: Template) => {
    const nodeConfig = JSON.stringify(template.nodeData, null, 2);
    navigator.clipboard.writeText(nodeConfig).then(() => {
      toast({ title: `Copied "${template.name}" config to clipboard` });
    }).catch(() => {
      toast({ title: 'Failed to copy', variant: 'destructive' });
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Agent Library</h1>
        <p className="text-sm text-muted-foreground mt-1">Pre-built agent nodes ready to use in your workflows</p>
      </div>

      <div className="flex items-center gap-4 flex-wrap">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-dim" />
          <Input placeholder="Search agents..." value={search} onChange={(e) => setSearch(e.target.value)} className="pl-9" />
        </div>
        {!isLoading && categories.length > 1 && (
          <Tabs value={activeCategory} onValueChange={setActiveCategory}>
            <TabsList>
              {categories.map((cat) => (
                <TabsTrigger key={cat} value={cat} className="text-xs">{cat}</TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        )}
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-5 space-y-3">
                <div className="flex items-start justify-between">
                  <Skeleton className="w-10 h-10 rounded-xl" />
                  <Skeleton className="w-16 h-5 rounded-full" />
                </div>
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-3 w-24" />
                <div className="flex gap-2">
                  <Skeleton className="h-8 flex-1" />
                  <Skeleton className="h-8 flex-1" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.length === 0 && (
            <div className="col-span-full flex flex-col items-center py-16 text-center">
              <p className="text-sm text-muted-foreground">
                {search ? `No agents match "${search}"` : 'No agents available'}
              </p>
            </div>
          )}
          {filtered.map((template) => {
            const provider = template.nodeData?.data?.provider || 'anthropic';
            const Icon = providerIcons[provider] || Brain;
            return (
              <Card key={template.id} className="group hover:border-input-border transition-colors">
                <CardContent className="p-5">
                  <div className="flex items-start justify-between mb-3">
                    <div className={`w-10 h-10 rounded-xl ${providerColors[provider] || providerColors.anthropic} flex items-center justify-center`}>
                      <Icon className="h-5 w-5" />
                    </div>
                    <Badge variant="secondary" className="text-xs">{template.category}</Badge>
                  </div>
                  <h3 className="font-semibold text-foreground mb-1">{template.name}</h3>
                  <p className="text-sm text-muted-foreground mb-1">{template.description}</p>
                  <p className="text-xs text-dim mb-3 font-mono">{template.nodeData?.data?.model}</p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="gap-1.5 flex-1"
                      onClick={() => handleCopy(template)}
                    >
                      <Copy className="h-3.5 w-3.5" />
                      Copy
                    </Button>
                    <Button
                      size="sm"
                      className="gap-1.5 flex-1"
                      onClick={() => toast({ title: `Open a workflow to add "${template.name}"` })}
                    >
                      <Plus className="h-3.5 w-3.5" />
                      Use
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
