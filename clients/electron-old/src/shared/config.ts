// Centralized configuration for the Electron app
// Supports environment variables and build-time configuration

export interface AppConfig {
  backend: {
    url: string;
    timeout: number;
  };
  sync: {
    intervalMs: number;
    retryDelayMs: number;
    maxRetries: number;
  };
  app: {
    name: string;
    version: string;
    environment: 'development' | 'staging' | 'production';
  };
}

// Default configuration
const defaultConfig: AppConfig = {
  backend: {
    url: 'http://localhost:6000',
    timeout: 10000, // 10 seconds
  },
  sync: {
    intervalMs: 5000, // 5 seconds
    retryDelayMs: 1000, // 1 second
    maxRetries: 3,
  },
  app: {
    name: 'Tennex',
    version: '1.0.0',
    environment: 'development',
  },
};

// Environment variable overrides
function getEnvConfig(): Partial<AppConfig> {
  return {
    backend: {
      url: process.env.TENNEX_BACKEND_URL || defaultConfig.backend.url,
      timeout: parseInt(process.env.TENNEX_BACKEND_TIMEOUT || '') || defaultConfig.backend.timeout,
    },
    sync: {
      intervalMs: parseInt(process.env.TENNEX_SYNC_INTERVAL || '') || defaultConfig.sync.intervalMs,
      retryDelayMs: parseInt(process.env.TENNEX_RETRY_DELAY || '') || defaultConfig.sync.retryDelayMs,
      maxRetries: parseInt(process.env.TENNEX_MAX_RETRIES || '') || defaultConfig.sync.maxRetries,
    },
    app: {
      name: process.env.TENNEX_APP_NAME || defaultConfig.app.name,
      version: process.env.npm_package_version || defaultConfig.app.version,
      environment: (process.env.NODE_ENV as AppConfig['app']['environment']) || defaultConfig.app.environment,
    },
  };
}

// Merge default config with environment overrides
function createConfig(): AppConfig {
  const envConfig = getEnvConfig();
  
  return {
    backend: { ...defaultConfig.backend, ...envConfig.backend },
    sync: { ...defaultConfig.sync, ...envConfig.sync },
    app: { ...defaultConfig.app, ...envConfig.app },
  };
}

// Export the final configuration
export const config = createConfig();

// Utility functions for common config access
export const getBackendUrl = () => config.backend.url;
export const getSyncInterval = () => config.sync.intervalMs;
export const isProduction = () => config.app.environment === 'production';
export const isDevelopment = () => config.app.environment === 'development';

// Log configuration on startup (excluding sensitive data)
export function logConfig() {
  console.log('🔧 App Configuration:', {
    backend: {
      url: config.backend.url,
      timeout: config.backend.timeout,
    },
    sync: config.sync,
    app: config.app,
  });
}
