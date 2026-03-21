import { NextRequest, NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { integrationRegistry } from '@/lib/integrations/registry';
import { storeCredentials } from '@/lib/integrations/credentials';
import { logger } from '@/lib/logger';
import type { IntegrationId } from '@/types/integration';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const userId = await getAuthUserId();

    if (!integrationRegistry.has(id)) {
      return NextResponse.redirect(new URL('/integrations?error=unknown', request.url));
    }

    const searchParams = request.nextUrl.searchParams;
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    const error = searchParams.get('error');

    if (error) {
      return NextResponse.redirect(new URL(`/integrations?error=${error}`, request.url));
    }

    if (!code) {
      return NextResponse.redirect(new URL('/integrations?error=no_code', request.url));
    }

    // Verify state matches cookie
    const storedState = request.cookies.get(`oauth_state_${id}`)?.value;
    if (!storedState || storedState !== state) {
      return NextResponse.redirect(new URL('/integrations?error=invalid_state', request.url));
    }

    const integration = integrationRegistry.get(id as IntegrationId);
    const baseUrl = process.env.NEXTAUTH_URL || 'http://localhost:3000';
    const redirectUri = `${baseUrl}/api/integrations/${id}/callback`;

    const credentials = await integration.exchangeCode(code, redirectUri);
    await storeCredentials(userId, id as IntegrationId, credentials);

    // Clear state cookie
    const response = NextResponse.redirect(new URL('/integrations?connected=' + id, request.url));
    response.cookies.delete(`oauth_state_${id}`);
    return response;
  } catch (error) {
    logger.error('Integration OAuth callback failed', { integrationId: (await params).id, error: String(error) });
    return NextResponse.redirect(new URL('/integrations?error=exchange_failed', request.url));
  }
}
