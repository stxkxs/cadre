import { NextRequest, NextResponse } from 'next/server';
import { providerRegistry } from '@/lib/providers/registry';
import { db } from '@/lib/db';
import { userApiKeys } from '@/lib/db/schema';
import { encryptApiKey } from '@/lib/crypto';
import { eq, and } from 'drizzle-orm';
import type { ModelProvider } from '@/lib/engine/types';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';

export async function POST(request: NextRequest) {
  try {
    const userId = await getAuthUserId();

    // Rate limit: 5 key tests per minute per user
    const rl = rateLimit(`model-test:${userId}`, 5);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests. Please wait.' }, { status: 429 });
    }
    const body = await request.json();
    const { provider, apiKey } = body as { provider: ModelProvider; apiKey: string };

    if (!provider) {
      return NextResponse.json(
        { error: 'Provider is required' },
        { status: 400 }
      );
    }

    if (!providerRegistry.has(provider)) {
      return NextResponse.json(
        { error: `Unknown provider: ${provider}. Valid providers: ${providerRegistry.getIds().join(', ')}` },
        { status: 400 }
      );
    }

    // Claude Code uses CLI auth, not an API key
    if (provider === 'claude-code') {
      const providerInstance = providerRegistry.get(provider);
      const isValid = await providerInstance.validateKey('');
      return NextResponse.json({ valid: isValid });
    }

    if (!apiKey || typeof apiKey !== 'string' || apiKey.trim().length < 10) {
      return NextResponse.json(
        { error: 'API key is required and must be at least 10 characters' },
        { status: 400 }
      );
    }

    const providerInstance = providerRegistry.get(provider);
    const isValid = await providerInstance.validateKey(apiKey);

    if (isValid) {
      const encrypted = encryptApiKey(apiKey, userId);

      // Upsert API key
      const existing = await db
        .select()
        .from(userApiKeys)
        .where(and(eq(userApiKeys.userId, userId), eq(userApiKeys.provider, provider)));

      if (existing.length > 0) {
        await db
          .update(userApiKeys)
          .set({
            encryptedKey: encrypted.encryptedKey,
            iv: encrypted.iv,
            authTag: encrypted.authTag,
            isValid: true,
            updatedAt: new Date(),
          })
          .where(eq(userApiKeys.id, existing[0].id));
      } else {
        await db.insert(userApiKeys).values({
          userId,
          provider,
          encryptedKey: encrypted.encryptedKey,
          iv: encrypted.iv,
          authTag: encrypted.authTag,
          isValid: true,
        });
      }
    }

    return NextResponse.json({ valid: isValid });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }
    return NextResponse.json(
      { error: 'Failed to validate API key' },
      { status: 500 }
    );
  }
}
