'use client';

import { ArrowLeft, FileQuestion, Home } from 'lucide-react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';

export default function NotFound() {
  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="max-w-md w-full text-center">
        <div className="flex justify-center mb-6">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-muted rounded-2xl">
            <FileQuestion className="h-8 w-8 text-muted-foreground" />
          </div>
        </div>

        <h1 className="text-6xl font-bold text-muted-foreground mb-4">404</h1>

        <h2 className="text-2xl font-bold text-foreground mb-2">
          Page Not Found
        </h2>

        <p className="text-muted-foreground mb-8">
          The page you're looking for doesn't exist or has been moved.
        </p>

        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <Button asChild>
            <Link href="/">
              <Home className="h-4 w-4 mr-2" />
              Go Home
            </Link>
          </Button>

          <Button variant="outline" onClick={() => window.history.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Go Back
          </Button>
        </div>

        <div className="mt-8">
          <p className="text-sm text-muted-foreground mb-2">
            Looking for something specific?
          </p>
          <div className="flex flex-wrap gap-2 justify-center text-xs">
            <Link href="/dashboard" className="text-primary hover:underline">
              Dashboard
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link
              href="/dashboard/servers"
              className="text-primary hover:underline"
            >
              Servers
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link
              href="/dashboard/config"
              className="text-primary hover:underline"
            >
              Configuration
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link href="/auth/login" className="text-primary hover:underline">
              Login
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
