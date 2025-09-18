import 'vitest/globals';
import '@testing-library/jest-dom';

import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers';
import 'vitest';

declare module 'vitest' {
  // For module augmentation, we need interfaces (not type aliases) to extend the existing Vitest types
  // These interfaces are intentionally empty as they only extend TestingLibraryMatchers
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface Assertion<T = unknown> extends TestingLibraryMatchers<T, void> {}
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface AsymmetricMatchersContaining
    extends TestingLibraryMatchers<unknown, void> {}
}

declare global {
  namespace Vi {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface JestAssertion<T = unknown>
      extends TestingLibraryMatchers<T, void> {}
  }

  // Vitest globals
  const expect: typeof import('vitest').expect;
  const test: typeof import('vitest').test;
  const it: typeof import('vitest').it;
  const describe: typeof import('vitest').describe;
  const vi: typeof import('vitest').vi;
  const beforeAll: typeof import('vitest').beforeAll;
  const afterAll: typeof import('vitest').afterAll;
  const beforeEach: typeof import('vitest').beforeEach;
  const afterEach: typeof import('vitest').afterEach;
}

interface ImportMetaEnv {
  readonly NEXT_PUBLIC_API_URL: string;
  // add more env variables here as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
