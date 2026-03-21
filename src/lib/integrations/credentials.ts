import { db } from '@/lib/db';
import { integrationConnections } from '@/lib/db/schema';
import { eq, and } from 'drizzle-orm';
import { decryptApiKey, encryptApiKey } from '@/lib/crypto';
import { integrationRegistry } from './registry';
import type { IntegrationId, IntegrationCredentials } from '@/types/integration';

// Prevents concurrent token refreshes for the same integration+user
const refreshLocks = new Map<string, Promise<IntegrationCredentials>>();

export async function getValidCredentials(
  userId: string,
  integrationId: IntegrationId
): Promise<IntegrationCredentials | null> {
  const [connection] = await db
    .select()
    .from(integrationConnections)
    .where(
      and(
        eq(integrationConnections.userId, userId),
        eq(integrationConnections.integrationId, integrationId),
        eq(integrationConnections.isActive, true)
      )
    );

  if (!connection) return null;

  const accessToken = decryptApiKey(
    connection.encryptedAccessToken,
    connection.iv,
    connection.authTag,
    userId
  );

  let refreshTokenValue: string | undefined;
  if (connection.encryptedRefreshToken) {
    refreshTokenValue = decryptApiKey(
      connection.encryptedRefreshToken,
      connection.iv,
      connection.authTag,
      userId
    );
  }

  const credentials: IntegrationCredentials = {
    accessToken,
    refreshToken: refreshTokenValue,
    tokenExpiresAt: connection.tokenExpiresAt || undefined,
    metadata: (connection.metadata as Record<string, unknown>) || undefined,
  };

  // Auto-refresh if token expires within 5 minutes
  if (credentials.tokenExpiresAt && credentials.refreshToken) {
    const fiveMinutes = 5 * 60 * 1000;
    if (credentials.tokenExpiresAt.getTime() - Date.now() < fiveMinutes) {
      const lockKey = `${integrationId}:${userId}`;
      const existing = refreshLocks.get(lockKey);
      if (existing) {
        // Another caller is already refreshing — await same promise
        try {
          return await existing;
        } catch {
          // If the other refresh failed, return existing credentials
          return credentials;
        }
      }

      const refreshPromise = (async () => {
        try {
          const integration = integrationRegistry.get(integrationId);
          const refreshed = await integration.refreshToken(credentials);
          await storeCredentials(userId, integrationId, refreshed);
          return refreshed;
        } finally {
          refreshLocks.delete(lockKey);
        }
      })();

      refreshLocks.set(lockKey, refreshPromise);

      try {
        return await refreshPromise;
      } catch {
        // Return existing credentials if refresh fails
      }
    }
  }

  return credentials;
}

export async function storeCredentials(
  userId: string,
  integrationId: IntegrationId,
  credentials: IntegrationCredentials
): Promise<void> {
  const { encryptedKey: encryptedAccessToken, iv, authTag } = encryptApiKey(
    credentials.accessToken,
    userId
  );

  let encryptedRefreshToken: string | null = null;
  if (credentials.refreshToken) {
    const result = encryptApiKey(credentials.refreshToken, userId);
    encryptedRefreshToken = result.encryptedKey;
  }

  // Upsert: delete existing then insert (atomic)
  await db.transaction(async (tx) => {
    await tx
      .delete(integrationConnections)
      .where(
        and(
          eq(integrationConnections.userId, userId),
          eq(integrationConnections.integrationId, integrationId)
        )
      );

    await tx.insert(integrationConnections).values({
      userId,
      integrationId,
      encryptedAccessToken,
      encryptedRefreshToken,
      iv,
      authTag,
      tokenExpiresAt: credentials.tokenExpiresAt || null,
      metadata: credentials.metadata || {},
      isActive: true,
    });
  });
}

export async function removeCredentials(
  userId: string,
  integrationId: IntegrationId
): Promise<void> {
  await db
    .delete(integrationConnections)
    .where(
      and(
        eq(integrationConnections.userId, userId),
        eq(integrationConnections.integrationId, integrationId)
      )
    );
}
