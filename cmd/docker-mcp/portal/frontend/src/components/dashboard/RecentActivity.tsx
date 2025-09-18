'use client';

import { Container, Server, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';

export function RecentActivity() {
  const activities = [
    {
      id: '1',
      type: 'server_enabled',
      message: 'Server "context7" was enabled',
      timestamp: '2 minutes ago',
      icon: Server,
      color: 'text-success-600',
    },
    {
      id: '2',
      type: 'container_started',
      message: 'Container for "magic" started successfully',
      timestamp: '5 minutes ago',
      icon: Container,
      color: 'text-docker-600',
    },
    {
      id: '3',
      type: 'config_updated',
      message: 'Configuration updated for "sequential"',
      timestamp: '15 minutes ago',
      icon: Settings,
      color: 'text-warning-600',
    },
  ];

  return (
    <div className="card">
      <div className="card-header">
        <h3 className="text-lg font-semibold">Recent Activity</h3>
      </div>
      <div className="card-content">
        <div className="space-y-4">
          {activities.map(activity => {
            const Icon = activity.icon;
            return (
              <div key={activity.id} className="flex items-start gap-3">
                <div className={`mt-0.5 ${activity.color}`}>
                  <Icon className="h-4 w-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-foreground">{activity.message}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {activity.timestamp}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
        <div className="mt-4 pt-4 border-t">
          <Button
            variant="link"
            size="sm"
            onClick={() => {
              // TODO: Implement view all activity navigation
            }}
          >
            View all activity
          </Button>
        </div>
      </div>
    </div>
  );
}
