import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { integrationRegistry } from '@/lib/integrations/registry';
import type { IntegrationId } from '@/types/integration';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`actions:${userId}`, 30);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!integrationRegistry.has(id)) {
      return NextResponse.json({ error: 'Unknown integration' }, { status: 404 });
    }

    const integration = integrationRegistry.get(id as IntegrationId);
    const actions = integration.getActions();

    return NextResponse.json({ actions });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `GET /api/integrations/${(await params).id}/actions`);
  }
}
