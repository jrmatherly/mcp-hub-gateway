import { Container, Server, Settings, Shield } from 'lucide-react';
import Link from 'next/link';
import { Suspense } from 'react';
import { Button } from '@/components/ui/button';

// Loading component for Suspense fallback
function LoadingCard() {
  return (
    <div className="rounded-lg border bg-card p-6 shadow-sm animate-pulse">
      <div className="h-8 w-8 bg-muted rounded mb-4"></div>
      <div className="h-6 bg-muted rounded w-3/4 mb-2"></div>
      <div className="h-4 bg-muted rounded w-full"></div>
    </div>
  );
}

// Feature card component
interface FeatureCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  href: string;
  color: string;
}

function FeatureCard({
  icon: Icon,
  title,
  description,
  href,
  color,
}: FeatureCardProps) {
  return (
    <Link
      href={href}
      className="group relative overflow-hidden rounded-lg border bg-card p-6 shadow-sm transition-all hover:shadow-md hover:scale-[1.02] active:scale-[0.98]"
    >
      <div
        className={`inline-flex h-12 w-12 items-center justify-center rounded-lg ${color} mb-4`}
      >
        <Icon className="h-6 w-6 text-white" />
      </div>
      <h3 className="font-semibold text-lg mb-2 group-hover:text-primary transition-colors">
        {title}
      </h3>
      <p className="text-muted-foreground text-sm leading-relaxed">
        {description}
      </p>
    </Link>
  );
}

// Main page component
export default function HomePage() {
  return (
    <div className="container-responsive py-8 lg:py-12">
      {/* Hero Section */}
      <div className="text-center max-w-3xl mx-auto mb-12">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-docker-500 to-docker-600 rounded-2xl mb-6">
          <Container className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-4xl font-bold tracking-tight mb-4 bg-gradient-to-r from-docker-600 to-docker-800 bg-clip-text text-transparent">
          MCP Portal
        </h1>
        <p className="text-xl text-muted-foreground mb-8">
          A comprehensive web interface for managing Model Context Protocol
          (MCP) servers with Docker integration.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Button asChild size="lg">
            <Link href="/dashboard">Go to Dashboard</Link>
          </Button>
          <Button asChild variant="outline" size="lg">
            <Link href="/auth/login">Sign In</Link>
          </Button>
        </div>
      </div>

      {/* Features Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3 mb-12">
        <Suspense fallback={<LoadingCard />}>
          <FeatureCard
            icon={Server}
            title="Server Management"
            description="Enable, disable, and monitor MCP servers with real-time status updates and health monitoring."
            href="/dashboard/servers"
            color="bg-docker-500"
          />
        </Suspense>

        <Suspense fallback={<LoadingCard />}>
          <FeatureCard
            icon={Container}
            title="Container Lifecycle"
            description="Full Docker container management with resource monitoring and automated cleanup."
            href="/dashboard/containers"
            color="bg-success-500"
          />
        </Suspense>

        <Suspense fallback={<LoadingCard />}>
          <FeatureCard
            icon={Settings}
            title="Configuration"
            description="Centralized configuration management with encrypted storage and bulk operations."
            href="/dashboard/config"
            color="bg-warning-500"
          />
        </Suspense>

        <Suspense fallback={<LoadingCard />}>
          <FeatureCard
            icon={Shield}
            title="Security & Auth"
            description="Azure AD integration with role-based access control and audit logging."
            href="/dashboard/security"
            color="bg-error-500"
          />
        </Suspense>
      </div>

      {/* Quick Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <div className="text-center p-6 rounded-lg border bg-card">
          <div className="text-2xl font-bold text-docker-600 mb-1">0</div>
          <div className="text-sm text-muted-foreground">Active Servers</div>
        </div>
        <div className="text-center p-6 rounded-lg border bg-card">
          <div className="text-2xl font-bold text-success-600 mb-1">0</div>
          <div className="text-sm text-muted-foreground">
            Running Containers
          </div>
        </div>
        <div className="text-center p-6 rounded-lg border bg-card">
          <div className="text-2xl font-bold text-warning-600 mb-1">0</div>
          <div className="text-sm text-muted-foreground">Configurations</div>
        </div>
        <div className="text-center p-6 rounded-lg border bg-card">
          <div className="text-2xl font-bold text-error-600 mb-1">0</div>
          <div className="text-sm text-muted-foreground">Alerts</div>
        </div>
      </div>

      {/* Footer */}
      <footer className="mt-16 pt-8 border-t text-center text-sm text-muted-foreground">
        <p>
          Built with Next.js 15 and React 19 |{' '}
          <Link
            href="https://docker.com"
            className="hover:text-primary transition-colors"
          >
            Docker Inc.
          </Link>
        </p>
      </footer>
    </div>
  );
}
