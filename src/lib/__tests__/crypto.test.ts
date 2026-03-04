import { describe, it, expect } from 'vitest';
import { encryptApiKey, decryptApiKey } from '../crypto';

describe('crypto', () => {
  it('round-trips encrypt and decrypt', () => {
    const apiKey = 'sk-test-1234567890abcdef';
    const userId = 'user-1';
    const { encryptedKey, iv, authTag } = encryptApiKey(apiKey, userId);
    const decrypted = decryptApiKey(encryptedKey, iv, authTag, userId);
    expect(decrypted).toBe(apiKey);
  });

  it('produces different ciphertext for different users', () => {
    const apiKey = 'sk-test-same-key';
    const result1 = encryptApiKey(apiKey, 'user-1');
    const result2 = encryptApiKey(apiKey, 'user-2');
    expect(result1.encryptedKey).not.toBe(result2.encryptedKey);
  });

  it('detects auth tag tampering', () => {
    const { encryptedKey, iv, authTag } = encryptApiKey('secret-key', 'user-1');
    // Tamper with the auth tag
    const tamperedTag = authTag.slice(0, -2) + 'ff';
    expect(() => decryptApiKey(encryptedKey, iv, tamperedTag, 'user-1')).toThrow();
  });

  it('uses random IVs (different for each encryption)', () => {
    const apiKey = 'sk-test-key';
    const userId = 'user-1';
    const result1 = encryptApiKey(apiKey, userId);
    const result2 = encryptApiKey(apiKey, userId);
    expect(result1.iv).not.toBe(result2.iv);
  });

  it('fails to decrypt with wrong user', () => {
    const { encryptedKey, iv, authTag } = encryptApiKey('secret', 'user-1');
    expect(() => decryptApiKey(encryptedKey, iv, authTag, 'user-2')).toThrow();
  });

  it('handles empty string', () => {
    const { encryptedKey, iv, authTag } = encryptApiKey('', 'user-1');
    const decrypted = decryptApiKey(encryptedKey, iv, authTag, 'user-1');
    expect(decrypted).toBe('');
  });

  it('handles long API keys', () => {
    const longKey = 'sk-' + 'a'.repeat(500);
    const { encryptedKey, iv, authTag } = encryptApiKey(longKey, 'user-1');
    const decrypted = decryptApiKey(encryptedKey, iv, authTag, 'user-1');
    expect(decrypted).toBe(longKey);
  });
});
