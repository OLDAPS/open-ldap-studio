import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { CommandProvider, useCommands } from '@/app/CommandProvider';
import type { CommandSet } from '@/bridge/types';
import { fakeApplicationClient } from '../helpers/applicationClient';

const commandSet: CommandSet = {
  platform: 'linux',
  menus: ['LDAP'],
  tree: {
    File: [],
    Edit: [],
    Search: [],
    LDAP: [{ label: 'Refresh', commandId: 'refresh' }],
    Schema: [],
    Credentials: [],
    Preferences: [],
    Help: [],
  },
  commands: [
    {
      id: 'refresh',
      label: 'Refresh',
      menu: 'LDAP',
      order: 0,
      scope: 'global',
      enablement: 'connection.open',
    },
  ],
  bindings: { refresh: 'Ctrl+Enter' },
};

afterEach(cleanup);

it('loads menus and enabled keyboard commands through the injected application client', async () => {
  const getCommands = vi.fn().mockResolvedValue(commandSet);
  const client = fakeApplicationClient({ isAvailable: () => true, getCommands });
  const { result } = renderHook(useCommands, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });
  await waitFor(() => expect(result.current.commandSet).toEqual(commandSet));
  expect(getCommands).toHaveBeenCalledExactlyOnceWith();
  expect(result.current.menu('LDAP')).toEqual(commandSet.tree.LDAP);
  expect(result.current.chordFor('refresh')).toBe('Ctrl+Enter');

  const refresh = vi.fn();
  const unregister = result.current.register('refresh', refresh);
  const pressShortcut = () =>
    window.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Enter',
        ctrlKey: true,
        cancelable: true,
      }),
    );
  act(() => {
    pressShortcut();
  });
  expect(refresh).not.toHaveBeenCalled();
  act(() => result.current.setEnablement({ connectionOpen: true }));
  expect(result.current.isCommandEnabled('refresh')).toBe(true);
  act(() => {
    pressShortcut();
  });
  expect(refresh).toHaveBeenCalledTimes(1);
  unregister();
  act(() => {
    pressShortcut();
  });
  expect(refresh).toHaveBeenCalledTimes(1);
});

it('ignores commands returned by an obsolete application client', async () => {
  let resolve!: (commands: CommandSet) => void;
  const snapshot = new Promise<CommandSet>((done) => {
    resolve = done;
  });
  let client = fakeApplicationClient({ isAvailable: () => true, getCommands: () => snapshot });
  const { result, rerender } = renderHook(useCommands, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });
  client = fakeApplicationClient({ isAvailable: () => true, getCommands: async () => commandSet });
  rerender();
  await waitFor(() => expect(result.current.commandSet).toEqual(commandSet));

  await act(async () => resolve({ ...commandSet, commands: [], bindings: {} }));
  expect(result.current.commandSet).toEqual(commandSet);
});

it('keeps commands empty when the application client is unavailable', () => {
  const getCommands = vi.fn();
  const client = fakeApplicationClient({ getCommands });
  const { result } = renderHook(useCommands, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApplicationClientProvider client={client}>
        <CommandProvider>{children}</CommandProvider>
      </ApplicationClientProvider>
    ),
  });
  expect(getCommands).not.toHaveBeenCalled();
  expect(result.current.commands).toEqual([]);
});
