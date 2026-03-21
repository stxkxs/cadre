import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { runs } from '@/lib/db/schema';
import { eq, and } from 'drizzle-orm';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();
    const { id } = await params;
    const [run] = await db
      .select()
      .from(runs)
      .where(and(eq(runs.id, id), eq(runs.userId, userId)));

    if (!run) {
      return NextResponse.json({ error: 'Run not found' }, { status: 404 });
    }

    return NextResponse.json(run);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    return NextResponse.json(
      { error: 'Failed to fetch run' },
      { status: 500 }
    );
  }
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();
    const { id } = await params;
    const body = await request.json();

    if (body.action === 'cancel') {
      const [run] = await db
        .select()
        .from(runs)
        .where(and(eq(runs.id, id), eq(runs.userId, userId)));

      if (!run) {
        return NextResponse.json({ error: 'Run not found' }, { status: 404 });
      }

      if (['completed', 'failed', 'cancelled'].includes(run.status)) {
        return NextResponse.json({ error: 'Run is already finished' }, { status: 400 });
      }

      const [updated] = await db
        .update(runs)
        .set({ status: 'cancelled', completedAt: new Date() })
        .where(and(eq(runs.id, id), eq(runs.userId, userId)))
        .returning();

      return NextResponse.json(updated);
    }

    return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    return NextResponse.json(
      { error: 'Failed to update run' },
      { status: 500 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`delete-run:${userId}`, 30);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    const { id } = await params;

    const [run] = await db
      .select()
      .from(runs)
      .where(and(eq(runs.id, id), eq(runs.userId, userId)));

    if (!run) {
      return NextResponse.json({ error: 'Run not found' }, { status: 404 });
    }

    if (run.status === 'running') {
      return NextResponse.json({ error: 'Cannot delete a running run' }, { status: 400 });
    }

    await db.delete(runs).where(and(eq(runs.id, id), eq(runs.userId, userId)));

    return NextResponse.json({ success: true });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    return NextResponse.json(
      { error: 'Failed to delete run' },
      { status: 500 }
    );
  }
}
