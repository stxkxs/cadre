import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { workflows, runs } from '@/lib/db/schema';
import { eq, and } from 'drizzle-orm';
import { getAuthUserId } from '@/lib/api-auth';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();
    const { id } = await params;

    if (!UUID_RE.test(id)) {
      return NextResponse.json({ error: 'Invalid workflow ID' }, { status: 400 });
    }

    const [workflow] = await db
      .select()
      .from(workflows)
      .where(and(eq(workflows.id, id), eq(workflows.userId, userId)));

    if (!workflow) {
      return NextResponse.json({ error: 'Workflow not found' }, { status: 404 });
    }

    return NextResponse.json(workflow);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error('[GET /api/workflows/:id]', message);
    return NextResponse.json({ error: 'Failed to fetch workflow' }, { status: 500 });
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();
    const { id } = await params;

    if (!UUID_RE.test(id)) {
      return NextResponse.json({ error: 'Invalid workflow ID' }, { status: 400 });
    }

    const body = await request.json();

    // Validate inputs
    if (body.name !== undefined && (typeof body.name !== 'string' || body.name.trim().length === 0)) {
      return NextResponse.json({ error: 'Workflow name cannot be empty' }, { status: 400 });
    }
    if (body.name && body.name.length > 200) {
      return NextResponse.json({ error: 'Workflow name must be under 200 characters' }, { status: 400 });
    }
    if (body.description !== undefined && typeof body.description === 'string' && body.description.length > 5000) {
      return NextResponse.json({ error: 'Description must be under 5000 characters' }, { status: 400 });
    }
    if (body.graphData) {
      const graphStr = JSON.stringify(body.graphData);
      if (graphStr.length > 5_000_000) {
        return NextResponse.json({ error: 'Graph data is too large (max 5MB)' }, { status: 400 });
      }
    }

    // Only update fields that are provided
    const updateData: Record<string, unknown> = { updatedAt: new Date() };
    if (body.name !== undefined) updateData.name = (body.name as string).trim().slice(0, 200);
    if (body.description !== undefined) updateData.description = ((body.description || '') as string).trim().slice(0, 5000);
    if (body.graphData !== undefined) updateData.graphData = body.graphData;
    if (body.variables !== undefined) updateData.variables = body.variables;

    const [updated] = await db
      .update(workflows)
      .set(updateData)
      .where(and(eq(workflows.id, id), eq(workflows.userId, userId)))
      .returning();

    if (!updated) {
      return NextResponse.json({ error: 'Workflow not found' }, { status: 404 });
    }

    return NextResponse.json(updated);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error('[PUT /api/workflows/:id]', message);
    return NextResponse.json({ error: 'Failed to update workflow' }, { status: 500 });
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const userId = await getAuthUserId();
    const { id } = await params;

    if (!UUID_RE.test(id)) {
      return NextResponse.json({ error: 'Invalid workflow ID' }, { status: 400 });
    }

    // Cascade: delete associated runs first
    await db.delete(runs).where(and(eq(runs.workflowId, id), eq(runs.userId, userId)));
    await db.delete(workflows).where(and(eq(workflows.id, id), eq(workflows.userId, userId)));
    return NextResponse.json({ success: true });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error('[DELETE /api/workflows/:id]', message);
    return NextResponse.json({ error: 'Failed to delete workflow' }, { status: 500 });
  }
}
