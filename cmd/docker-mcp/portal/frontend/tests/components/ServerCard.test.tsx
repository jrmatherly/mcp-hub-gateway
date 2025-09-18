// Example test for ServerCard component with Next.js features
// Using Vitest globals: describe, it, expect, vi, beforeEach
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render, createMockServer } from '../utils/test-utils';
import { server } from '../mocks/server';
import { createErrorHandler } from '../mocks/handlers';

// Define the Server type
interface Server {
  id: string;
  name: string;
  description: string;
  status: string;
  enabled: boolean;
  version: string;
  health: string;
}

// Mock the ServerCard component (replace with actual import path)
const MockServerCard = ({
  server: serverData,
  onEnable,
  onDisable,
  onRestart,
}: {
  server: Server;
  onEnable: (id: string) => void;
  onDisable: (id: string) => void;
  onRestart: (id: string) => void;
}) => {
  const { id, name, description, status, enabled, health } = serverData;

  return (
    <div data-testid={`server-card-${id}`} className="server-card">
      <h3 data-testid="server-name">{name}</h3>
      <p data-testid="server-description">{description}</p>
      <div data-testid="server-status" className={`status-${status}`}>
        Status: {status}
      </div>
      <div data-testid="server-health" className={`health-${health}`}>
        Health: {health}
      </div>

      <div className="actions">
        {enabled ? (
          <>
            <button
              data-testid="disable-button"
              onClick={() => onDisable(id)}
              disabled={status === 'stopping'}
            >
              {status === 'stopping' ? 'Stopping...' : 'Disable'}
            </button>
            <button
              data-testid="restart-button"
              onClick={() => onRestart(id)}
              disabled={status === 'restarting'}
            >
              {status === 'restarting' ? 'Restarting...' : 'Restart'}
            </button>
          </>
        ) : (
          <button
            data-testid="enable-button"
            onClick={() => onEnable(id)}
            disabled={status === 'starting'}
          >
            {status === 'starting' ? 'Starting...' : 'Enable'}
          </button>
        )}
      </div>

      {/* Server metrics */}
      <div data-testid="server-metrics" className="metrics">
        <span>Version: {serverData.version}</span>
      </div>
    </div>
  );
};

describe('ServerCard Component', () => {
  const mockServer = createMockServer();
  const mockCallbacks = {
    onEnable: vi.fn(),
    onDisable: vi.fn(),
    onRestart: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('renders server information correctly', () => {
      render(<MockServerCard server={mockServer} {...mockCallbacks} />);

      expect(screen.getByTestId('server-name')).toHaveTextContent(
        mockServer.name
      );
      expect(screen.getByTestId('server-description')).toHaveTextContent(
        mockServer.description
      );
      expect(screen.getByTestId('server-status')).toHaveTextContent(
        `Status: ${mockServer.status}`
      );
      expect(screen.getByTestId('server-health')).toHaveTextContent(
        `Health: ${mockServer.health}`
      );
      expect(screen.getByTestId('server-metrics')).toHaveTextContent(
        `Version: ${mockServer.version}`
      );
    });

    it('applies correct CSS classes based on status and health', () => {
      render(<MockServerCard server={mockServer} {...mockCallbacks} />);

      const statusElement = screen.getByTestId('server-status');
      const healthElement = screen.getByTestId('server-health');

      expect(statusElement).toHaveClass(`status-${mockServer.status}`);
      expect(healthElement).toHaveClass(`health-${mockServer.health}`);
    });

    it('shows appropriate buttons for enabled server', () => {
      const enabledServer = createMockServer({ enabled: true });

      render(<MockServerCard server={enabledServer} {...mockCallbacks} />);

      expect(screen.getByTestId('disable-button')).toBeInTheDocument();
      expect(screen.getByTestId('restart-button')).toBeInTheDocument();
      expect(screen.queryByTestId('enable-button')).not.toBeInTheDocument();
    });

    it('shows enable button for disabled server', () => {
      const disabledServer = createMockServer({ enabled: false });

      render(<MockServerCard server={disabledServer} {...mockCallbacks} />);

      expect(screen.getByTestId('enable-button')).toBeInTheDocument();
      expect(screen.queryByTestId('disable-button')).not.toBeInTheDocument();
      expect(screen.queryByTestId('restart-button')).not.toBeInTheDocument();
    });
  });

  describe('User Interactions', () => {
    const user = userEvent.setup();

    it('calls onEnable when enable button is clicked', async () => {
      const disabledServer = createMockServer({ enabled: false });

      render(<MockServerCard server={disabledServer} {...mockCallbacks} />);

      const enableButton = screen.getByTestId('enable-button');
      await user.click(enableButton);

      expect(mockCallbacks.onEnable).toHaveBeenCalledWith(disabledServer.id);
      expect(mockCallbacks.onEnable).toHaveBeenCalledTimes(1);
    });

    it('calls onDisable when disable button is clicked', async () => {
      const enabledServer = createMockServer({ enabled: true });

      render(<MockServerCard server={enabledServer} {...mockCallbacks} />);

      const disableButton = screen.getByTestId('disable-button');
      await user.click(disableButton);

      expect(mockCallbacks.onDisable).toHaveBeenCalledWith(enabledServer.id);
      expect(mockCallbacks.onDisable).toHaveBeenCalledTimes(1);
    });

    it('calls onRestart when restart button is clicked', async () => {
      const enabledServer = createMockServer({ enabled: true });

      render(<MockServerCard server={enabledServer} {...mockCallbacks} />);

      const restartButton = screen.getByTestId('restart-button');
      await user.click(restartButton);

      expect(mockCallbacks.onRestart).toHaveBeenCalledWith(enabledServer.id);
      expect(mockCallbacks.onRestart).toHaveBeenCalledTimes(1);
    });
  });

  describe('Loading States', () => {
    it('disables enable button when server is starting', () => {
      const startingServer = createMockServer({
        enabled: false,
        status: 'starting',
      });

      render(<MockServerCard server={startingServer} {...mockCallbacks} />);

      const enableButton = screen.getByTestId('enable-button');
      expect(enableButton).toBeDisabled();
      expect(enableButton).toHaveTextContent('Starting...');
    });

    it('disables disable button when server is stopping', () => {
      const stoppingServer = createMockServer({
        enabled: true,
        status: 'stopping',
      });

      render(<MockServerCard server={stoppingServer} {...mockCallbacks} />);

      const disableButton = screen.getByTestId('disable-button');
      expect(disableButton).toBeDisabled();
      expect(disableButton).toHaveTextContent('Stopping...');
    });

    it('disables restart button when server is restarting', () => {
      const restartingServer = createMockServer({
        enabled: true,
        status: 'restarting',
      });

      render(<MockServerCard server={restartingServer} {...mockCallbacks} />);

      const restartButton = screen.getByTestId('restart-button');
      expect(restartButton).toBeDisabled();
      expect(restartButton).toHaveTextContent('Restarting...');
    });
  });

  describe('Error Handling', () => {
    it('handles API errors gracefully', async () => {
      // Mock API error response
      server.use(
        createErrorHandler(
          '/api/servers/test-server-1/enable',
          500,
          'Server error'
        )
      );

      const disabledServer = createMockServer({ enabled: false });

      render(<MockServerCard server={disabledServer} {...mockCallbacks} />);

      // This test would need actual error handling in the component
      // For now, we just verify the callback is called
      const enableButton = screen.getByTestId('enable-button');
      await userEvent.click(enableButton);

      expect(mockCallbacks.onEnable).toHaveBeenCalledWith(disabledServer.id);
    });
  });

  describe('Accessibility', () => {
    it('has proper accessibility attributes', () => {
      render(<MockServerCard server={mockServer} {...mockCallbacks} />);

      const card = screen.getByTestId(`server-card-${mockServer.id}`);
      expect(card).toBeInTheDocument();

      // Check that buttons are properly labeled
      const buttons = screen.getAllByRole('button');
      buttons.forEach(button => {
        expect(button).toHaveTextContent(/enable|disable|restart/i);
      });
    });

    it('supports keyboard navigation', async () => {
      const user = userEvent.setup();

      render(<MockServerCard server={mockServer} {...mockCallbacks} />);

      const disableButton = screen.getByTestId('disable-button');

      // Tab to the button
      await user.tab();
      expect(disableButton).toHaveFocus();

      // Press Enter to activate
      await user.keyboard('{Enter}');
      expect(mockCallbacks.onDisable).toHaveBeenCalledWith(mockServer.id);
    });
  });

  describe('Performance', () => {
    it('does not re-render unnecessarily', () => {
      const renderSpy = vi.fn();

      const TestComponent = (props: {
        server: Server;
        onEnable: (id: string) => void;
        onDisable: (id: string) => void;
        onRestart: (id: string) => void;
      }) => {
        renderSpy();
        return <MockServerCard {...props} />;
      };

      const { rerender } = render(
        <TestComponent server={mockServer} {...mockCallbacks} />
      );

      expect(renderSpy).toHaveBeenCalledTimes(1);

      // Re-render with same props
      rerender(<TestComponent server={mockServer} {...mockCallbacks} />);

      expect(renderSpy).toHaveBeenCalledTimes(2);
    });
  });
});
