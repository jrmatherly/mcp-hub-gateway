// Simple component test to verify the setup works
import React from 'react';
// Using Vitest globals: describe, it, expect, vi
import { render, screen, userEvent } from '../utils/test-utils';

// Simple test component
const Button = ({
  onClick,
  children,
  disabled = false,
}: {
  onClick?: () => void;
  children: React.ReactNode;
  disabled?: boolean;
}) => (
  <button onClick={onClick} disabled={disabled} data-testid="test-button">
    {children}
  </button>
);

describe('Button Component', () => {
  it('renders button text correctly', () => {
    render(<Button>Click me</Button>);

    expect(screen.getByTestId('test-button')).toBeInTheDocument();
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('calls onClick when clicked', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    render(<Button onClick={handleClick}>Click me</Button>);

    const button = screen.getByTestId('test-button');
    await user.click(button);

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('disables button when disabled prop is true', () => {
    render(<Button disabled>Click me</Button>);

    const button = screen.getByTestId('test-button');
    expect(button).toBeDisabled();
  });

  it('does not call onClick when disabled', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    render(
      <Button onClick={handleClick} disabled>
        Click me
      </Button>
    );

    const button = screen.getByTestId('test-button');
    await user.click(button);

    expect(handleClick).not.toHaveBeenCalled();
  });
});
