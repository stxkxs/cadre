import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { integrationRegistry } from '@/lib/integrations/registry';
import { getValidCredentials } from '@/lib/integrations/credentials';
import type { IntegrationId } from '@/types/integration';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`integration-status:${userId}`, 20);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!integrationRegistry.has(id)) {
      return NextResponse.json({ error: 'Unknown integration' }, { status: 404 });
    }

    const credentials = await getValidCredentials(userId, id as IntegrationId);
    if (!credentials) {
      return NextResponse.json({ connected: false, healthy: false });
    }

    const integration = integrationRegistry.get(id as IntegrationId);
    const healthy = await integration.testConnection(credentials);

    return NextResponse.json({ connected: true, healthy });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `GET /api/integrations/${(await params).id}/status`);
  }
}
