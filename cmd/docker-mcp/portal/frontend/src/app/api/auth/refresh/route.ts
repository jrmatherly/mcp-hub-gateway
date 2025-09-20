/**
 * API Route: /api/auth/refresh
 *
 * Handles JWT token refresh using the refresh token stored in HTTP-only cookies.
 * This endpoint allows the frontend to maintain authentication without requiring
 * the user to log in again when the access token expires.
 */

import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { env } from '@/env.mjs';
import { authLogger } from '@/lib/logger';

/**
 * POST /api/auth/refresh
 *
 * Refresh the access token using the refresh token from cookies
 */
export async function POST(_request: NextRequest): Promise<NextResponse> {
  try {
    const cookieStore = await cookies();
    const refreshTokenCookie = cookieStore.get(
      `${env.SESSION_COOKIE_NAME}_refresh`
    );

    if (!refreshTokenCookie?.value) {
      authLogger.warn('Token refresh requested without refresh token cookie');
      return NextResponse.json(
        {
          success: false,
          error: 'refresh_token_missing',
          message: 'Refresh token not found. Please log in again.',
        },
        { status: 401 }
      );
    }

    // Import JWT utilities
    const { refreshAccessToken, verifyToken } = await import('@/lib/jwt-utils');

    // Verify the refresh token first
    const refreshPayload = await verifyToken(refreshTokenCookie.value);
    if (!refreshPayload) {
      authLogger.warn('Invalid refresh token provided');

      // Clear invalid cookies
      const response = NextResponse.json(
        {
          success: false,
          error: 'invalid_refresh_token',
          message: 'Invalid refresh token. Please log in again.',
        },
        { status: 401 }
      );

      response.cookies.delete(env.SESSION_COOKIE_NAME);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_refresh`);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_user`);

      return response;
    }

    // Refresh the access token
    const tokenResult = await refreshAccessToken(refreshTokenCookie.value);

    if (!tokenResult) {
      authLogger.warn('Failed to refresh access token', {
        userId: refreshPayload.sub,
      });

      // Clear cookies on refresh failure
      const response = NextResponse.json(
        {
          success: false,
          error: 'refresh_failed',
          message: 'Failed to refresh token. Please log in again.',
        },
        { status: 401 }
      );

      response.cookies.delete(env.SESSION_COOKIE_NAME);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_refresh`);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_user`);

      return response;
    }

    // Set new access token cookie
    const isProduction = process.env.NODE_ENV === 'production';
    const cookieOptions = {
      httpOnly: env.SESSION_COOKIE_HTTPONLY,
      secure: env.SESSION_COOKIE_SECURE || isProduction,
      sameSite: env.SESSION_COOKIE_SAMESITE as 'strict' | 'lax' | 'none',
      path: '/',
      maxAge: tokenResult.expiresIn,
    };

    const response = NextResponse.json({
      success: true,
      data: {
        expiresIn: tokenResult.expiresIn,
        user: {
          id: refreshPayload.sub,
          email: refreshPayload.email,
          name: refreshPayload.name,
          picture: refreshPayload.picture,
        },
      },
    });

    response.cookies.set(
      env.SESSION_COOKIE_NAME,
      tokenResult.accessToken,
      cookieOptions
    );

    authLogger.info('Successfully refreshed access token', {
      userId: refreshPayload.sub,
      expiresIn: tokenResult.expiresIn,
    });

    return response;
  } catch (error) {
    authLogger.error('Unexpected error during token refresh', error);

    return NextResponse.json(
      {
        success: false,
        error: 'internal_error',
        message: 'An unexpected error occurred during token refresh.',
      },
      { status: 500 }
    );
  }
}

/**
 * GET method not allowed - refresh should use POST
 */
export async function GET(): Promise<NextResponse> {
  return NextResponse.json(
    {
      error: 'method_not_allowed',
      message: 'Token refresh must use POST method',
    },
    { status: 405 }
  );
}
