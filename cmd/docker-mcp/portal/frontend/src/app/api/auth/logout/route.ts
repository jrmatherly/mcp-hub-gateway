/**
 * API Route: /api/auth/logout
 *
 * Handles user logout by clearing session cookies and optionally revoking tokens.
 * Provides both local logout (clear cookies) and global logout (Azure AD signout).
 */

import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { env } from '@/env.mjs';
import { authLogger } from '@/lib/logger';

/**
 * POST /api/auth/logout
 *
 * Log out the user by clearing session cookies and optionally revoking tokens
 */
export async function POST(request: NextRequest): Promise<NextResponse> {
  try {
    const cookieStore = await cookies();
    const body = await request.json().catch(() => ({}));
    const { globalLogout = false } = body;

    // Get current tokens for revocation
    const accessToken = cookieStore.get(env.SESSION_COOKIE_NAME)?.value;
    const refreshToken = cookieStore.get(
      `${env.SESSION_COOKIE_NAME}_refresh`
    )?.value;
    const userCookie = cookieStore.get(
      `${env.SESSION_COOKIE_NAME}_user`
    )?.value;

    let userId: string | undefined;
    try {
      if (userCookie) {
        const userData = JSON.parse(userCookie);
        userId = userData.id;
      }
    } catch {
      // Ignore JSON parse errors
    }

    // Revoke tokens if available
    if (accessToken || refreshToken) {
      try {
        const { revokeToken } = await import('@/lib/jwt-utils');

        if (accessToken) {
          await revokeToken(accessToken);
        }
        if (refreshToken) {
          await revokeToken(refreshToken);
        }

        authLogger.info('Tokens revoked during logout', { userId });
      } catch (error) {
        // Log but don't fail logout if revocation fails
        authLogger.warn('Failed to revoke tokens during logout', error);
      }
    }

    // Clear all session cookies
    const response = NextResponse.json({
      success: true,
      message: 'Successfully logged out',
      globalLogout,
    });

    // Cookie clearing options
    const clearCookieOptions = {
      path: '/',
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: env.SESSION_COOKIE_SAMESITE as 'strict' | 'lax' | 'none',
      maxAge: 0, // Immediately expire
    };

    // Clear all session cookies
    response.cookies.set(env.SESSION_COOKIE_NAME, '', clearCookieOptions);
    response.cookies.set(
      `${env.SESSION_COOKIE_NAME}_refresh`,
      '',
      clearCookieOptions
    );
    response.cookies.set(
      `${env.SESSION_COOKIE_NAME}_user`,
      '',
      clearCookieOptions
    );

    // If global logout is requested, include Azure AD logout URL
    if (globalLogout) {
      const tenantId = env.AZURE_TENANT_ID;
      const postLogoutRedirectUri =
        env.NEXT_PUBLIC_AZURE_POST_LOGOUT_URI || request.nextUrl.origin;

      if (tenantId) {
        const azureLogoutUrl = new URL(
          `https://login.microsoftonline.com/${tenantId}/oauth2/v2.0/logout`
        );
        azureLogoutUrl.searchParams.set(
          'post_logout_redirect_uri',
          postLogoutRedirectUri
        );

        response.headers.set('X-Azure-Logout-URL', azureLogoutUrl.toString());
      }
    }

    authLogger.info('User logged out successfully', {
      userId,
      globalLogout,
      userAgent: request.headers.get('user-agent'),
    });

    return response;
  } catch (error) {
    authLogger.error('Unexpected error during logout', error);

    // Even if there's an error, we should still clear cookies
    const response = NextResponse.json(
      {
        success: false,
        error: 'logout_error',
        message:
          'An error occurred during logout, but session has been cleared',
      },
      { status: 500 }
    );

    // Clear cookies anyway
    const clearCookieOptions = {
      path: '/',
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: env.SESSION_COOKIE_SAMESITE as 'strict' | 'lax' | 'none',
      maxAge: 0,
    };

    response.cookies.set(env.SESSION_COOKIE_NAME, '', clearCookieOptions);
    response.cookies.set(
      `${env.SESSION_COOKIE_NAME}_refresh`,
      '',
      clearCookieOptions
    );
    response.cookies.set(
      `${env.SESSION_COOKIE_NAME}_user`,
      '',
      clearCookieOptions
    );

    return response;
  }
}

/**
 * GET /api/auth/logout
 *
 * Alternative logout endpoint that can be called from a GET request
 * (useful for logout links, though POST is preferred)
 */
export async function GET(request: NextRequest): Promise<NextResponse> {
  authLogger.info('Logout requested via GET method');

  // Redirect to login page after clearing cookies
  const loginUrl = new URL('/auth/login', request.nextUrl.origin);
  loginUrl.searchParams.set('message', 'logged_out');

  const response = NextResponse.redirect(loginUrl);

  // Clear cookies
  const clearCookieOptions = {
    path: '/',
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: env.SESSION_COOKIE_SAMESITE as 'strict' | 'lax' | 'none',
    maxAge: 0,
  };

  response.cookies.set(env.SESSION_COOKIE_NAME, '', clearCookieOptions);
  response.cookies.set(
    `${env.SESSION_COOKIE_NAME}_refresh`,
    '',
    clearCookieOptions
  );
  response.cookies.set(
    `${env.SESSION_COOKIE_NAME}_user`,
    '',
    clearCookieOptions
  );

  return response;
}
