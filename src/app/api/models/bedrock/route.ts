import { NextResponse } from 'next/server';
import { getAuthUserId } from '@/lib/api-auth';
import { rateLimit } from '@/lib/rate-limit';
import { BedrockProvider } from '@/lib/providers/bedrock';

let cachedModels: { id: string; name: string; provider: string }[] | null = null;
let cacheExpiry = 0;
const CACHE_TTL = 60 * 60 * 1000; // 1 hour

export async function GET() {
  try {
    const userId = await getAuthUserId();

    const rl = rateLimit(`bedrock-models:${userId}`, 5);
    if (!rl.success) {
      return NextResponse.json({ error: 'Too many requests' }, { status: 429 });
    }

    const now = Date.now();
    if (cachedModels && now < cacheExpiry) {
      return NextResponse.json({ models: cachedModels });
    }

    const provider = new BedrockProvider();
    const models = await provider.listModels();

    cachedModels = models;
    cacheExpiry = now + CACHE_TTL;

    return NextResponse.json({ models });
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Failed to list models';
    return NextResponse.json({ error: message, models: [] }, { status: 500 });
  }
}
