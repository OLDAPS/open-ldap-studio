/**
 * Reading the entry the active tab points at.
 *
 * Nothing is cached: reopening a tab re-reads. An entry editor showing a
 * remembered copy of something another administrator has since changed is the
 * failure FR-105 exists to prevent, and it is worse than a moment's wait.
 */
import { useCallback, useEffect, useRef, useState } from 'react';

import { useApplicationClient } from '@/app/ApplicationClientProvider';
import { entryLDIFLines } from './ldif';
import type { Entry, Result } from '@/bridge/types';

export interface EntryViewModel {
  entry?: Entry;
  result?: Result;
  loading: boolean;
  error?: string;
  reload: () => void;
  ldifLines: string[];
}

export function useEntryViewModel(
  profileId: string | undefined,
  dn: string | undefined,
  includeOperational: boolean,
): EntryViewModel {
  const client = useApplicationClient();
  const [entry, setEntry] = useState<Entry>();
  const [result, setResult] = useState<Result>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [nudge, setNudge] = useState(0);

  // Discards a response for an entry the user has already navigated away from.
  const generation = useRef(0);
  const reload = useCallback(() => setNudge((n) => n + 1), []);

  useEffect(() => {
    if (!profileId || !dn) return;
    return client.subscribe('entry:changed', (changed) => {
      if (changed.profileId === profileId && changed.dn.toLowerCase() === dn.toLowerCase())
        reload();
    });
  }, [client, profileId, dn, reload]);

  useEffect(() => {
    generation.current += 1;
    const mine = generation.current;

    setEntry(undefined);
    setResult(undefined);
    setError(undefined);
    setLoading(false);

    if (!profileId || !dn || !client.isAvailable()) return;

    setLoading(true);
    (async () => {
      try {
        const [read, outcome] = await client.readEntry(profileId, dn, includeOperational);
        if (generation.current !== mine) return;
        setEntry(read);
        setResult(outcome);
      } catch (cause) {
        if (generation.current !== mine) return;
        setError(cause instanceof Error ? cause.message : String(cause));
      } finally {
        if (generation.current === mine) setLoading(false);
      }
    })();
    return () => {
      generation.current += 1;
    };
  }, [client, profileId, dn, includeOperational, nudge]);

  return {
    entry,
    result,
    loading,
    error,
    reload,
    ldifLines: dn ? entryLDIFLines(dn, entry?.attributes ?? []) : [],
  };
}
