/**
 * API Route: /api/auth/profile
 *
 * Handles user profile operations including getting and updating user information.
 * Uses JWT tokens from HTTP-only cookies for authentication.
 */

import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { env } from '@/env.mjs';
import { authLogger } from '@/lib/logger';
import type { User } from '@/types';

/**
 * Get authenticated user from cookies
 */
async function getAuthenticatedUser(): Promise<{
  user: User;
  token: string;
} | null> {
  try {
    const cookieStore = await cookies();
    const accessToken = cookieStore.get(env.SESSION_COOKIE_NAME)?.value;

    if (!accessToken) {
      return null;
    }

    const { verifyToken, payloadToUser } = await import('@/lib/jwt-utils');

    const payload = await verifyToken(accessToken);
    if (!payload) {
      return null;
    }

    const user = payloadToUser(payload);
    return { user, token: accessToken };
  } catch (error) {
    authLogger.error('Failed to get authenticated user', error);
    return null;
  }
}

/**
 * GET /api/auth/profile
 *
 * Get the current user's profile information
 */
export async function GET(): Promise<NextResponse> {
  try {
    const authResult = await getAuthenticatedUser();

    if (!authResult) {
      return NextResponse.json(
        {
          success: false,
          error: 'unauthorized',
          message: 'Authentication required',
        },
        { status: 401 }
      );
    }

    const { user } = authResult;

    authLogger.debug('Profile requested', { userId: user.id });

    return NextResponse.json({
      success: true,
      data: {
        id: user.id,
        email: user.email,
        name: user.name,
        picture: user.picture,
        roles: user.roles,
        tenantId: user.tenantId,
        createdAt: user.createdAt,
        updatedAt: user.updatedAt,
      },
    });
  } catch (error) {
    authLogger.error('Unexpected error getting user profile', error);

    return NextResponse.json(
      {
        success: false,
        error: 'internal_error',
        message: 'An unexpected error occurred',
      },
      { status: 500 }
    );
  }
}

/**
 * PUT /api/auth/profile
 *
 * Update the current user's profile information
 */
export async function PUT(request: NextRequest): Promise<NextResponse> {
  try {
    const authResult = await getAuthenticatedUser();

    if (!authResult) {
      return NextResponse.json(
        {
          success: false,
          error: 'unauthorized',
          message: 'Authentication required',
        },
        { status: 401 }
      );
    }

    const { user } = authResult;

    // Parse request body
    let updates: Partial<User>;
    try {
      updates = await request.json();
    } catch {
      return NextResponse.json(
        {
          success: false,
          error: 'invalid_json',
          message: 'Invalid JSON in request body',
        },
        { status: 400 }
      );
    }

    // Validate and sanitize updates
    const allowedFields = ['name', 'picture'] as const;
    const sanitizedUpdates: Partial<
      Pick<User, (typeof allowedFields)[number]>
    > = {};

    for (const field of allowedFields) {
      if (field in updates && updates[field] !== undefined) {
        const value = updates[field];

        // Validate field-specific constraints
        switch (field) {
          case 'name':
            if (
              typeof value === 'string' &&
              value.trim().length > 0 &&
              value.length <= 100
            ) {
              sanitizedUpdates.name = value.trim();
            } else {
              return NextResponse.json(
                {
                  success: false,
                  error: 'invalid_name',
                  message:
                    'Name must be a non-empty string with maximum 100 characters',
                },
                { status: 400 }
              );
            }
            break;

          case 'picture':
            if (
              typeof value === 'string' &&
              (value === '' || isValidUrl(value))
            ) {
              sanitizedUpdates.picture = value || undefined;
            } else {
              return NextResponse.json(
                {
                  success: false,
                  error: 'invalid_picture',
                  message: 'Picture must be a valid URL or empty string',
                },
                { status: 400 }
              );
            }
            break;
        }
      }
    }

    // Check if there are any valid updates
    if (Object.keys(sanitizedUpdates).length === 0) {
      return NextResponse.json(
        {
          success: false,
          error: 'no_updates',
          message: 'No valid updates provided',
        },
        { status: 400 }
      );
    }

    // In a real application, you would update the user in your database here
    // For this implementation, we'll simulate an update and return the updated user

    const updatedUser: User = {
      ...user,
      ...sanitizedUpdates,
      updatedAt: new Date(),
    };

    // If the user data changed significantly, you might want to issue new tokens
    // For this implementation, we'll just return the updated user data

    authLogger.info('User profile updated', {
      userId: user.id,
      updatedFields: Object.keys(sanitizedUpdates),
    });

    return NextResponse.json({
      success: true,
      data: {
        id: updatedUser.id,
        email: updatedUser.email,
        name: updatedUser.name,
        picture: updatedUser.picture,
        roles: updatedUser.roles,
        tenantId: updatedUser.tenantId,
        createdAt: updatedUser.createdAt,
        updatedAt: updatedUser.updatedAt,
      },
      message: 'Profile updated successfully',
    });
  } catch (error) {
    authLogger.error('Unexpected error updating user profile', error);

    return NextResponse.json(
      {
        success: false,
        error: 'internal_error',
        message: 'An unexpected error occurred',
      },
      { status: 500 }
    );
  }
}

/**
 * Validate if a string is a valid URL
 */
function isValidUrl(string: string): boolean {
  try {
    const url = new URL(string);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

/**
 * DELETE /api/auth/profile
 *
 * Delete the current user's profile (account deletion)
 */
export async function DELETE(): Promise<NextResponse> {
  // Account deletion would be implemented here
  // This is a sensitive operation that would require additional verification

  authLogger.warn('Account deletion requested but not implemented');

  return NextResponse.json(
    {
      success: false,
      error: 'not_implemented',
      message:
        'Account deletion is not yet implemented. Please contact support.',
    },
    { status: 501 }
  );
}

/**
 * PATCH method not allowed - use PUT for updates
 */
export async function PATCH(): Promise<NextResponse> {
  return NextResponse.json(
    {
      error: 'method_not_allowed',
      message: 'Use PUT method for profile updates',
    },
    { status: 405 }
  );
}
