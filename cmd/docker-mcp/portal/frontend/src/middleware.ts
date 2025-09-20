/**
 * Next.js Middleware for Authentication and Route Protection
 *
 * This middleware runs on every request and handles:
 * - JWT token validation
 * - Route protection for authenticated areas
 * - Automatic token refresh
 * - Security headers
 * - Rate limiting
 */

import { NextRequest, NextResponse } from 'next/server';
import { env } from '@/env.mjs';

/**
 * Routes that require authentication
 */
const PROTECTED_ROUTES = [
  '/dashboard',
  '/admin',
  '/api/auth/profile',
  '/api/auth/logout',
  '/api/auth/sessions',
  '/api/auth/refresh',
  '/api/servers',
  '/api/catalogs',
] as const;

/**
 * Routes that should redirect to dashboard if user is already authenticated
 */
const AUTH_ROUTES = ['/auth/login', '/auth/register'] as const;

/**
 * Public routes that don't require authentication
 */
const PUBLIC_ROUTES = [
  '/',
  '/auth/callback',
  '/api/auth/config',
  '/api/health',
  '/api/reference',
] as const;

/**
 * Check if a path matches any pattern in the given array
 */
function matchesRoute(pathname: string, routes: readonly string[]): boolean {
  return routes.some(route => {
    if (route.endsWith('*')) {
      // Wildcard matching
      return pathname.startsWith(route.slice(0, -1));
    }
    return pathname === route || pathname.startsWith(`${route}/`);
  });
}

/**
 * Validate JWT token from cookies
 */
async function validateTokenFromCookies(request: NextRequest): Promise<{
  isValid: boolean;
  needsRefresh: boolean;
  user?: {
    id: string;
    email: string;
    name: string;
  };
}> {
  try {
    const cookieName = env.SESSION_COOKIE_NAME;
    const accessToken = request.cookies.get(cookieName)?.value;

    if (!accessToken) {
      return { isValid: false, needsRefresh: false };
    }

    // Use Edge Runtime compatible JWT utilities with jose
    const { verifyToken, getTokenExpiration } = await import('@/lib/jwt-edge');

    // Verify token with full signature validation using jose
    const payload = await verifyToken(accessToken);

    if (!payload) {
      return { isValid: false, needsRefresh: true };
    }

    const tokenInfo = getTokenExpiration(payload);

    // If token expires in less than 5 minutes, mark for refresh
    const needsRefresh = tokenInfo.expiresIn < 300; // 5 minutes

    if (tokenInfo.isExpired) {
      return { isValid: false, needsRefresh: true };
    }

    return {
      isValid: true,
      needsRefresh,
      user: {
        id: payload.sub,
        email: payload.email,
        name: payload.name || '',
      },
    };
  } catch (error) {
    console.error('Token validation error in middleware:', error);
    return { isValid: false, needsRefresh: false };
  }
}

/**
 * Add security headers to response
 */
function addSecurityHeaders(response: NextResponse): NextResponse {
  // Content Security Policy
  const csp = [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://unpkg.com",
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    "font-src 'self' https://fonts.gstatic.com",
    "img-src 'self' data: https:",
    "connect-src 'self' https://login.microsoftonline.com https://graph.microsoft.com ws: wss:",
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join('; ');

  response.headers.set('Content-Security-Policy', csp);
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
  response.headers.set(
    'Permissions-Policy',
    'camera=(), microphone=(), geolocation=()'
  );

  // Only set HSTS in production with HTTPS
  if (process.env.NODE_ENV === 'production') {
    response.headers.set(
      'Strict-Transport-Security',
      'max-age=31536000; includeSubDomains'
    );
  }

  return response;
}

/**
 * Simple rate limiting based on IP
 */
const rateLimitMap = new Map<string, { count: number; lastReset: number }>();

function checkRateLimit(
  ip: string,
  maxRequests: number = 100,
  windowMs: number = 60000
): boolean {
  const now = Date.now();
  const current = rateLimitMap.get(ip);

  if (!current) {
    rateLimitMap.set(ip, { count: 1, lastReset: now });
    return true;
  }

  // Reset window if expired
  if (now - current.lastReset > windowMs) {
    rateLimitMap.set(ip, { count: 1, lastReset: now });
    return true;
  }

  // Check if limit exceeded
  if (current.count >= maxRequests) {
    return false;
  }

  // Increment count
  current.count++;
  return true;
}

/**
 * Main middleware function
 */
export async function middleware(request: NextRequest): Promise<NextResponse> {
  const { pathname } = request.nextUrl;

  // Get client IP for rate limiting
  const clientIP =
    request.headers.get('x-forwarded-for')?.split(',')[0] ||
    request.headers.get('x-real-ip') ||
    'unknown';

  // Rate limiting
  if (!checkRateLimit(clientIP, 100, 60000)) {
    const response = NextResponse.json(
      { error: 'Rate limit exceeded' },
      { status: 429 }
    );
    return addSecurityHeaders(response);
  }

  // Skip middleware for static files and Next.js internals
  if (
    pathname.startsWith('/_next/') ||
    pathname.startsWith('/static/') ||
    (pathname.includes('.') && !pathname.includes('/api/'))
  ) {
    return NextResponse.next();
  }

  // Check if route is public
  if (matchesRoute(pathname, PUBLIC_ROUTES)) {
    const response = NextResponse.next();
    return addSecurityHeaders(response);
  }

  // Validate authentication for protected routes
  const tokenValidation = await validateTokenFromCookies(request);

  // Handle protected routes
  if (matchesRoute(pathname, PROTECTED_ROUTES)) {
    if (!tokenValidation.isValid) {
      // Clear invalid cookies
      const response = NextResponse.redirect(
        new URL('/auth/login', request.url)
      );
      response.cookies.delete(env.SESSION_COOKIE_NAME);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_refresh`);
      response.cookies.delete(`${env.SESSION_COOKIE_NAME}_user`);
      return addSecurityHeaders(response);
    }

    // If token needs refresh, add header to indicate this to the client
    if (tokenValidation.needsRefresh) {
      const response = NextResponse.next();
      response.headers.set('X-Token-Refresh-Needed', 'true');
      return addSecurityHeaders(response);
    }

    // Add user context to request headers for API routes
    if (pathname.startsWith('/api/') && tokenValidation.user) {
      const response = NextResponse.next();
      response.headers.set('X-User-ID', tokenValidation.user.id);
      response.headers.set('X-User-Email', tokenValidation.user.email);
      return addSecurityHeaders(response);
    }

    const response = NextResponse.next();
    return addSecurityHeaders(response);
  }

  // Handle auth routes (login, register)
  if (matchesRoute(pathname, AUTH_ROUTES)) {
    // If user is already authenticated, redirect to dashboard
    if (tokenValidation.isValid) {
      const response = NextResponse.redirect(
        new URL('/dashboard', request.url)
      );
      return addSecurityHeaders(response);
    }

    const response = NextResponse.next();
    return addSecurityHeaders(response);
  }

  // For all other routes, just add security headers
  const response = NextResponse.next();
  return addSecurityHeaders(response);
}

/**
 * Configure which routes the middleware should run on
 */
export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
};
