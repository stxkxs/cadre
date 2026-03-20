import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { webhookEvents, webhookTriggers, workflows, runs } from '@/lib/db/schema';
import { eq, and, inArray } from 'drizzle-orm';
import { integrationRegistry } from '@/lib/integrations/registry';
import { getWebhookSecret } from '@/lib/integrations/webhook-secrets';
import { rateLimit } from '@/lib/rate-limit';
import { Executor } from '@/lib/engine/executor';
import { decryptApiKey } from '@/lib/crypto';
import { logger } from '@/lib/logger';
import { userApiKeys } from '@/lib/db/schema';
import type { IntegrationId } from '@/types/integration';
import type { WorkflowNode, WorkflowEdge } from '@/lib/engine/types';

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ provider: string }> }
) {
  const { provider } = await params;

  // Rate limit webhooks per provider (no auth — external service calls)
  const rl = rateLimit(`webhook:${provider}`, 60);
  if (!rl.success) {
    return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
  }

  if (!integrationRegistry.has(provider)) {
    return NextResponse.json({ error: 'Unknown provider' }, { status: 404 });
  }

  const integrationId = provider as IntegrationId;
  const integration = integrationRegistry.get(integrationId);
  const rawBody = await request.text();

  // Verify webhook signature
  const secret = getWebhookSecret(integrationId);
  if (secret) {
    const headers: Record<string, string> = {};
    request.headers.forEach((value, key) => { headers[key] = value; });

    const valid = integration.verifyWebhookSignature(
      { headers, body: rawBody, rawBody: Buffer.from(rawBody) },
      secret
    );
    if (!valid) {
      return NextResponse.json({ error: 'Invalid signature' }, { status: 401 });
    }
  }

  // Parse event
  const headers: Record<string, string> = {};
  request.headers.forEach((value, key) => { headers[key] = value; });
  const event = integration.parseWebhookEvent({
    headers,
    body: rawBody,
    rawBody: Buffer.from(rawBody),
  });

  // Log event
  const [webhookEvent] = await db
    .insert(webhookEvents)
    .values({
      integrationId,
      eventType: event.eventType,
      payload: event.payload,
      sourceId: event.sourceId,
      status: 'received',
    })
    .returning();

  // Find matching triggers and fire workflows (fire-and-forget)
  const triggers = await db
    .select()
    .from(webhookTriggers)
    .where(
      and(
        eq(webhookTriggers.integrationId, integrationId),
        eq(webhookTriggers.eventType, event.eventType),
        eq(webhookTriggers.isActive, true)
      )
    );

  if (triggers.length > 0) {
    // Batch-fetch all needed workflows and API keys in 2 queries
    const workflowIds = [...new Set(triggers.map(t => t.workflowId))];
    const userIds = [...new Set(triggers.map(t => t.userId))];

    const [batchedWorkflows, batchedKeys] = await Promise.all([
      db.select().from(workflows).where(inArray(workflows.id, workflowIds)),
      db.select().from(userApiKeys).where(inArray(userApiKeys.userId, userIds)),
    ]);

    const workflowMap = new Map(batchedWorkflows.map(w => [w.id, w]));
    const keysByUser = new Map<string, typeof batchedKeys>();
    for (const key of batchedKeys) {
      const existing = keysByUser.get(key.userId) || [];
      existing.push(key);
      keysByUser.set(key.userId, existing);
    }

    for (const trigger of triggers) {
      fireWorkflow(
        trigger.workflowId, trigger.userId, event.payload, webhookEvent.id,
        workflowMap.get(trigger.workflowId), keysByUser.get(trigger.userId) || []
      ).catch(err => {
        logger.error('Failed to fire workflow from webhook', { workflowId: trigger.workflowId, error: String(err) });
      });
    }
  }

  // Update event status
  await db
    .update(webhookEvents)
    .set({ status: triggers.length > 0 ? 'processed' : 'ignored', processedAt: new Date() })
    .where(eq(webhookEvents.id, webhookEvent.id));

  return NextResponse.json({ received: true });
}

async function fireWorkflow(
  workflowId: string,
  userId: string,
  webhookPayload: Record<string, unknown>,
  eventId: string,
  prefetchedWorkflow?: typeof workflows.$inferSelect,
  prefetchedKeys?: (typeof userApiKeys.$inferSelect)[]
): Promise<void> {
  const workflow = prefetchedWorkflow ?? (await db
    .select()
    .from(workflows)
    .where(and(eq(workflows.id, workflowId), eq(workflows.userId, userId)))
  )[0];

  if (!workflow || workflow.userId !== userId) return;

  const graphData = workflow.graphData as { nodes: WorkflowNode[]; edges: WorkflowEdge[] };

  // Use pre-fetched keys or fetch individually
  const keys = prefetchedKeys ?? await db.select().from(userApiKeys).where(eq(userApiKeys.userId, userId));
  const apiKeys: Record<string, string> = {};
  for (const key of keys) {
    try {
      apiKeys[key.provider] = decryptApiKey(key.encryptedKey, key.iv, key.authTag, userId);
    } catch (err) { logger.warn('Failed to decrypt API key', { provider: key.provider, userId, error: String(err) }); }
  }

  const [run] = await db
    .insert(runs)
    .values({ workflowId, userId, status: 'running' })
    .returning();

  const executor = new Executor(graphData.nodes, graphData.edges || [], {
    apiKeys,
    variables: {
      ...((workflow.variables as Record<string, string>) || {}),
      _webhookPayload: JSON.stringify(webhookPayload),
      _webhookEventId: eventId,
    },
  });

  executor.execute().then(async (result) => {
    await db
      .update(runs)
      .set({
        status: result.status,
        nodeStates: result.nodeStates,
        tokenUsage: result.totalTokens,
        context: result.context,
        completedAt: new Date(),
      })
      .where(eq(runs.id, run.id));
  }).catch(async (error) => {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    await db
      .update(runs)
      .set({ status: 'failed', nodeStates: { _error: { status: 'failed', error: errorMessage } }, completedAt: new Date() })
      .where(eq(runs.id, run.id));
  });
}
