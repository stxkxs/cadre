export type CadreEnv = 'local' | 'dev' | 'staging' | 'prod';

interface DbConfig {
  url: string;
  poolSize: number;
}

interface AuthConfig {
  secret: string;
  url: string;
  password: string;
}

interface AppConfig {
  env: CadreEnv;
  logLevel: string;
}

interface Config {
  db: DbConfig;
  auth: AuthConfig;
  app: AppConfig;
}

let _config: Config | null = null;

function requiredVar(name: string, env: CadreEnv): string {
  const value = process.env[name];
  if (!value) {
    if (env === 'prod' || env === 'staging') {
      throw new Error(`Missing required environment variable: ${name}`);
    }
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
      url: optionalVar('NEXTAUTH_URL', 'http://localhost:3000'),
      password: requiredVar('AUTH_PASSWORD', env),
    },
    app: {
      env,
      logLevel: optionalVar('LOG_LEVEL', env === 'prod' ? 'warn' : 'debug'),
    },
  };

  return _config;
}

export function isProd(): boolean {
  return getConfig().app.env === 'prod';
}
