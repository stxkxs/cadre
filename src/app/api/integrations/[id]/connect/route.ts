import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { integrationRegistry } from '@/lib/integrations/registry';
import type { IntegrationId } from '@/types/integration';
import { randomUUID } from 'crypto';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    const rl = rateLimit(`connect:${userId}`, 10);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    if (!integrationRegistry.has(id)) {
      return NextResponse.json({ error: 'Unknown integration' }, { status: 404 });
    }

    const integration = integrationRegistry.get(id as IntegrationId);
    const state = randomUUID();
    const baseUrl = process.env.NEXTAUTH_URL || 'http://localhost:3000';
    const redirectUri = `${baseUrl}/api/integrations/${id}/callback`;

    const authUrl = integration.getAuthorizationUrl(state, redirectUri);

    // Store state in a short-lived cookie for CSRF protection
    const response = NextResponse.redirect(authUrl);
    response.cookies.set(`oauth_state_${id}`, state, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 600, // 10 minutes
      path: '/',
    });

    return response;
  } catch (error) {
    const { handleApiError } = await import('@/lib/api-error');
    return handleApiError(error, `GET /api/integrations/${(await params).id}/connect`);
  }
}
