import { NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { db } from '@/lib/db';
import { integrationConnections } from '@/lib/db/schema';
import { eq } from 'drizzle-orm';
import { integrationRegistry } from '@/lib/integrations/registry';

export async function GET() {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`integrations:${userId}`, 30);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    const connections = await db
      .select({
        integrationId: integrationConnections.integrationId,
        isActive: integrationConnections.isActive,
        updatedAt: integrationConnections.updatedAt,
      })
      .from(integrationConnections)
      .where(eq(integrationConnections.userId, userId));

    const connectionMap = new Map(connections.map(c => [c.integrationId, c]));

    const integrations = integrationRegistry.getAll().map(integration => ({
      id: integration.id,
      name: integration.name,
      config: integration.config,
      connected: connectionMap.has(integration.id) && connectionMap.get(integration.id)?.isActive,
      connectedAt: connectionMap.get(integration.id)?.updatedAt || null,
    }));

    return NextResponse.json({ integrations });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, 'GET /api/integrations');
  }
}
