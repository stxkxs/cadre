export type CadreEnv = 'local' | 'dev' | 'staging' | 'prod';

interface DbConfig {
  url: string;
  poolSize: number;
}

interface AuthConfig {
  secret: string;
  githubId: string;
  githubSecret: string;
  url: string;
}

interface AwsConfig {
  region: string;
  s3Bucket: string;
}

interface IntegrationsConfig {
  webhookBaseUrl: string;
  webhookSecret: string;
}

interface AppConfig {
  env: CadreEnv;
  logLevel: string;
  encryptionSecret: string;
}

interface Config {
  db: DbConfig;
  auth: AuthConfig;
  aws: AwsConfig;
  integrations: IntegrationsConfig;
  app: AppConfig;
}

let _config: Config | null = null;

function requiredVar(name: string, env: CadreEnv): string {
  const value = process.env[name];
  if (!value) {
    if (env === 'prod' || env === 'staging') {
      throw new Error(`Missing required environment variable: ${name}`);
    }
    // Dynamic import avoided — logger depends on env vars that config is still reading.
    // Use console.warn here since logger.warn would create a circular dependency.
    console.warn(`[config] Missing env var ${name} (non-critical in ${env})`);
    return '';
  }
  return value;
}

function optionalVar(name: string, fallback: string): string {
  return process.env[name] || fallback;
}

export function getConfig(): Config {
  if (_config) return _config;

  const env = (process.env.CADRE_ENV || 'local') as CadreEnv;

  _config = {
    db: {
      url: requiredVar('DATABASE_URL', env),
      poolSize: parseInt(optionalVar('DB_POOL_SIZE', '10'), 10),
    },
    auth: {
      secret: requiredVar('AUTH_SECRET', env),
      githubId: requiredVar('AUTH_GITHUB_ID', env),
      githubSecret: requiredVar('AUTH_GITHUB_SECRET', env),
      url: optionalVar('NEXTAUTH_URL', 'http://localhost:3000'),
    },
    aws: {
      region: optionalVar('AWS_REGION', 'us-east-1'),
      s3Bucket: optionalVar('S3_BUCKET', 'cadre-artifacts'),
    },
    integrations: {
      webhookBaseUrl: optionalVar('WEBHOOK_BASE_URL', 'http://localhost:3000/api/webhooks'),
      webhookSecret: optionalVar('WEBHOOK_SECRET', ''),
    },
    app: {
      env,
      logLevel: optionalVar('LOG_LEVEL', env === 'prod' ? 'warn' : 'debug'),
      encryptionSecret: requiredVar('ENCRYPTION_SECRET', env),
    },
  };

  return _config;
}

export function isProd(): boolean {
  return getConfig().app.env === 'prod';
}
