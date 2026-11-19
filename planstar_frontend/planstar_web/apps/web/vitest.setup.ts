import '@testing-library/jest-dom/vitest';

import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// 未开启 globals，Testing Library 不会自动注册 cleanup，这里显式清理。
afterEach(() => {
  cleanup();
});
