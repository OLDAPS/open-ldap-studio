import {
  act,
  cleanup,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { useSession } from '@/app/session';
import { BrowserView } from '@/features/browser/components/BrowserView';
import { useDitTreeViewModel } from '@/features/browser/useDitTreeViewModel';
import type { MoreRow } from '@/features/browser/useDitTreeViewModel';
import { useEntryViewModel } from '@/features/browser/useEntryViewModel';
import type { ConnState, Result, RootDSE } from '@/bridge/types';
import { eventClient } from '../../helpers/eventClient';

const success: Result = { code: 0, matchedDn: '', diagnosticMessage: '', interpretation: '' };
const root = 'dc=example,dc=org';
const dse: RootDSE = {
  namingContexts: [root],
  supportedControl: [],
  supportedExtension: [],
  supportedSaslMechanisms: [],
  supportedLdapVersion: [],
  subschemaSubentry: '',
  raw: {},
};

function moreRow(rows: ReturnType<typeof useDitTreeViewModel>['rows']): MoreRow | undefined {
  return rows.find((row): row is MoreRow => row.kind === 'more');
}

beforeEach(() =>
  useSession.setState({
    connections: {},
    activeProfileId: undefined,
    tabs: [],
    activeTabId: undefined,
    jobs: {},
    notices: [],
  }),
);
afterEach(cleanup);

it('reloads the selected entry after a matching mutation event', async () => {
  const readEntry = vi
    .fn()
    .mockResolvedValueOnce([
      { dn: 'cn=a,' + root, attributes: [{ type: 'mail', values: ['b2xk'] }], hasChildren: 'no' },
      success,
    ])
    .mockResolvedValueOnce([
      { dn: 'cn=a,' + root, attributes: [{ type: 'mail', values: ['bmV3'] }], hasChildren: 'no' },
      success,
    ]);
  const events = eventClient({ readEntry });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <ApplicationClientProvider client={events.client}>{children}</ApplicationClientProvider>
  );
  const dn = 'cn=a,' + root;
  const { result } = renderHook(() => useEntryViewModel('p1', dn, false), { wrapper });
  await waitFor(() => expect(result.current.entry?.attributes[0]?.values).toEqual(['b2xk']));

  act(() => events.emit('entry:changed', { profileId: 'p2', dn, source: 'commit' }));
  expect(readEntry).toHaveBeenCalledTimes(1);
  act(() => events.emit('entry:changed', { profileId: 'p1', dn, source: 'commit' }));
  await waitFor(() => expect(result.current.entry?.attributes[0]?.values).toEqual(['bmV3']));
  expect(readEntry).toHaveBeenCalledTimes(2);
});

it('keeps the paging continuation after a temporary failure', async () => {
  const listChildren = vi
    .fn()
    .mockResolvedValueOnce({
      entries: [{ dn: 'cn=one,' + root, hasChildren: 'no' }],
      cookie: 'next',
      sessionId: 'session-1',
      result: success,
    })
    .mockRejectedValueOnce(new Error('temporary network failure'))
    .mockResolvedValueOnce({
      entries: [{ dn: 'cn=two,' + root, hasChildren: 'no' }],
      cookie: '',
      sessionId: 'session-1',
      result: success,
    });
  const events = eventClient({
    rootDSE: async (): Promise<[RootDSE, Result]> => [dse, success],
    listChildren,
  });
  const { result } = renderHook(() => useDitTreeViewModel('p1', true), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={events.client}>{children}</ApplicationClientProvider>
    ),
  });
  await waitFor(() => expect(result.current.loadingRoots).toBe(false));
  act(() => result.current.toggle(root));
  await waitFor(() => expect(moreRow(result.current.rows)?.loaded).toBe(1));
  act(() => result.current.fetchMore(root));
  await waitFor(() =>
    expect(result.current.rows.find((row) => row.kind === 'node' && row.dn === root)).toMatchObject(
      { error: 'temporary network failure' },
    ),
  );
  expect(moreRow(result.current.rows)).toMatchObject({ loaded: 1, restartRequired: false });
  act(() => result.current.fetchMore(root));
  await waitFor(() =>
    expect(
      result.current.rows.some((row) => row.kind === 'node' && row.dn === 'cn=two,' + root),
    ).toBe(true),
  );
  expect(listChildren.mock.calls[1]?.[2]).toMatchObject({ cookie: 'next', sessionId: 'session-1' });
  expect(listChildren.mock.calls[2]?.[2]).toMatchObject({ cookie: 'next', sessionId: 'session-1' });
});

it('offers a first-page restart when a paging session changes', async () => {
  const listChildren = vi
    .fn()
    .mockResolvedValueOnce({
      entries: [{ dn: 'cn=old,' + root, hasChildren: 'no' }],
      cookie: 'next',
      sessionId: 'session-1',
      result: success,
    })
    .mockRejectedValueOnce(
      new Error('the paging session changed; refresh the results before requesting another page'),
    )
    .mockResolvedValueOnce({
      entries: [{ dn: 'cn=fresh,' + root, hasChildren: 'no' }],
      cookie: '',
      sessionId: 'session-2',
      result: success,
    });
  const events = eventClient({
    rootDSE: async (): Promise<[RootDSE, Result]> => [dse, success],
    listChildren,
  });
  const { result } = renderHook(() => useDitTreeViewModel('p1', true), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={events.client}>{children}</ApplicationClientProvider>
    ),
  });
  await waitFor(() => expect(result.current.loadingRoots).toBe(false));
  act(() => result.current.toggle(root));
  await waitFor(() => expect(moreRow(result.current.rows)?.loaded).toBe(1));
  act(() => result.current.fetchMore(root));
  await waitFor(() => expect(moreRow(result.current.rows)?.restartRequired).toBe(true));
  act(() => result.current.fetchMore(root));
  expect(listChildren).toHaveBeenCalledTimes(2);
  act(() => result.current.restart(root));
  await waitFor(() =>
    expect(
      result.current.rows.some((row) => row.kind === 'node' && row.dn === 'cn=fresh,' + root),
    ).toBe(true),
  );
  expect(
    result.current.rows.some((row) => row.kind === 'node' && row.dn === 'cn=old,' + root),
  ).toBe(false);
  expect(listChildren.mock.calls[2]?.[2]).toMatchObject({ cookie: undefined });
});

it('refreshes the tree after mutation events and from its toolbar', async () => {
  const rootDSE = vi.fn().mockResolvedValue([dse, success]);
  const events = eventClient({ rootDSE });
  const connection: ConnState = {
    profileId: 'p1',
    state: 'connected',
    boundDn: '',
    serverIdentity: 'ldap.example.org',
    tlsVerified: true,
    encrypted: true,
    readOnly: false,
    production: false,
    writesRequireConfirmation: false,
  };
  useSession.setState({ activeProfileId: 'p1', connections: { p1: connection } });
  render(
    <ApplicationClientProvider client={events.client}>
      <BrowserView />
    </ApplicationClientProvider>,
  );
  await waitFor(() => expect(rootDSE).toHaveBeenCalledTimes(1));

  fireEvent.click(
    within(screen.getByLabelText('Directory tree')).getByRole('button', { name: 'Refresh' }),
  );
  await waitFor(() => expect(rootDSE).toHaveBeenCalledTimes(2));
  act(() =>
    events.emit('entry:changed', { profileId: 'p1', dn: 'cn=a,' + root, source: 'commit' }),
  );
  await waitFor(() => expect(rootDSE).toHaveBeenCalledTimes(3));
  act(() =>
    events.emit('entry:changed', { profileId: 'p2', dn: 'cn=a,' + root, source: 'commit' }),
  );
  expect(rootDSE).toHaveBeenCalledTimes(3);
});
