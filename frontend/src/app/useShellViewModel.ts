import { useEffect } from 'react';
import { useApplicationClient } from './ApplicationClientProvider';
import { useCommands } from './CommandProvider';
import { useSession } from './session';
import type { ApplicationEventMap } from '@/application/client';

/** Own the shell's subscriptions, lifecycle snapshots and command enablement. */
export function useShellViewModel() {
  const client = useApplicationClient();
  const perspective = useSession((s) => s.perspective);
  const theme = useSession((s) => s.theme);
  const density = useSession((s) => s.density);
  const setConnState = useSession((s) => s.setConnState);
  const upsertJob = useSession((s) => s.upsertJob);
  const addNotice = useSession((s) => s.addNotice);
  const addCapabilityNotice = useSession((s) => s.addCapabilityNotice);
  const connection = useSession((s) => s.activeConnection());
  const tabs = useSession((s) => s.tabs);

  const { setEnablement } = useCommands();

  // The theme is an attribute on the root so every token swap is one repaint.
  useEffect(() => {
    const root = document.documentElement;
    const resolved =
      theme === 'system'
        ? window.matchMedia?.('(prefers-color-scheme: light)').matches
          ? 'light'
          : 'dark'
        : theme;
    root.setAttribute('data-theme', resolved);
    root.setAttribute('data-density', density);
  }, [theme, density]);

  // Enablement is derived from session state, so a menu item and its shortcut
  // are never out of step.
  useEffect(() => {
    setEnablement({
      connectionOpen: connection?.state === 'connected',
      connectionReadOnly: Boolean(connection?.readOnly),
      profileSelected: Boolean(connection),
      editorOpen: tabs.length > 0,
    });
  }, [connection, tabs.length, setEnablement]);

  // Every event the Go core emits lands here once, and is fanned out from the
  // store rather than subscribed to in a dozen components.
  useEffect(() => {
    let active = true;
    const updatedProfiles = new Set<string>();
    const subscriptions = {
      connState: (handler: (event: ApplicationEventMap['conn:state']) => void) =>
        client.subscribe('conn:state', handler),
      jobStarted: (handler: (event: ApplicationEventMap['job:started']) => void) =>
        client.subscribe('job:started', handler),
      jobProgress: (handler: (event: ApplicationEventMap['job:progress']) => void) =>
        client.subscribe('job:progress', handler),
      jobFinished: (handler: (event: ApplicationEventMap['job:finished']) => void) =>
        client.subscribe('job:finished', handler),
      capabilityUnavailable: (
        handler: (event: ApplicationEventMap['capability:unavailable']) => void,
      ) => client.subscribe('capability:unavailable', handler),
      credentialStoreUnavailable: (
        handler: (event: ApplicationEventMap['credential:storeUnavailable']) => void,
      ) => client.subscribe('credential:storeUnavailable', handler),
      connLost: (handler: (event: ApplicationEventMap['conn:lost']) => void) =>
        client.subscribe('conn:lost', handler),
    };
    const unsubscribers = [
      subscriptions.connState((state) => {
        updatedProfiles.add(state.profileId);
        setConnState(state);
      }),
      subscriptions.jobStarted(({ jobId, kind, mode, total, profileId }) =>
        upsertJob({
          id: jobId,
          kind,
          mode,
          state: 'running',
          profileId,
          total: total ?? 0,
          done: 0,
          message: '',
          startedAt: new Date().toISOString(),
          paused: false,
        }),
      ),
      subscriptions.jobProgress(({ jobId, done, total, message }) => {
        const existing = useSession.getState().jobs[jobId];
        if (existing) upsertJob({ ...existing, done, total, message });
      }),
      subscriptions.jobFinished(({ jobId, state, summary, error }) => {
        const existing = useSession.getState().jobs[jobId];
        if (existing) {
          upsertJob({ ...existing, state, summary, error, endedAt: new Date().toISOString() });
        }
        if (state === 'indeterminate') {
          addNotice({
            level: 'error',
            message: 'Write outcome unknown; cancellation does not establish rollback',
            detail: error || summary,
          });
        } else if (state === 'failed' && error) {
          addNotice({ level: 'error', message: summary || 'Operation failed', detail: error });
        }
      }),
      // Every degradation is announced with the reason the server gave.
      subscriptions.capabilityUnavailable((notice) => addCapabilityNotice(notice)),
      subscriptions.credentialStoreUnavailable(({ reason }) =>
        addNotice({
          level: 'warning',
          message:
            'No platform credential store is available; secrets are held for this session only.',
          detail: reason,
        }),
      ),
      subscriptions.connLost(({ profileId, reason }) =>
        addNotice({ level: 'error', message: `Connection lost: ${profileId}`, detail: reason }),
      ),
    ];
    // Subscribe before requesting the snapshot. Live events take precedence
    // for their profiles even if the initial response arrives much later.
    if (client.isAvailable()) {
      client
        .connectionStates()
        .then((states) => {
          if (!active) return;
          states.forEach((state) => {
            if (!updatedProfiles.has(state.profileId)) setConnState(state);
          });
        })
        .catch((cause) => {
          if (!active) return;
          addNotice({
            level: 'error',
            message: 'Could not restore connection state',
            detail: String(cause),
          });
        });
    }
    return () => {
      active = false;
      unsubscribers.forEach((off) => off());
    };
  }, [client, setConnState, upsertJob, addNotice, addCapabilityNotice]);

  return { perspective };
}
