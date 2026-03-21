import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { workflows } from '@/lib/db/schema';
import { getAuthUserId } from '@/lib/api-auth';
import { eq, desc } from 'drizzle-orm';
import { logger } from '@/lib/logger';
import { rateLimit } from '@/lib/rate-limit';

export async function GET(request: NextRequest) {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`list-workflows:${userId}`, 60);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }
    const limit = Math.min(parseInt(request.nextUrl.searchParams.get('limit') || '50') || 50, 100);
    const offset = Math.max(parseInt(request.nextUrl.searchParams.get('offset') || '0') || 0, 0);

    const allWorkflows = await db
      .select()
      .from(workflows)
      .where(eq(workflows.userId, userId))
      .orderBy(desc(workflows.updatedAt))
      .limit(limit)
      .offset(offset);
    return NextResponse.json(allWorkflows);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    const message = error instanceof Error ? error.message : 'Unknown error';
    logger.error('Failed to fetch workflows', { route: 'GET /api/workflows', error: message });
    return NextResponse.json(
      { error: 'Failed to fetch workflows', detail: message },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const userId = await getAuthUserId();
    const body = await request.json();
    const { name, description, graphData, variables } = body;

    if (!name || typeof name !== 'string' || name.trim().length === 0) {
      return NextResponse.json({ error: 'Workflow name is required' }, { status: 400 });
    }
    if (name.trim().length > 200) {
      return NextResponse.json({ error: 'Workflow name must be under 200 characters' }, { status: 400 });
    }
    if (description && typeof description === 'string' && description.length > 5000) {
      return NextResponse.json({ error: 'Description must be under 5000 characters' }, { status: 400 });
    }
    if (graphData) {
      const graphStr = JSON.stringify(graphData);
      if (graphStr.length > 5_000_000) {
        return NextResponse.json({ error: 'Graph data is too large (max 5MB)' }, { status: 400 });
      }
    }

    const [workflow] = await db
      .insert(workflows)
      .values({
        userId,
        name: name.trim().slice(0, 200),
        description: ((description || '') as string).trim().slice(0, 5000),
        graphData: graphData || { nodes: [], edges: [] },
        variables: variables || {},
      })
      .returning();

    return NextResponse.json(workflow, { status: 201 });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    const message = error instanceof Error ? error.message : 'Unknown error';
    logger.error('Failed to create workflow', { route: 'POST /api/workflows', error: message });
    return NextResponse.json(
      { error: 'Failed to create workflow', detail: message },
      { status: 500 }
    );
  }
}
