/**
 * Reading the entry the active tab points at.
 *
 * Nothing is cached: reopening a tab re-reads. An entry editor showing a
 * remembered copy of something another administrator has since changed is the
 * failure FR-105 exists to prevent, and it is worse than a moment's wait.
 */
import { useCallback, useEffect, useRef, useState } from 'react';

import { bridge, isEmbedded } from '@/bridge/client';
import type { Entry, Result } from '@/bridge/types';

export interface EntryViewModel {
  entry?: Entry;
  result?: Result;
  loading: boolean;
  error?: string;
  reload: () => void;
}

export function useEntryViewModel(
  profileId: string | undefined,
  dn: string | undefined,
  includeOperational: boolean,
): EntryViewModel {
  const [entry, setEntry] = useState<Entry>();
  const [result, setResult] = useState<Result>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [nudge, setNudge] = useState(0);

  // Discards a response for an entry the user has already navigated away from.
  const generation = useRef(0);

  useEffect(() => {
    generation.current += 1;
    const mine = generation.current;

    setEntry(undefined);
    setResult(undefined);
    setError(undefined);

    if (!profileId || !dn || !isEmbedded()) return;

    setLoading(true);
    (async () => {
      try {
        const [read, outcome] = await bridge.readEntry(profileId, dn, includeOperational);
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
  }, [profileId, dn, includeOperational, nudge]);

  const reload = useCallback(() => setNudge((n) => n + 1), []);

  return { entry, result, loading, error, reload };
}
