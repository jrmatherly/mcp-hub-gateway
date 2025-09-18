import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Authentication',
  description: 'Sign in to MCP Portal',
};

interface AuthLayoutProps {
  children: React.ReactNode;
}

export default function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen flex">
      {/* Left side - Branding */}
      <div className="hidden lg:flex lg:flex-1 bg-gradient-to-br from-docker-600 to-docker-800 relative overflow-hidden">
        <div className="flex-1 flex items-center justify-center p-12">
          <div className="max-w-md text-center text-white">
            <div className="mb-8">
              <div className="inline-flex items-center justify-center w-20 h-20 bg-white/10 rounded-2xl backdrop-blur-sm border border-white/20 mb-6">
                <svg
                  className="w-10 h-10 text-white"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                  aria-hidden="true"
                >
                  <path
                    fillRule="evenodd"
                    d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
              </div>
              <h1 className="text-3xl font-bold mb-4">Welcome to MCP Portal</h1>
              <p className="text-docker-100 text-lg leading-relaxed">
                Manage your Model Context Protocol servers with ease. Secure,
                powerful, and built for developers.
              </p>
            </div>

            <div className="space-y-4 text-docker-100">
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 bg-docker-300 rounded-full"></div>
                <span>Docker-native server management</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 bg-docker-300 rounded-full"></div>
                <span>Real-time monitoring and logs</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 bg-docker-300 rounded-full"></div>
                <span>Enterprise-grade security</span>
              </div>
            </div>
          </div>
        </div>

        {/* Background decoration */}
        <div className="absolute inset-0 bg-gradient-to-t from-docker-900/20 to-transparent"></div>
        <div className="absolute -top-24 -right-24 w-96 h-96 bg-white/5 rounded-full"></div>
        <div className="absolute -bottom-32 -left-32 w-96 h-96 bg-white/5 rounded-full"></div>
      </div>

      {/* Right side - Auth forms */}
      <div className="flex-1 flex items-center justify-center p-8 bg-background">
        <div className="w-full max-w-md">{children}</div>
      </div>
    </div>
  );
}
