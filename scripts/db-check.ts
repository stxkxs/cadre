#!/usr/bin/env tsx
/**
 * Checks database connectivity. Run before deployment to verify DB access.
 * Usage: npx tsx scripts/db-check.ts
 */

import postgres from 'postgres';

async function main() {
  const url = process.env.DATABASE_URL;
  if (!url) {
    console.error('DATABASE_URL is not set');
    process.exit(1);
  }

  console.log('Checking database connection...');
  const sql = postgres(url, { prepare: false, connect_timeout: 10 });

  try {
    const result = await sql`SELECT 1 as ok`;
    if (result[0]?.ok === 1) {
      console.log('Database connection successful');
    }

    // Check if tables exist
    const tables = await sql`
      SELECT table_name
      FROM information_schema.tables
      WHERE table_schema = 'public'
      ORDER BY table_name
    `;

    if (tables.length === 0) {
      console.warn('No tables found. Run: pnpm db:push');
    } else {
      console.log(`Found ${tables.length} tables: ${tables.map(t => t.table_name).join(', ')}`);
    }
  } catch (err) {
    console.error('Database connection failed:', err instanceof Error ? err.message : err);
    process.exit(1);
  } finally {
    await sql.end();
  }
}

main();
