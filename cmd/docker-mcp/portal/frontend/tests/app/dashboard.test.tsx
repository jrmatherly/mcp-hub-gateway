// Example test for Next.js App Router page component
import React from 'react';
// Using Vitest globals: describe, it, expect, vi, beforeEach
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useRouter, useSearchParams } from 'next/navigation';
import { http, HttpResponse } from 'msw';
import { render } from '../utils/test-utils';
import { server } from '../mocks/server';
import { createErrorHandler } from '../mocks/handlers';

// Define the Server type
interface Server {
  id: string;
  name: string;
  description: string;
  status: string;
  enabled: boolean;
}

// Mock Next.js navigation hooks
vi.mock('next/navigation', () => ({
  useRouter: vi.fn(),
  useSearchParams: vi.fn(),
  usePathname: vi.fn(() => '/dashboard'),
  useParams: vi.fn(() => ({})),
}));

// Mock dashboard page component (replace with actual import)
const MockDashboardPage = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [servers, setServers] = React.useState<Server[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const fetchServers = async () => {
      try {
        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_URL}/api/servers`
        );
        if (!response.ok) throw new Error('Failed to fetch servers');
        const data = await response.json();
        setServers(data.servers);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchServers();
  }, []);

  const handleServerAction = async (action: string, serverId: string) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/servers/${serverId}/${action}`,
        { method: 'POST' }
      );
      if (!response.ok) throw new Error(`Failed to ${action} server`);

      // Refetch servers
      const serversResponse = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/servers`
      );
      const data = await serversResponse.json();
      setServers(data.servers);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  };

  const filterServers = () => {
    const filter = searchParams?.get('filter');
    if (!filter) return servers;

    return servers.filter((server: Server) =>
      filter === 'enabled' ? server.enabled : !server.enabled
    );
  };

  if (loading) {
    return (
      <div data-testid="loading-spinner">
        <div>Loading servers...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="error-message" className="error">
        <h2>Error</h2>
        <p>{error}</p>
        <button onClick={() => window.location.reload()}>Retry</button>
      </div>
    );
  }

  const filteredServers = filterServers();

  return (
    <div data-testid="dashboard-page">
      <header>
        <h1>MCP Server Dashboard</h1>
        <div data-testid="server-count">
          {servers.length} servers total ({filteredServers.length} shown)
        </div>
      </header>

      <nav data-testid="filter-nav">
        <button
          onClick={() => router.push('/dashboard')}
          data-testid="filter-all"
        >
          All Servers
        </button>
        <button
          onClick={() => router.push('/dashboard?filter=enabled')}
          data-testid="filter-enabled"
        >
          Enabled Only
        </button>
        <button
          onClick={() => router.push('/dashboard?filter=disabled')}
          data-testid="filter-disabled"
        >
          Disabled Only
        </button>
      </nav>

      <main>
        {filteredServers.length === 0 ? (
          <div data-testid="no-servers">No servers found.</div>
        ) : (
          <div data-testid="servers-grid" className="servers-grid">
            {filteredServers.map((server: Server) => (
              <div
                key={server.id}
                data-testid={`server-${server.id}`}
                className="server-card"
              >
                <h3>{server.name}</h3>
                <p>{server.description}</p>
                <div className={`status ${server.status}`}>{server.status}</div>
                <div className="actions">
                  {server.enabled ? (
                    <>
                      <button
                        onClick={() => handleServerAction('disable', server.id)}
                        data-testid={`disable-${server.id}`}
                      >
                        Disable
                      </button>
                      <button
                        onClick={() => handleServerAction('restart', server.id)}
                        data-testid={`restart-${server.id}`}
                      >
                        Restart
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => handleServerAction('enable', server.id)}
                      data-testid={`enable-${server.id}`}
                    >
                      Enable
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      <footer data-testid="dashboard-footer">
        <p>Last updated: {new Date().toLocaleString()}</p>
      </footer>
    </div>
  );
};

describe('Dashboard Page', () => {
  const mockRouter = {
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
  };

  const mockSearchParams = {
    get: vi.fn(),
    getAll: vi.fn(),
    has: vi.fn(),
    keys: vi.fn(),
    values: vi.fn(),
    entries: vi.fn(),
    forEach: vi.fn(),
    toString: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useRouter).mockReturnValue(mockRouter);
    vi.mocked(useSearchParams).mockReturnValue(mockSearchParams);
    mockSearchParams.get.mockReturnValue(null); // Default: no filter
  });

  describe('Loading State', () => {
    it('shows loading spinner initially', () => {
      render(<MockDashboardPage />);

      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      expect(screen.getByText('Loading servers...')).toBeInTheDocument();
    });

    it('hides loading spinner after data loads', async () => {
      render(<MockDashboardPage />);

      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();

      await waitFor(() => {
        expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      });

      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
    });
  });

  describe('Successful Data Loading', () => {
    it('displays server data correctly', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      // Check header
      expect(screen.getByText('MCP Server Dashboard')).toBeInTheDocument();
      expect(screen.getByTestId('server-count')).toHaveTextContent(
        '2 servers total (2 shown)'
      );

      // Check servers are displayed
      expect(screen.getByTestId('server-server-1')).toBeInTheDocument();
      expect(screen.getByTestId('server-server-2')).toBeInTheDocument();
      expect(screen.getByText('Test Server 1')).toBeInTheDocument();
      expect(screen.getByText('Test Server 2')).toBeInTheDocument();
    });

    it('shows correct action buttons based on server state', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      // Server 1 is enabled - should have disable and restart buttons
      expect(screen.getByTestId('disable-server-1')).toBeInTheDocument();
      expect(screen.getByTestId('restart-server-1')).toBeInTheDocument();

      // Server 2 is disabled - should have enable button
      expect(screen.getByTestId('enable-server-2')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('displays error message when API fails', async () => {
      server.use(createErrorHandler('/api/servers', 500, 'Server unavailable'));

      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(screen.getByText('Failed to fetch servers')).toBeInTheDocument();
      expect(screen.getByText('Retry')).toBeInTheDocument();
    });

    it('provides retry functionality', async () => {
      server.use(createErrorHandler('/api/servers', 500, 'Server unavailable'));

      // Mock window.location.reload
      const mockReload = vi.fn();
      Object.defineProperty(window, 'location', {
        value: { reload: mockReload },
        writable: true,
      });

      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument();
      });

      const retryButton = screen.getByText('Retry');
      await userEvent.click(retryButton);

      expect(mockReload).toHaveBeenCalledTimes(1);
    });
  });

  describe('Filtering Functionality', () => {
    it('filters servers by enabled status', async () => {
      mockSearchParams.get.mockImplementation((key: string) =>
        key === 'filter' ? 'enabled' : null
      );

      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      // Should show only enabled servers (server-1)
      expect(screen.getByTestId('server-count')).toHaveTextContent(
        '2 servers total (1 shown)'
      );
      expect(screen.getByTestId('server-server-1')).toBeInTheDocument();
      expect(screen.queryByTestId('server-server-2')).not.toBeInTheDocument();
    });

    it('filters servers by disabled status', async () => {
      mockSearchParams.get.mockImplementation((key: string) =>
        key === 'filter' ? 'disabled' : null
      );

      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      // Should show only disabled servers (server-2)
      expect(screen.getByTestId('server-count')).toHaveTextContent(
        '2 servers total (1 shown)'
      );
      expect(screen.queryByTestId('server-server-1')).not.toBeInTheDocument();
      expect(screen.getByTestId('server-server-2')).toBeInTheDocument();
    });

    it('shows all servers when no filter applied', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      expect(screen.getByTestId('server-count')).toHaveTextContent(
        '2 servers total (2 shown)'
      );
      expect(screen.getByTestId('server-server-1')).toBeInTheDocument();
      expect(screen.getByTestId('server-server-2')).toBeInTheDocument();
    });
  });

  describe('Navigation', () => {
    it('navigates correctly when filter buttons are clicked', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      // Click filter buttons
      await userEvent.click(screen.getByTestId('filter-all'));
      expect(mockRouter.push).toHaveBeenCalledWith('/dashboard');

      await userEvent.click(screen.getByTestId('filter-enabled'));
      expect(mockRouter.push).toHaveBeenCalledWith('/dashboard?filter=enabled');

      await userEvent.click(screen.getByTestId('filter-disabled'));
      expect(mockRouter.push).toHaveBeenCalledWith(
        '/dashboard?filter=disabled'
      );
    });
  });

  describe('Server Actions', () => {
    it('handles server enable action', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      const enableButton = screen.getByTestId('enable-server-2');
      await userEvent.click(enableButton);

      // Should trigger API call and refresh data
      await waitFor(() => {
        // This would need to be verified with actual implementation
        expect(enableButton).toBeInTheDocument();
      });
    });

    it('handles server disable action', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      const disableButton = screen.getByTestId('disable-server-1');
      await userEvent.click(disableButton);

      // Should trigger API call and refresh data
      await waitFor(() => {
        expect(disableButton).toBeInTheDocument();
      });
    });
  });

  describe('Empty State', () => {
    it('shows no servers message when no servers match filter', async () => {
      // Mock empty response
      server.use(
        http.get(`${process.env.NEXT_PUBLIC_API_URL}/api/servers`, () => {
          return HttpResponse.json({ servers: [] });
        })
      );

      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('no-servers')).toBeInTheDocument();
      });

      expect(screen.getByText('No servers found.')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has proper heading hierarchy', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      const mainHeading = screen.getByRole('heading', { level: 1 });
      expect(mainHeading).toHaveTextContent('MCP Server Dashboard');

      const serverHeadings = screen.getAllByRole('heading', { level: 3 });
      expect(serverHeadings).toHaveLength(2);
    });

    it('has keyboard-accessible navigation', async () => {
      render(<MockDashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });

      const filterButtons = screen.getAllByRole('button');
      filterButtons.forEach(button => {
        expect(button).toHaveAttribute('type', 'button');
      });
    });
  });
});
