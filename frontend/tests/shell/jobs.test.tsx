import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { useJobsViewModel } from '@/app/useJobsViewModel';
import { useSession } from '@/app/session';
import { fakeApplicationClient } from '../helpers/applicationClient';

afterEach(cleanup);

it('reports a rejected cancellation without an unhandled promise', async () => {
  useSession.setState({ notices: [] });
  const client = fakeApplicationClient({ cancel: vi.fn().mockRejectedValue(new Error('job no longer exists')) });
  const { result } = renderHook(useJobsViewModel, { wrapper: ({ children }: { children: ReactNode }) => (
    <ApplicationClientProvider client={client}>{children}</ApplicationClientProvider>
  ) });
  await act(() => result.current.cancelJob('job-1'));
  expect(useSession.getState().notices.at(-1)).toMatchObject({
    level: 'error', message: 'Could not cancel the operation', detail: 'job no longer exists',
  });
});
