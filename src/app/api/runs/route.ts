import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { runs } from '@/lib/db/schema';
import { eq, and, desc } from 'drizzle-orm';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';

export async function GET(request: NextRequest) {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`list-runs:${userId}`, 60);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }
    const workflowId = request.nextUrl.searchParams.get('workflowId');

    const limit = Math.min(parseInt(request.nextUrl.searchParams.get('limit') || '50') || 50, 100);
    const offset = Math.max(parseInt(request.nextUrl.searchParams.get('offset') || '0') || 0, 0);

    const status = request.nextUrl.searchParams.get('status');

    const conditions = [eq(runs.userId, userId)];
    if (workflowId) {
      conditions.push(eq(runs.workflowId, workflowId));
    }
    if (status && ['pending', 'running', 'completed', 'failed', 'cancelled'].includes(status)) {
      conditions.push(eq(runs.status, status));
    }

    const allRuns = await db
      .select({
        id: runs.id,
        workflowId: runs.workflowId,
        status: runs.status,
        tokenUsage: runs.tokenUsage,
        startedAt: runs.startedAt,
        completedAt: runs.completedAt,
      })
      .from(runs)
      .where(and(...conditions))
      .orderBy(desc(runs.startedAt))
      .limit(limit)
      .offset(offset);

    return NextResponse.json(allRuns);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    return NextResponse.json(
      { error: 'Failed to fetch runs' },
      { status: 500 }
    );
  }
}
