// This will be placed in tests/unit/config.test.ts
import { loadConfig } from '@/config/loader.js';

describe('Config Loader', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('should load default configuration', () => {
    process.env.AI_PROVIDER = 'claude';
    const config = loadConfig();

    expect(config.aiProvider).toBe('claude');
    expect(config.gatewayPort).toBe(18789);
    expect(config.gatewayHost).toBe('localhost');
    expect(config.logLevel).toBe('info');
  });

  it('should override defaults with environment variables', () => {
    process.env.GATEWAY_PORT = '9000';
    process.env.LOG_LEVEL = 'debug';
    const config = loadConfig();

    expect(config.gatewayPort).toBe(9000);
    expect(config.logLevel).toBe('debug');
  });

  it('should parse tool whitelist', () => {
    process.env.TOOL_WHITELIST = 'nmap,curl,dig,whois';
    const config = loadConfig();

    expect(config.toolWhitelist).toContain('nmap');
    expect(config.toolWhitelist).toContain('curl');
    expect(config.toolWhitelist).toHaveLength(4);
  });

  it('should set telegram configuration', () => {
    process.env.ENABLE_TELEGRAM = 'true';
    process.env.TELEGRAM_TOKEN = 'token123';
    process.env.TELEGRAM_CHAT_ID = 'chat123';
    const config = loadConfig();

    expect(config.enableTelegram).toBe(true);
    expect(config.telegramToken).toBe('token123');
    expect(config.telegramChatId).toBe('chat123');
  });

  it('should handle invalid port gracefully', () => {
    process.env.GATEWAY_PORT = 'invalid';
    expect(() => loadConfig()).toThrow();
  });
});
