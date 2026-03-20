import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { workflows, runs, userApiKeys, integrationConnections } from '@/lib/db/schema';
import { eq, and } from 'drizzle-orm';
import { Executor } from '@/lib/engine/executor';
import { Graph } from '@/lib/engine/graph';
import { decryptApiKey } from '@/lib/crypto';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import type { WorkflowNode, WorkflowEdge } from '@/lib/engine/types';
import type { IntegrationCredentials } from '@/types/integration';
import { mkdirSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    // Rate limit: 10 runs per minute per user
    const rl = rateLimit(`run:${userId}`, 10);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests. Please wait.' }, { status: 429 });
    }

    // Validate UUID format
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id)) {
      return NextResponse.json({ error: 'Invalid workflow ID' }, { status: 400 });
    }

    // Fetch workflow
    const [workflow] = await db
      .select()
      .from(workflows)
      .where(and(eq(workflows.id, id), eq(workflows.userId, userId)));

    if (!workflow) {
      return NextResponse.json({ error: 'Workflow not found' }, { status: 404 });
    }

    const graphData = workflow.graphData as { nodes: WorkflowNode[]; edges: WorkflowEdge[] };

    // Validate graph before execution
    if (!graphData.nodes || graphData.nodes.length === 0) {
      console.error(`[run] Workflow ${id}: no nodes`);
      return NextResponse.json(
        { error: 'Workflow has no nodes. Add nodes before running.' },
        { status: 400 }
      );
    }

    const graph = new Graph(graphData.nodes, graphData.edges || []);
    const validation = graph.validate();
    if (!validation.valid) {
      console.error(`[run] Workflow ${id}: validation failed:`, validation.errors);
      return NextResponse.json(
        { error: 'Invalid workflow graph', details: validation.errors },
        { status: 400 }
      );
    }

    // Fetch API keys
    const keys = await db
      .select()
      .from(userApiKeys)
      .where(eq(userApiKeys.userId, userId));

    const apiKeys: Record<string, string> = {};
    for (const key of keys) {
      try {
        apiKeys[key.provider] = decryptApiKey(key.encryptedKey, key.iv, key.authTag, userId);
      } catch {
        // Skip invalid keys
      }
    }

    // Check if required providers have keys
    const requiredProviders = new Set(
      graphData.nodes
        .filter((n) => n.type === 'agent' && n.data?.provider)
        .map((n) => n.data.provider as string)
    );

    console.log(`[run] Workflow ${id}: required providers:`, [...requiredProviders], 'configured keys:', Object.keys(apiKeys));
    const missingKeys = [...requiredProviders].filter((p) => p !== 'claude-code' && p !== 'bedrock' && !apiKeys[p]);
    if (missingKeys.length > 0) {
      console.error(`[run] Workflow ${id}: missing keys for:`, missingKeys);
      return NextResponse.json(
        { error: `Missing API keys for: ${missingKeys.join(', ')}. Configure them in Settings.` },
        { status: 400 }
      );
    }

    // Fetch integration credentials for integration nodes
    const integrationCredentialsMap: Record<string, IntegrationCredentials> = {};
    const requiredIntegrations = new Set(
      graphData.nodes
        .filter((n) => n.type === 'integration' && n.data?.integrationId)
        .map((n) => n.data.integrationId as string)
    );

    if (requiredIntegrations.size > 0) {
      const connections = await db
        .select()
        .from(integrationConnections)
        .where(and(eq(integrationConnections.userId, userId), eq(integrationConnections.isActive, true)));

      for (const conn of connections) {
        if (requiredIntegrations.has(conn.integrationId)) {
          try {
            const accessToken = decryptApiKey(conn.encryptedAccessToken, conn.iv, conn.authTag, userId);
            integrationCredentialsMap[conn.integrationId] = {
              accessToken,
              metadata: (conn.metadata as Record<string, unknown>) || undefined,
            };
          } catch { /* skip invalid */ }
        }
      }

      const missingIntegrations = [...requiredIntegrations].filter(i => !integrationCredentialsMap[i]);
      if (missingIntegrations.length > 0) {
        return NextResponse.json(
          { error: `Missing integration connections for: ${missingIntegrations.join(', ')}. Connect them in Settings → Integrations.` },
          { status: 400 }
        );
      }
    }

    // Check if any agent nodes have workspace enabled
    const hasWorkspace = graphData.nodes.some(
      (n) => n.type === 'agent' && n.data?.workspace && n.data.workspace !== 'off'
    );

    // Create workspace directory if needed
    let workspacePath: string | undefined;
    if (hasWorkspace) {
      workspacePath = join(homedir(), '.cadre', 'workspaces', id);
      mkdirSync(workspacePath, { recursive: true });
    }

    // Create run record
    const [run] = await db
      .insert(runs)
      .values({
        workflowId: id,
        userId,
        status: 'running',
      })
      .returning();

    // Execute workflow in background, with live DB updates
    const executor = new Executor(graphData.nodes, graphData.edges || [], {
      apiKeys,
      variables: (workflow.variables as Record<string, string>) || {},
      workspacePath,
      integrationCredentials: integrationCredentialsMap,
      onEvent: async (event) => {
        // Write intermediate state to DB so SSE stream picks it up
        if (event.type === 'node-start' || event.type === 'node-complete' || event.type === 'node-error') {
          try {
            const state = executor.getState();
            await db
              .update(runs)
              .set({ nodeStates: state.nodeStates, tokenUsage: state.totalTokens })
              .where(eq(runs.id, run.id));
          } catch { /* best effort */ }
        }
      },
    });

    executor.execute().then(async (result) => {
      const context = result.context as Record<string, unknown>;
      // Store workspace path in run context so the UI can reference it
      if (workspacePath) {
        context._workspacePath = workspacePath;
      }
      await db
        .update(runs)
        .set({
          status: result.status,
          nodeStates: result.nodeStates,
          tokenUsage: result.totalTokens,
          context,
          completedAt: new Date(),
        })
        .where(eq(runs.id, run.id));
    }).catch(async (error) => {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      console.error(`[run ${run.id}] Execution failed:`, errorMessage);
      await db
        .update(runs)
        .set({
          status: 'failed',
          nodeStates: { _error: { status: 'failed', error: errorMessage } },
          completedAt: new Date(),
        })
        .where(eq(runs.id, run.id));
    });

    return NextResponse.json({ runId: run.id, status: 'running' }, { status: 202 });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, 'POST /api/workflows/:id/run');
  }
}
