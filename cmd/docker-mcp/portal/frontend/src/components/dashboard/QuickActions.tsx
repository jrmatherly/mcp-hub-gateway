'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';

export function QuickActions() {
  const router = useRouter();

  const actions = [
    {
      label: 'Add Server',
      description: 'Add a new MCP server',
      href: '/dashboard/servers/add',
      variant: 'default' as const,
    },
    {
      label: 'View Logs',
      description: 'Check system logs',
      href: '/dashboard/logs',
      variant: 'secondary' as const,
    },
    {
      label: 'Settings',
      description: 'Configure portal settings',
      href: '/dashboard/settings',
      variant: 'outline' as const,
    },
  ];

  return (
    <div className="card">
      <div className="card-header">
        <h3 className="text-lg font-semibold">Quick Actions</h3>
      </div>
      <div className="card-content">
        <div className="grid gap-3">
          {actions.map((action, index) => (
            <Button
              key={index}
              variant={action.variant}
              className="h-auto p-4 justify-start text-left"
              onClick={() => {
                router.push(action.href);
              }}
            >
              <div>
                <div className="font-medium mb-1">{action.label}</div>
                <div className="text-sm opacity-90">{action.description}</div>
              </div>
            </Button>
          ))}
        </div>
      </div>
    </div>
  );
}
