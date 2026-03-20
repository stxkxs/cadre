import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { db } from '@/lib/db';
import { webhookTriggers } from '@/lib/db/schema';
import { eq } from 'drizzle-orm';
import { integrationRegistry } from '@/lib/integrations/registry';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export async function GET() {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`triggers:${userId}`, 30);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    const triggers = await db
      .select()
      .from(webhookTriggers)
      .where(eq(webhookTriggers.userId, userId));

    return NextResponse.json({ triggers });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, 'GET /api/webhooks/triggers');
  }
}

export async function POST(request: NextRequest) {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`create-trigger:${userId}`, 5);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    const body = await request.json();
    const { workflowId, integrationId, eventType, filter } = body;

    if (!workflowId || !integrationId || !eventType) {
      return NextResponse.json(
        { error: 'workflowId, integrationId, and eventType are required' },
        { status: 400 }
      );
    }

    if (!UUID_RE.test(workflowId)) {
      return NextResponse.json({ error: 'Invalid workflowId format' }, { status: 400 });
    }

    if (typeof eventType !== 'string' || eventType.length > 200) {
      return NextResponse.json({ error: 'eventType must be a string (max 200 chars)' }, { status: 400 });
    }

    if (!integrationRegistry.has(integrationId)) {
      return NextResponse.json({ error: 'Unknown integration' }, { status: 400 });
    }

    const [trigger] = await db
      .insert(webhookTriggers)
      .values({
        userId,
        workflowId,
        integrationId,
        eventType,
        filter: filter || {},
        isActive: true,
      })
      .returning();

    return NextResponse.json({ trigger }, { status: 201 });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, 'POST /api/webhooks/triggers');
  }
}
