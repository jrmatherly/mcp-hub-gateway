/**
 * Edge Runtime Compatible JWT Utilities using jose
 *
 * This file contains JWT utilities that work in Next.js Edge Runtime (middleware).
 * Based on official Next.js documentation and jose library best practices.
 *
 * @see https://github.com/panva/jose
 * @see https://nextjs.org/docs/app/guides/authentication
 */

import { jwtVerify, SignJWT, type JWTPayload } from 'jose';
import { env } from '@/env.mjs';

/**
 * Extended JWT Payload interface for our application
 */
export interface AuthPayload extends JWTPayload {
  sub: string;
  email: string;
  name?: string;
  roles?: string[];
  tenantId?: string;
}

/**
 * Token expiration info
 */
export interface TokenExpiration {
  isExpired: boolean;
  expiresAt: Date | null;
  expiresIn: number;
}

/**
 * Get the secret key for JWT operations
 * Uses TextEncoder to convert string to Uint8Array for jose
 */
function getSecretKey(): Uint8Array {
  const secret = env.JWT_SECRET || 'development-secret-change-in-production';
  return new TextEncoder().encode(secret);
}

/**
 * Verify JWT token with full signature validation (Edge Runtime compatible)
 * This performs complete JWT verification including signature validation
 */
export async function verifyToken(token: string): Promise<AuthPayload | null> {
  try {
    const { payload } = await jwtVerify(token, getSecretKey(), {
      algorithms: ['HS256'],
    });

    // Type assertion since we know our token structure
    return payload as AuthPayload;
  } catch (error) {
    // Token verification failed (expired, invalid signature, etc.)
    console.error('JWT verification failed:', error);
    return null;
  }
}

/**
 * Decode JWT without verification (for reading claims only)
 * WARNING: This does NOT verify the token signature!
 * Use only when you need to read claims before full verification.
 */
export function decodeTokenUnsafe(token: string): AuthPayload | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    // Decode payload (middle part)
    const payload = parts[1];
    // Handle URL-safe base64
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded) as AuthPayload;
  } catch {
    return null;
  }
}

/**
 * Get token expiration info from payload
 */
export function getTokenExpiration(payload: AuthPayload | null): TokenExpiration {
  if (!payload || !payload.exp) {
    return {
      isExpired: true,
      expiresAt: null,
      expiresIn: 0,
    };
  }

  const expiresAt = new Date(payload.exp * 1000);
  const now = new Date();
  const expiresIn = Math.floor((expiresAt.getTime() - now.getTime()) / 1000);

  return {
    isExpired: expiresIn <= 0,
    expiresAt,
    expiresIn: Math.max(0, expiresIn),
  };
}

/**
 * Check if token needs refresh (expires in less than 5 minutes)
 */
export function needsTokenRefresh(payload: AuthPayload | null): boolean {
  const expiration = getTokenExpiration(payload);
  return expiration.isExpired || expiration.expiresIn < 300; // 5 minutes
}

/**
 * Create a new JWT token (Edge Runtime compatible)
 * Used for creating session tokens after OAuth authentication
 */
export async function createToken(payload: AuthPayload): Promise<string> {
  const iat = Math.floor(Date.now() / 1000);
  const exp = iat + 60 * 60 * 24; // 24 hours

  return new SignJWT(payload)
    .setProtectedHeader({ alg: 'HS256', typ: 'JWT' })
    .setExpirationTime(exp)
    .setIssuedAt(iat)
    .setNotBefore(iat)
    .sign(getSecretKey());
}

/**
 * Extract Bearer token from Authorization header
 */
export function extractBearerToken(authHeader: string | null): string | null {
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }
  return authHeader.substring(7);
}

/**
 * Validate OAuth state parameter (constant-time comparison)
 */
export function validateOAuthState(received: string, stored: string): boolean {
  if (!received || !stored || received.length !== stored.length) {
    return false;
  }

  // Constant-time comparison to prevent timing attacks
  let result = 0;
  for (let i = 0; i < received.length; i++) {
    result |= received.charCodeAt(i) ^ stored.charCodeAt(i);
  }

  return result === 0;
}

/**
 * Generate secure random state using Web Crypto API
 * For OAuth state parameter generation
 */
export async function generateSecureState(): Promise<string> {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);

  // Convert to base64url
  const base64 = btoa(String.fromCharCode(...array));
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

/**
 * Generate session ID for tracking
 */
export async function generateSessionId(): Promise<string> {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);

  // Convert to hex string
  return Array.from(array)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}