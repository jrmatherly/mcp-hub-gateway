// Test to verify Vitest setup is working correctly
// Using Vitest globals: describe, it, expect, vi

describe('Vitest Setup', () => {
  it('should run basic tests', () => {
    expect(true).toBe(true);
  });

  it('should support async tests', async () => {
    const promise = Promise.resolve('test');
    const result = await promise;
    expect(result).toBe('test');
  });

  it('should support mocking with vi', () => {
    const mockFn = vi.fn();
    mockFn('test');
    expect(mockFn).toHaveBeenCalledWith('test');
  });

  it('should have access to environment variables', () => {
    expect(process.env.NODE_ENV).toBe('test');
    expect(process.env.NEXT_PUBLIC_API_URL).toBe('http://localhost:8080');
  });

  it('should support DOM globals', () => {
    expect(window).toBeDefined();
    expect(document).toBeDefined();
    expect(localStorage).toBeDefined();
  });

  it('should have mocked WebSocket', () => {
    expect(WebSocket).toBeDefined();
    const ws = new WebSocket('ws://localhost:8080');
    expect(ws).toBeDefined();
  });

  it('should have mocked IntersectionObserver', () => {
    expect(IntersectionObserver).toBeDefined();
    const observer = new IntersectionObserver(() => {});
    expect(observer).toBeDefined();
  });

  it('should have mocked ResizeObserver', () => {
    expect(ResizeObserver).toBeDefined();
    const observer = new ResizeObserver(() => {});
    expect(observer).toBeDefined();
  });

  it('should have mocked matchMedia', () => {
    expect(window.matchMedia).toBeDefined();
    const media = window.matchMedia('(max-width: 768px)');
    expect(media).toBeDefined();
  });
});
