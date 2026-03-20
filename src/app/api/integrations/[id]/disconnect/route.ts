import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { integrationRegistry } from '@/lib/integrations/registry';
import { removeCredentials } from '@/lib/integrations/credentials';
import type { IntegrationId } from '@/types/integration';

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`disconnect:${userId}`, 10);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!integrationRegistry.has(id)) {
      return NextResponse.json({ error: 'Unknown integration' }, { status: 404 });
    }

    await removeCredentials(userId, id as IntegrationId);

    return NextResponse.json({ success: true });
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `DELETE /api/integrations/${(await params).id}/disconnect`);
  }
}
