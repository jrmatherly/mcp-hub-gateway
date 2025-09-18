import { Container } from 'lucide-react';

export default function Loading() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="relative">
          <Container className="h-8 w-8 text-docker-500 animate-pulse" />
          <div className="absolute inset-0 h-8 w-8 animate-spin border-2 border-docker-200 border-t-docker-500 rounded-full"></div>
        </div>
        <div className="text-center">
          <h2 className="text-lg font-semibold text-foreground mb-1">
            Loading MCP Portal
          </h2>
          <p className="text-sm text-muted-foreground">
            Please wait while we prepare your dashboard...
          </p>
        </div>
      </div>
    </div>
  );
}
