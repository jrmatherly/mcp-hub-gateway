/**
 * API Route: /api/auth/config
 *
 * Returns public Azure AD configuration for client-side MSAL initialization.
 * This endpoint provides only the public configuration that's safe to expose
 * to the browser, while keeping sensitive values server-side only.
 */

import { NextResponse } from 'next/server';
import { env } from '@/env.mjs';

/**
 * GET /api/auth/config
 *
 * Returns the public MSAL configuration for the client.
 * The actual Azure AD credentials are stored server-side only.
 */
export async function GET() {
  try {
    // Return only the public configuration
    // Client ID and Tenant ID are considered public in Azure AD apps
    // The client secret is NEVER sent to the client
    const config = {
      auth: {
        clientId: env.AZURE_CLIENT_ID,
        authority: `https://login.microsoftonline.com/${env.AZURE_TENANT_ID}`,
        redirectUri: env.NEXT_PUBLIC_AZURE_REDIRECT_URI,
        postLogoutRedirectUri: env.NEXT_PUBLIC_AZURE_POST_LOGOUT_URI,
      },
      scopes: {
        loginRequest: ['openid', 'profile', 'email', 'User.Read'],
        apiRequest: [`api://${env.AZURE_CLIENT_ID}/access_as_user`],
      },
      features: {
        enablePersistence: true,
        enableSilentAcquire: true,
      },
    };

    // Set cache headers to prevent caching sensitive configuration
    return NextResponse.json(config, {
      headers: {
        'Cache-Control': 'no-store, no-cache, must-revalidate',
        Pragma: 'no-cache',
        Expires: '0',
      },
    });
  } catch (error) {
    console.error('Failed to generate auth config:', error);
    return NextResponse.json(
      { error: 'Azure AD configuration not properly configured on server' },
      { status: 500 }
    );
  }
}
