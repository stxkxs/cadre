import { createCipheriv, createDecipheriv, randomBytes, createHash } from 'crypto';

const ALGORITHM = 'aes-256-gcm';
const IV_LENGTH = 16;

function deriveKey(userId: string): Buffer {
  const serverSecret = process.env.ENCRYPTION_SECRET || 'dev-secret-change-in-production';
  return createHash('sha256')
    .update(`${serverSecret}:${userId}`)
    .digest();
}

export function encryptApiKey(
  apiKey: string,
  userId: string
): { encryptedKey: string; iv: string; authTag: string } {
  const key = deriveKey(userId);
  const iv = randomBytes(IV_LENGTH);
  const cipher = createCipheriv(ALGORITHM, key, iv);

  let encrypted = cipher.update(apiKey, 'utf8', 'hex');
  encrypted += cipher.final('hex');
  const authTag = cipher.getAuthTag();

  return {
    encryptedKey: encrypted,
    iv: iv.toString('hex'),
    authTag: authTag.toString('hex'),
  };
}

export function decryptApiKey(
  encryptedKey: string,
  iv: string,
  authTag: string,
  userId: string
): string {
  const key = deriveKey(userId);
  const decipher = createDecipheriv(ALGORITHM, key, Buffer.from(iv, 'hex'));
  decipher.setAuthTag(Buffer.from(authTag, 'hex'));

  let decrypted = decipher.update(encryptedKey, 'hex', 'utf8');
  decrypted += decipher.final('utf8');
  return decrypted;
}
