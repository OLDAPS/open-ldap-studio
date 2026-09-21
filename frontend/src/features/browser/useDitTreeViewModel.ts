/**
 * The DIT tree's state: what has been fetched, what is expanded, what is left.
 *
 * Three things this deliberately does not do:
 *
 *   - It does not cache entries. A tree that remembers what a container held
 *     shows a stale directory after somebody else changes it (FR-105), so a
 *     collapse-and-reopen refetches.
 *   - It does not fetch on scroll. The next page is an explicit row the user
 *     clicks, and it says how many are left, because a tree that quietly
 *     issues searches against a production directory is a tree that can cost
 *     someone their afternoon (FR-018).
 *   - It does not guess whether a node has children. The server's tri-state
 *     answer is carried through: `unknown` renders as expandable-but-unproven
 *     rather than being resolved by a speculative search.
 */
import { useCallback, useEffect, useRef, useState } from 'react';

import { rdnOf } from './dn';
import { bridge, isEmbedded } from '@/bridge/client';
import type { Children, Entry, Profile } from '@/bridge/types';

export interface TreeRow {
  /** '' only for the synthetic "no naming context" placeholder. */
  dn: string;
  label: string;
  depth: number;
  hasChildren: Children;
  expanded: boolean;
  loading: boolean;
  /** Set on the row that failed, in the server's own words. */
  error?: string;
}

export interface MoreRow {
  kind: 'more';
  parentDn: string;
  depth: number;
  loaded: number;
  /** What the server said it was enforcing, 0 when it said nothing. */
  serverLimit: number;
  truncated: boolean;
  loading: boolean;
}

export type Row = ({ kind: 'node' } & TreeRow) | MoreRow;

/** One container's fetched children, and where the next page starts. */
interface Branch {
  children: { dn: string; hasChildren: Children }[];
  cookie?: string;
  loaded: number;
  serverLimit: number;
  truncated: boolean;
  loading: boolean;
  error?: string;
}

export interface DitTreeViewModel {
  rows: Row[];
  selected?: string;
  /** The naming contexts could not be read; nothing else will work either. */
  rootError?: string;
  loadingRoots: boolean;
  toggle: (dn: string) => void;
  select: (dn: string) => void;
  fetchMore: (parentDn: string) => void;
  refresh: () => void;
}

export function useDitTreeViewModel(
  profileId: string | undefined,
  connected: boolean,
): DitTreeViewModel {
  const [roots, setRoots] = useState<string[]>([]);
  const [rootError, setRootError] = useState<string>();
  const [loadingRoots, setLoadingRoots] = useState(false);
  const [branches, setBranches] = useState<Record<string, Branch>>({});
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [selected, setSelected] = useState<string>();
  const [nudge, setNudge] = useState(0);

  // Guards a late response from a connection the user has since left: without
  // it, switching profiles mid-fetch paints one server's tree under another
  // server's name.
  const generation = useRef(0);

  const reset = useCallback(() => {
    generation.current += 1;
    setRoots([]);
    setRootError(undefined);
    setBranches({});
    setExpanded(new Set());
    setSelected(undefined);
  }, []);

  // Roots: the naming contexts the server publishes, falling back to the
  // profile's configured base DN when it publishes none.
  useEffect(() => {
    reset();
    if (!profileId || !connected || !isEmbedded()) return;

    const mine = generation.current;
    setLoadingRoots(true);

    (async () => {
      try {
        const [dse] = await bridge.rootDSE(profileId);
        let contexts = dse.namingContexts ?? [];

        if (contexts.length === 0) {
          const profile: Profile = await bridge.getProfile(profileId);
          if (profile.baseDn) contexts = [profile.baseDn];
        }

        if (generation.current !== mine) return;
        setRoots(contexts);
        if (contexts.length === 0) {
          setRootError(
            'The server published no naming contexts and this connection has no base DN set.',
          );
        }
      } catch (cause) {
        if (generation.current !== mine) return;
        setRootError(cause instanceof Error ? cause.message : String(cause));
      } finally {
        if (generation.current === mine) setLoadingRoots(false);
      }
    })();
  }, [profileId, connected, reset, nudge]);

  const load = useCallback(
    async (parentDn: string, cookie?: string) => {
      if (!profileId) return;
      const mine = generation.current;

      setBranches((current) => ({
        ...current,
        [parentDn]: {
          children: current[parentDn]?.children ?? [],
          loaded: current[parentDn]?.loaded ?? 0,
          serverLimit: current[parentDn]?.serverLimit ?? 0,
          truncated: false,
          loading: true,
          cookie,
        },
      }));

      try {
        const page = await bridge.listChildren(profileId, parentDn, {
          size: 0, // the profile's page size decides
          cookie,
          includeOperational: false,
        });
        if (generation.current !== mine) return;

        setBranches((current) => {
          const existing = current[parentDn];
          // A continuation appends; a first page replaces, so a refresh cannot
          // leave two copies of the same child behind.
          const base = cookie ? (existing?.children ?? []) : [];
          const seen = new Set(base.map((c) => c.dn));
          const added = page.entries
            .filter((e: Entry) => !seen.has(e.dn))
            .map((e: Entry) => ({ dn: e.dn, hasChildren: e.hasChildren }));

          return {
            ...current,
            [parentDn]: {
              children: [...base, ...added],
              cookie: page.cookie || undefined,
              loaded: page.loadedCount || base.length + added.length,
              serverLimit: page.serverLimit ?? 0,
              truncated: Boolean(page.truncatedByServer),
              loading: false,
            },
          };
        });
      } catch (cause) {
        if (generation.current !== mine) return;
        setBranches((current) => ({
          ...current,
          [parentDn]: {
            children: current[parentDn]?.children ?? [],
            loaded: current[parentDn]?.loaded ?? 0,
            serverLimit: current[parentDn]?.serverLimit ?? 0,
            truncated: false,
            loading: false,
            error: cause instanceof Error ? cause.message : String(cause),
          },
        }));
      }
    },
    [profileId],
  );

  const toggle = useCallback(
    (dn: string) => {
      setExpanded((current) => {
        const next = new Set(current);
        if (next.has(dn)) {
          next.delete(dn);
          // Collapsing drops what was fetched: reopening asks the server
          // again rather than showing a snapshot of an hour ago.
          setBranches((all) => {
            const rest = { ...all };
            delete rest[dn];
            return rest;
          });
        } else {
          next.add(dn);
          void load(dn);
        }
        return next;
      });
    },
    [load],
  );

  const fetchMore = useCallback(
    (parentDn: string) => {
      const branch = branches[parentDn];
      if (branch?.cookie) void load(parentDn, branch.cookie);
    },
    [branches, load],
  );

  const refresh = useCallback(() => {
    setNudge((n) => n + 1);
  }, []);

  // Flatten depth-first into what the virtualiser renders.
  //
  // hasChildren is passed down rather than recomputed, because the only
  // authority on it is the server's answer for that entry: a node whose
  // children have not been fetched is not the same as a node that has none.
  const rows: Row[] = [];
  const walk = (dn: string, depth: number, hasChildren: Children) => {
    const branch = branches[dn];
    const isExpanded = expanded.has(dn);

    rows.push({
      kind: 'node',
      dn,
      label: depth === 0 ? dn : rdnOf(dn),
      depth,
      hasChildren,
      expanded: isExpanded,
      loading: Boolean(branch?.loading),
      error: branch?.error,
    });

    if (!isExpanded || !branch) return;

    for (const child of branch.children) {
      walk(child.dn, depth + 1, child.hasChildren);
    }

    if (branch.cookie || branch.truncated) {
      rows.push({
        kind: 'more',
        parentDn: dn,
        depth: depth + 1,
        loaded: branch.loaded,
        serverLimit: branch.serverLimit,
        truncated: branch.truncated,
        loading: branch.loading,
      });
    }
  };

  // A naming context's own children are unknown until it is opened.
  for (const root of roots) walk(root, 0, 'unknown');

  return {
    rows,
    selected,
    rootError,
    loadingRoots,
    toggle,
    select: setSelected,
    fetchMore,
    refresh,
  };
}
