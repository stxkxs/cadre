import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('config', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    vi.resetModules();
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  async function getConfigModule() {
    return await import('../config');
  }

  it('returns config object with expected shape', async () => {
    process.env.CADRE_ENV = 'local';
    const { getConfig } = await getConfigModule();
    const config = getConfig();

    expect(config.db).toBeDefined();
    expect(config.auth).toBeDefined();
    expect(config.aws).toBeDefined();
    expect(config.integrations).toBeDefined();
    expect(config.app).toBeDefined();
  });

  it('uses singleton caching', async () => {
    process.env.CADRE_ENV = 'local';
    const { getConfig } = await getConfigModule();
    const first = getConfig();
    const second = getConfig();
    expect(first).toBe(second);
  });

  it('reads CADRE_ENV correctly', async () => {
    process.env.CADRE_ENV = 'dev';
    const { getConfig } = await getConfigModule();
    expect(getConfig().app.env).toBe('dev');
  });

  it('defaults CADRE_ENV to local', async () => {
    delete process.env.CADRE_ENV;
    const { getConfig } = await getConfigModule();
    expect(getConfig().app.env).toBe('local');
  });

  it('requiredVar throws in prod when missing', async () => {
    process.env.CADRE_ENV = 'prod';
    delete process.env.DATABASE_URL;
    const { getConfig } = await getConfigModule();
    expect(() => getConfig()).toThrow('Missing required environment variable: DATABASE_URL');
  });

  it('requiredVar returns empty in local when missing', async () => {
    process.env.CADRE_ENV = 'local';
    delete process.env.DATABASE_URL;
    const { getConfig } = await getConfigModule();
    expect(getConfig().db.url).toBe('');
  });

  it('optionalVar uses fallback', async () => {
    process.env.CADRE_ENV = 'local';
    delete process.env.AWS_REGION;
    const { getConfig } = await getConfigModule();
    expect(getConfig().aws.region).toBe('us-east-1');
  });

  it('optionalVar uses env value when set', async () => {
    process.env.CADRE_ENV = 'local';
    process.env.AWS_REGION = 'eu-west-1';
    const { getConfig } = await getConfigModule();
    expect(getConfig().aws.region).toBe('eu-west-1');
  });

  it('isProd returns true for prod', async () => {
    process.env.CADRE_ENV = 'prod';
    process.env.DATABASE_URL = 'postgres://test';
    process.env.AUTH_SECRET = 'test';
    process.env.AUTH_GITHUB_ID = 'test';
    process.env.AUTH_GITHUB_SECRET = 'test';
    process.env.ENCRYPTION_SECRET = 'test';
    const { isProd } = await getConfigModule();
    expect(isProd()).toBe(true);
  });

  it('isProd returns false for local', async () => {
    process.env.CADRE_ENV = 'local';
    const { isProd } = await getConfigModule();
    expect(isProd()).toBe(false);
  });

  it('LOG_LEVEL defaults to debug in non-prod', async () => {
    process.env.CADRE_ENV = 'local';
    delete process.env.LOG_LEVEL;
    const { getConfig } = await getConfigModule();
    expect(getConfig().app.logLevel).toBe('debug');
  });

  it('LOG_LEVEL defaults to warn in prod', async () => {
    process.env.CADRE_ENV = 'prod';
    process.env.DATABASE_URL = 'postgres://test';
    process.env.AUTH_SECRET = 'test';
    process.env.AUTH_GITHUB_ID = 'test';
    process.env.AUTH_GITHUB_SECRET = 'test';
    process.env.ENCRYPTION_SECRET = 'test';
    delete process.env.LOG_LEVEL;
    const { getConfig } = await getConfigModule();
    expect(getConfig().app.logLevel).toBe('warn');
  });

  it('DB_POOL_SIZE defaults to 10', async () => {
    process.env.CADRE_ENV = 'local';
    delete process.env.DB_POOL_SIZE;
    const { getConfig } = await getConfigModule();
    expect(getConfig().db.poolSize).toBe(10);
  });
});
