'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { GitBranch, Play, Clock, Zap, Plus, ArrowRight, FolderOpen } from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { formatDate, formatTokens, formatCost } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

interface WorkflowItem {
  id: string;
  name: string;
  graphData: { nodes: unknown[] };
}

interface RunItem {
  id: string;
  workflowId: string;
  status: string;
  tokenUsage: { input: number; output: number; cost: number };
  startedAt: string;
  completedAt: string | null;
}

const statusColors: Record<string, 'success' | 'warning' | 'destructive' | 'default'> = {
  completed: 'success',
  running: 'warning',
  failed: 'destructive',
  pending: 'default',
  cancelled: 'destructive',
};

const statIconColors = [
  { iconBg: 'bg-[#0AEFB7]/5', iconText: 'text-[#0AEFB7]' },
  { iconBg: 'bg-sky-500/5', iconText: 'text-sky-500' },
  { iconBg: 'bg-amber-500/5', iconText: 'text-amber-500' },
  { iconBg: 'bg-emerald-500/5', iconText: 'text-emerald-500' },
];

export default function DashboardPage() {
  const [workflows, setWorkflows] = useState<WorkflowItem[]>([]);
  const [runs, setRuns] = useState<RunItem[]>([]);
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function fetchData() {
      try {
        const [wRes, rRes] = await Promise.all([
          fetch('/api/workflows'),
          fetch('/api/runs'),
        ]);
        if (wRes.ok) setWorkflows(await wRes.json());
        if (rRes.ok) setRuns(await rRes.json());
      } catch {
        // Silently handle — shows zeros
      } finally {
        setIsLoading(false);
      }
    }
    fetchData();
  }, []);

  const totalRuns = runs.length;
  const completedRuns = runs.filter((r) => r.status === 'completed').length;
  const totalTokens = runs.reduce((sum, r) => {
    const t = r.tokenUsage || { input: 0, output: 0 };
    return sum + (t.input || 0) + (t.output || 0);
  }, 0);
  const totalCost = runs.reduce((sum, r) => sum + ((r.tokenUsage as Record<string, number>)?.cost || 0), 0);

  const workflowNames = new Map(workflows.map((w) => [w.id, w.name]));
  const recentRuns = runs.slice(0, 6);

  const stats = [
    { label: 'Workflows', value: workflows.length.toString(), icon: GitBranch, sub: `${workflows.length} total` },
    { label: 'Total Runs', value: totalRuns.toString(), icon: Play, sub: `${completedRuns} completed` },
    { label: 'Tokens Used', value: formatTokens(totalTokens), icon: Zap, sub: formatCost(totalCost) + ' total cost' },
    { label: 'Success Rate', value: totalRuns > 0 ? `${Math.round((completedRuns / totalRuns) * 100)}%` : '-', icon: Clock, sub: `${totalRuns - completedRuns} failed/pending` },
  ];

  return (
    <div className="space-y-6">
      {/* Hero mesh gradient */}
      <div className="mesh-gradient -m-6 mb-0 px-6 pt-6 pb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-foreground font-display">Dashboard</h1>
            <p className="text-sm text-muted-foreground mt-1">Overview of your agent workflows</p>
          </div>
          <Link href="/workflows/new">
            <Button className="gap-2">
              <Plus className="h-4 w-4" />
              New Workflow
            </Button>
          </Link>
        </div>
      </div>

      {/* Stats Cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-8 w-16" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                  <Skeleton className="h-12 w-12 rounded-lg" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, i) => (
          <Card key={stat.label}>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">{stat.label}</p>
                  <p className="text-2xl font-bold text-foreground mt-1 font-display">{stat.value}</p>
                  <p className="text-xs text-dim mt-1">{stat.sub}</p>
                </div>
                <div className={`flex items-center justify-center w-12 h-12 rounded-lg ${statIconColors[i].iconBg}`}>
                  <stat.icon className={`h-6 w-6 ${statIconColors[i].iconText}`} />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card className="lg:col-span-1 border-dashed border-[#0AEFB7]/20 bg-[#0AEFB7]/[0.03]">
          <CardContent className="p-6 flex flex-col items-center justify-center text-center h-full min-h-[200px]">
            <div className="w-14 h-14 rounded-2xl bg-[#0AEFB7]/5 flex items-center justify-center mb-4">
              <Plus className="h-7 w-7 text-[#0AEFB7]" />
            </div>
            <h3 className="font-semibold text-foreground mb-2 font-display">Create Workflow</h3>
            <p className="text-sm text-muted-foreground mb-4">Build agent pipelines with our visual graph editor</p>
            <Link href="/workflows/new">
              <Button variant="outline" size="sm" className="gap-1.5">
                Get Started <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            </Link>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Recent Runs</CardTitle>
              <Link href="/workflows">
                <Button variant="ghost" size="sm" className="text-xs gap-1">
                  View All <ArrowRight className="h-3 w-3" />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {recentRuns.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <FolderOpen className="h-8 w-8 text-dim mb-2" />
                <p className="text-sm text-muted-foreground">No runs yet. Execute a workflow to see results here.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {recentRuns.map((run) => {
                  const tokens = run.tokenUsage || { input: 0, output: 0 };
                  return (
                    <div
                      key={run.id}
                      className="flex items-center justify-between rounded-lg bg-input-50 px-4 py-3 cursor-pointer hover:bg-hover transition-colors"
                      onClick={() => router.push(`/workflows/${run.workflowId}/runs/${run.id}`)}
                    >
                      <div className="flex items-center gap-3">
                        <Badge variant={statusColors[run.status] || 'default'}>{run.status}</Badge>
                        <span className="text-sm font-medium text-foreground">
                          {workflowNames.get(run.workflowId) || 'Unknown workflow'}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <span>{formatTokens((tokens.input || 0) + (tokens.output || 0))} tokens</span>
                        <span>{formatDate(run.startedAt)}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
