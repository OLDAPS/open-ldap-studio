import { StrictMode } from 'react';
import type { ReactNode } from 'react';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { fakeApplicationClient } from '../../helpers/applicationClient';
import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import { useDitTreeViewModel } from '@/features/browser/useDitTreeViewModel';
import { useEntryViewModel } from '@/features/browser/useEntryViewModel';

const client = vi.hoisted(() => ({
  rootDSE: vi.fn(),
  getProfile: vi.fn(),
  listChildren: vi.fn(),
  readEntry: vi.fn(),
}));

const applicationClient = fakeApplicationClient({ ...client, isAvailable: () => true });
const wrapper = ({ children }: { children: ReactNode }) => (
  <ApplicationClientProvider client={applicationClient}>{children}</ApplicationClientProvider>
);

const success = { code: 0, matchedDn: '', diagnosticMessage: '', interpretation: '' };

beforeEach(() => {
  vi.resetAllMocks();
  client.rootDSE.mockResolvedValue([{ namingContexts: ['dc=example,dc=org'] }, success]);
  client.listChildren.mockResolvedValue({ entries: [], result: success });
});
afterEach(cleanup);

it('dispatches one request per tree expansion in StrictMode', async () => {
  const { result } = renderHook(() => useDitTreeViewModel('p1', true), {
    wrapper: ({ children }) => <StrictMode>{wrapper({ children })}</StrictMode>,
  });
  await waitFor(() => expect(result.current.loadingRoots).toBe(false));
  act(() => result.current.toggle('dc=example,dc=org'));
  await waitFor(() => expect(client.listChildren).toHaveBeenCalledTimes(1));
  act(() => result.current.toggle('dc=example,dc=org'));
  act(() => result.current.toggle('dc=example,dc=org'));
  await waitFor(() => expect(client.listChildren).toHaveBeenCalledTimes(2));
});

it('clears entry loading when selection disappears and ignores the obsolete read', async () => {
  let resolve!: (value: unknown) => void;
  client.readEntry.mockReturnValue(new Promise((done) => { resolve = done; }));
  const { result, rerender } = renderHook(
    ({ dn }: { dn: string | undefined }) => useEntryViewModel('p1', dn, false),
    { wrapper, initialProps: { dn: 'cn=old,dc=example,dc=org' as string | undefined } },
  );
  await waitFor(() => expect(result.current.loading).toBe(true));
  rerender({ dn: undefined });
  expect(result.current.loading).toBe(false);
  await act(async () => resolve([{ dn: 'cn=old', attributes: [] }, success]));
  expect(result.current.entry).toBeUndefined();
  expect(result.current.loading).toBe(false);
});
