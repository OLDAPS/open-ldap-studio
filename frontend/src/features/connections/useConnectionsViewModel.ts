import { useCallback, useEffect, useRef, useState } from 'react';
import { useApplicationClient } from '@/app/ApplicationClientProvider';
import { useSession } from '@/app/session';
import type { ConnState, ProfileSummary } from '@/bridge/types';

export function useConnectionsViewModel() {
  const client = useApplicationClient();
  const [profiles, setProfiles] = useState<ProfileSummary[]>([]);
  const [selected, setSelected] = useState<string>();
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<string>();
  const generation = useRef(0);
  const connections = useSession((state) => state.connections);
  const setActiveProfile = useSession((state) => state.setActiveProfile);
  const setPerspective = useSession((state) => state.setPerspective);

  const refresh = useCallback(async () => {
    if (!client.isAvailable()) return;
    const mine = ++generation.current;
    try {
      const result = await client.listProfiles();
      if (generation.current === mine) setProfiles(result);
    } catch (cause) {
      if (generation.current === mine)
        setError(cause instanceof Error ? cause.message : String(cause));
    }
  }, [client]);
  useEffect(() => {
    void refresh();
    return () => {
      generation.current += 1;
    };
  }, [refresh]);

  const select = (id: string) => {
    setSelected(id);
    setActiveProfile(id);
  };
  const run = async (id: string, action: () => Promise<unknown>) => {
    setError(undefined);
    setBusy(id);
    try {
      await action();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(undefined);
    }
  };
  const connect = (id: string) => {
    setActiveProfile(id);
    return run(id, () => client.connect(id));
  };
  const disconnect = (id: string) => run(id, () => client.disconnect(id));
  const remove = (id: string) =>
    run(id, async () => {
      await client.deleteProfile(id);
      if (selected === id) setSelected(undefined);
      await refresh();
    });
  const saved = (id: string) => {
    select(id);
    void refresh();
  };
  const stateOf = (id: string): ConnState['state'] => connections[id]?.state ?? 'disconnected';
  const active = profiles.find((profile) => profile.id === selected);
  return {
    profiles,
    selected,
    busy,
    error,
    active,
    connections,
    available: client.isAvailable(),
    refresh,
    select,
    connect,
    disconnect,
    remove,
    saved,
    stateOf,
    browse: () => setPerspective('browser'),
  };
}

export type ConnectionsViewModel = ReturnType<typeof useConnectionsViewModel>;
