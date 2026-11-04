import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, expect, it } from 'vitest';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { CommandProvider } from '@/app/CommandProvider';
import { useSession } from '@/app/session';
import { useShellViewModel } from '@/app/useShellViewModel';
import type { ConnState } from '@/bridge/types';
import { fakeApplicationClient } from '../helpers/applicationClient';

function state(profileId: string, value: ConnState['state']): ConnState {
  return {
    profileId,
    state: value,
    boundDn: '',
    serverIdentity: 'ldap.example.org',
    tlsVerified: true,
    encrypted: true,
    readOnly: false,
    production: false,
    writesRequireConfirmation: false,
  };
}

function startupClient() {
  let resolve!: (states: ConnState[]) => void;
  let reject!: (cause: Error) => void;
  const snapshot = new Promise<ConnState[]>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  const listeners = new Set<(event: ConnState) => void>();
  return {
    resolve,
    reject,
    emit: (event: ConnState) => listeners.forEach((handler) => handler(event)),
    client: fakeApplicationClient({
      isAvailable: () => true,
      connectionStates: () => snapshot,
      subscribe: (name, handler) => {
        if (name !== 'conn:state') return () => {};
        const onState = handler as (event: ConnState) => void;
        listeners.add(onState);
        return () => {
          listeners.delete(onState);
        };
      },
    }),
  };
}

beforeEach(() => useSession.setState({ connections: {}, activeProfileId: undefined, notices: [] }));
afterEach(cleanup);

it.each([
  ['connected', 'disconnected'],
  ['disconnected', 'connected'],
] as const)('keeps a live %s event when a delayed snapshot reports %s', async (live, stale) => {
  const startup = startupClient();
  renderHook(useShellViewModel, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={startup.client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });

  const latest = state('p1', live);
  act(() => startup.emit(latest));
  await act(async () => startup.resolve([state('p1', stale), state('p2', 'connected')]));

  expect(useSession.getState().connections).toEqual({
    p1: latest,
    p2: state('p2', 'connected'),
  });
});

it('ignores a snapshot from a replaced application client', async () => {
  const previous = startupClient();
  const current = startupClient();
  let client = previous.client;
  const { rerender } = renderHook(useShellViewModel, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });
  client = current.client;
  rerender();

  await act(async () => current.resolve([state('p1', 'connected')]));
  await act(async () => previous.resolve([state('p1', 'disconnected'), state('p2', 'connected')]));

  expect(useSession.getState().connections).toEqual({ p1: state('p1', 'connected') });
});

it('ignores a startup failure after the shell unmounts', async () => {
  const startup = startupClient();
  const { unmount } = renderHook(useShellViewModel, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={startup.client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });
  unmount();
  await act(async () => startup.reject(new Error('old client disconnected')));
  expect(useSession.getState().notices).toEqual([]);
});
