import { NextResponse } from 'next/server';
import { randomUUID } from 'crypto';

export function apiError(message: string, status: number, details?: unknown) {
  const requestId = randomUUID().slice(0, 8);
  const body: Record<string, unknown> = {
    error: message,
    status,
    requestId,
  };
  if (details !== undefined) {
    body.details = details;
  }
  if (status >= 500) {
    console.error(`[${requestId}] ${message}`, details);
  }
  return NextResponse.json(body, { status });
}

export function handleApiError(error: unknown, context: string): NextResponse {
  if (error instanceof Error && error.message === 'Unauthorized') {
    return apiError('Unauthorized', 401);
  }
  const message = error instanceof Error ? error.message : 'Unknown error';
  console.error(`[${context}]`, message);
  return apiError('Internal server error', 500);
}
