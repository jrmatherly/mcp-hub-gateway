/**
 * Edge Runtime Compatible JWT Utilities
 *
 * This file contains JWT utilities that work in Edge Runtime (middleware).
 * Uses Web Crypto API instead of Node.js crypto module.
 */

import { env } from '@/env.mjs';

/**
 * JWT Payload interface
 */
export interface JWTPayload {
  sub: string;
  email: string;
  name?: string;
  exp?: number;
  iat?: number;
  roles?: string[];
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
 * Decode JWT without verification (Edge Runtime compatible)
 * WARNING: This does NOT verify the token signature!
 * Only use for reading claims in middleware where full verification isn't needed.
 */
export function decodeTokenUnsafe(token: string): JWTPayload | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    // Decode payload (middle part)
    const payload = parts[1];
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded) as JWTPayload;
  } catch {
    return null;
  }
}

/**
 * Get token expiration info (Edge Runtime compatible)
 */
export function getTokenExpiration(payload: JWTPayload | null): TokenExpiration {
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
 * Check if token is expired (Edge Runtime compatible)
 */
export function isTokenExpired(token: string): boolean {
  const payload = decodeTokenUnsafe(token);
  const expiration = getTokenExpiration(payload);
  return expiration.isExpired;
}

/**
 * Generate secure random state using Web Crypto API (Edge Runtime compatible)
 */
export async function generateSecureState(): Promise<string> {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);

  // Convert to base64url
  const base64 = btoa(String.fromCharCode(...array));
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

/**
 * Validate OAuth state parameter (Edge Runtime compatible)
 */
export function validateOAuthState(received: string, stored: string): boolean {
  if (!received || !stored) {
    return false;
  }

  // Constant-time comparison to prevent timing attacks
  if (received.length !== stored.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < received.length; i++) {
    result |= received.charCodeAt(i) ^ stored.charCodeAt(i);
  }

  return result === 0;
}

/**
 * Extract Bearer token from Authorization header (Edge Runtime compatible)
 */
export function extractBearerToken(authHeader: string | null): string | null {
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }
  return authHeader.substring(7);
}

/**
 * Simple session ID generator for Edge Runtime
 */
export async function generateSessionId(): Promise<string> {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);

  // Convert to hex string
  return Array.from(array)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}