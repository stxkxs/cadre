import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { db } from '@/lib/db';
import { webhookTriggers } from '@/lib/db/schema';
import { eq, and } from 'drizzle-orm';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`trigger-update:${userId}`, 20);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!UUID_RE.test(id)) {
      return NextResponse.json({ error: 'Invalid trigger ID' }, { status: 400 });
    }

    const body = await request.json();
    if (typeof body.isActive !== 'boolean') {
      return NextResponse.json({ error: 'isActive must be a boolean' }, { status: 400 });
    }

    const [trigger] = await db
      .update(webhookTriggers)
      .set({ isActive: body.isActive })
      .where(and(eq(webhookTriggers.id, id), eq(webhookTriggers.userId, userId)))
      .returning();

    if (!trigger) {
      return NextResponse.json({ error: 'Trigger not found' }, { status: 404 });
    }

    return NextResponse.json({ trigger });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `PATCH /api/webhooks/triggers/${(await params).id}`);
  }
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`trigger-delete:${userId}`, 20);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!UUID_RE.test(id)) {
      return NextResponse.json({ error: 'Invalid trigger ID' }, { status: 400 });
    }

    const [deleted] = await db
      .delete(webhookTriggers)
      .where(and(eq(webhookTriggers.id, id), eq(webhookTriggers.userId, userId)))
      .returning();

    if (!deleted) {
      return NextResponse.json({ error: 'Trigger not found' }, { status: 404 });
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `DELETE /api/webhooks/triggers/${(await params).id}`);
  }
}
