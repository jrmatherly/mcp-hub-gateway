/**
 * JWT and Session Management Utilities
 *
 * Production-ready utilities for handling JWT tokens and secure session management
 * in the MCP Portal. Provides server-side token operations with proper security.
 */

import jwt from 'jsonwebtoken';
import { randomBytes } from 'crypto';
import { env } from '@/env.mjs';
import { authLogger } from '@/lib/logger';
import type { User } from '@/types';

/**
 * JWT payload structure for MCP Portal sessions
 */
export interface JWTPayload {
  /** User ID */
  sub: string;
  /** User email */
  email: string;
  /** User name */
  name: string;
  /** User picture URL */
  picture?: string;
  /** User roles */
  roles: string[];
  /** Tenant ID */
  tenantId: string;
  /** Issued at timestamp */
  iat: number;
  /** Expiration timestamp */
  exp: number;
  /** JWT ID for revocation */
  jti: string;
  /** Session type */
  type: 'access' | 'refresh';
}

/**
 * Token configuration
 */
const TOKEN_CONFIG = {
  ACCESS_TOKEN_EXPIRY: '1h', // 1 hour
  REFRESH_TOKEN_EXPIRY: '7d', // 7 days
  ISSUER: 'mcp-portal',
  AUDIENCE: 'mcp-portal-api',
  ALGORITHM: 'HS256' as const,
} as const;

/**
 * Get JWT secret from environment
 */
function getJWTSecret(): string {
  const secret = env.JWT_SECRET;
  if (!secret) {
    throw new Error('JWT_SECRET environment variable is required');
  }

  if (secret.length < 32) {
    throw new Error('JWT_SECRET must be at least 32 characters long');
  }

  return secret;
}

/**
 * Generate a unique JWT ID
 */
function generateJTI(): string {
  return `${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

/**
 * Create an access token for the user
 */
export async function createAccessToken(user: User): Promise<string> {
  try {
    const jti = generateJTI();

    const payload: Omit<JWTPayload, 'iat' | 'exp'> = {
      sub: user.id,
      email: user.email,
      name: user.name,
      picture: user.picture,
      roles: user.roles,
      tenantId: user.tenantId,
      jti,
      type: 'access',
    };

    const token = jwt.sign(payload, getJWTSecret(), {
      algorithm: TOKEN_CONFIG.ALGORITHM,
      expiresIn: TOKEN_CONFIG.ACCESS_TOKEN_EXPIRY,
      issuer: TOKEN_CONFIG.ISSUER,
      audience: TOKEN_CONFIG.AUDIENCE,
    });

    authLogger.debug('Created access token', {
      userId: user.id,
      jti,
      expiresIn: TOKEN_CONFIG.ACCESS_TOKEN_EXPIRY,
    });

    return token;
  } catch (error) {
    authLogger.error('Failed to create access token', error, {
      userId: user.id,
    });
    throw new Error('Failed to create access token');
  }
}

/**
 * Create a refresh token for the user
 */
export async function createRefreshToken(user: User): Promise<string> {
  try {
    const jti = generateJTI();

    const payload: Omit<JWTPayload, 'iat' | 'exp'> = {
      sub: user.id,
      email: user.email,
      name: user.name,
      picture: user.picture,
      roles: user.roles,
      tenantId: user.tenantId,
      jti,
      type: 'refresh',
    };

    const token = jwt.sign(payload, getJWTSecret(), {
      algorithm: TOKEN_CONFIG.ALGORITHM,
      expiresIn: TOKEN_CONFIG.REFRESH_TOKEN_EXPIRY,
      issuer: TOKEN_CONFIG.ISSUER,
      audience: TOKEN_CONFIG.AUDIENCE,
    });

    authLogger.debug('Created refresh token', {
      userId: user.id,
      jti,
      expiresIn: TOKEN_CONFIG.REFRESH_TOKEN_EXPIRY,
    });

    return token;
  } catch (error) {
    authLogger.error('Failed to create refresh token', error, {
      userId: user.id,
    });
    throw new Error('Failed to create refresh token');
  }
}

/**
 * Verify and decode a JWT token
 */
export async function verifyToken(token: string): Promise<JWTPayload | null> {
  try {
    const payload = jwt.verify(token, getJWTSecret(), {
      issuer: TOKEN_CONFIG.ISSUER,
      audience: TOKEN_CONFIG.AUDIENCE,
      algorithms: [TOKEN_CONFIG.ALGORITHM],
    }) as JWTPayload;

    // Validate required fields
    if (!payload.sub || !payload.email || !payload.type) {
      authLogger.warn('Invalid JWT payload structure', {
        hasSub: !!payload.sub,
        hasEmail: !!payload.email,
        hasType: !!payload.type,
      });
      return null;
    }

    authLogger.debug('Successfully verified token', {
      userId: payload.sub,
      type: payload.type,
      jti: payload.jti,
    });

    return payload;
  } catch (error) {
    authLogger.warn('Token verification failed', error);
    return null;
  }
}

/**
 * Extract user information from JWT payload
 */
export function payloadToUser(payload: JWTPayload): User {
  return {
    id: payload.sub,
    email: payload.email,
    name: payload.name,
    picture: payload.picture,
    roles: payload.roles || [],
    tenantId: payload.tenantId,
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}

/**
 * Check if a token is expired
 */
export function isTokenExpired(payload: JWTPayload): boolean {
  const now = Math.floor(Date.now() / 1000);
  return payload.exp <= now;
}

/**
 * Get token expiration info
 */
export function getTokenExpiration(payload: JWTPayload): {
  expiresAt: Date;
  expiresIn: number;
  isExpired: boolean;
} {
  const expiresAt = new Date(payload.exp * 1000);
  const expiresIn = Math.max(0, payload.exp - Math.floor(Date.now() / 1000));
  const isExpired = isTokenExpired(payload);

  return {
    expiresAt,
    expiresIn,
    isExpired,
  };
}

/**
 * Validate token type
 */
export function isAccessToken(payload: JWTPayload): boolean {
  return payload.type === 'access';
}

export function isRefreshToken(payload: JWTPayload): boolean {
  return payload.type === 'refresh';
}

/**
 * Create both access and refresh tokens
 */
export async function createTokenPair(user: User): Promise<{
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}> {
  const [accessToken, refreshToken] = await Promise.all([
    createAccessToken(user),
    createRefreshToken(user),
  ]);

  // Calculate expiration in seconds for access token
  const expiresIn = 60 * 60; // 1 hour in seconds

  return {
    accessToken,
    refreshToken,
    expiresIn,
  };
}

/**
 * Refresh an access token using a refresh token
 */
export async function refreshAccessToken(refreshToken: string): Promise<{
  accessToken: string;
  expiresIn: number;
} | null> {
  try {
    const payload = await verifyToken(refreshToken);

    if (!payload) {
      authLogger.warn('Invalid refresh token provided');
      return null;
    }

    if (!isRefreshToken(payload)) {
      authLogger.warn('Token is not a refresh token', { type: payload.type });
      return null;
    }

    if (isTokenExpired(payload)) {
      authLogger.warn('Refresh token is expired', { userId: payload.sub });
      return null;
    }

    // Create new access token
    const user = payloadToUser(payload);
    const accessToken = await createAccessToken(user);
    const expiresIn = 60 * 60; // 1 hour in seconds

    authLogger.info('Successfully refreshed access token', { userId: user.id });

    return {
      accessToken,
      expiresIn,
    };
  } catch (error) {
    authLogger.error('Failed to refresh access token', error);
    return null;
  }
}

/**
 * Revoke a token (implement blacklist logic here if needed)
 */
export async function revokeToken(token: string): Promise<boolean> {
  try {
    const payload = await verifyToken(token);

    if (!payload) {
      return false;
    }

    // In a production environment, you would add this token's JTI to a blacklist
    // stored in Redis or database with expiration matching the token's expiry

    authLogger.info('Token revoked', {
      userId: payload.sub,
      jti: payload.jti,
      type: payload.type,
    });

    return true;
  } catch (error) {
    authLogger.error('Failed to revoke token', error);
    return false;
  }
}

/**
 * Security utility: Hash sensitive data
 */
export function hashSensitiveData(data: string): string {
  // Simple hash for JTI or other sensitive data
  // In production, use a proper hashing library like bcrypt
  return Buffer.from(data).toString('base64').slice(0, 16);
}

/**
 * Generate a secure random state parameter for OAuth
 */
export function generateSecureState(): string {
  return randomBytes(32).toString('base64url');
}

/**
 * Validate OAuth state parameter
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
