/**
 * The application shell, built once and shared by every perspective.
 *
 * Its shape is the wireframe's: a frameless title bar carrying the menus, a
 * six-icon activity rail, a sidebar, a document area, the progress-and-logs
 * panel, and a status bar that is never hidden (screens.md § Shell).
 */
import { useEffect } from 'react';

import { ActivityRail } from './ActivityRail';
import { StatusBar } from './StatusBar';
import { TitleBar } from './TitleBar';
import { events } from '@/bridge/events';
import { bridge, isEmbedded } from '@/bridge/client';
import { useCommands } from '@/commands/CommandProvider';
import { useSession } from '@/store/session';
import { BrowserView } from '@/views/BrowserView';
import { ConnectionsView } from '@/views/ConnectionsView';
import { FilesView } from '@/views/FilesView';
import { PreferencesView } from '@/views/PreferencesView';
import { SchemaView } from '@/views/SchemaView';
import { SearchesView } from '@/views/SearchesView';

export function Shell() {
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
    const unsubscribers = [
      events.connState((state) => setConnState(state)),
      events.jobStarted(({ jobId, kind, mode, total, profileId }) =>
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
      events.jobProgress(({ jobId, done, total, message }) => {
        const existing = useSession.getState().jobs[jobId];
        if (existing) upsertJob({ ...existing, done, total, message });
      }),
      events.jobFinished(({ jobId, state, summary, error }) => {
        const existing = useSession.getState().jobs[jobId];
        if (existing) {
          upsertJob({ ...existing, state, summary, error, endedAt: new Date().toISOString() });
        }
        if (state === 'failed' && error) {
          addNotice({ level: 'error', message: summary || 'Operation failed', detail: error });
        }
      }),
      // Every degradation is announced with the reason the server gave.
      events.capabilityUnavailable((notice) => addCapabilityNotice(notice)),
      events.credentialStoreUnavailable(({ reason }) =>
        addNotice({
          level: 'warning',
          message:
            'No platform credential store is available; secrets are held for this session only.',
          detail: reason,
        }),
      ),
      events.connLost(({ profileId, reason }) =>
        addNotice({ level: 'error', message: `Connection lost: ${profileId}`, detail: reason }),
      ),
    ];
    return () => unsubscribers.forEach((off) => off());
  }, [setConnState, upsertJob, addNotice, addCapabilityNotice]);

  // The status bar must be truthful from the first frame, so existing
  // connection state is pulled once at start-up rather than waited for.
  useEffect(() => {
    if (!isEmbedded()) return;
    bridge
      .connectionStates()
      .then((states) => states.forEach(setConnState))
      .catch(() => undefined);
  }, [setConnState]);

  return (
    <div className="shell">
      <TitleBar />

      <div className="shell__body">
        <ActivityRail />

        {perspective === 'connections' ? <ConnectionsView /> : null}
        {perspective === 'browser' ? <BrowserView /> : null}
        {perspective === 'searches' ? <SearchesView /> : null}
        {perspective === 'schema' ? <SchemaView /> : null}
        {perspective === 'files' ? <FilesView /> : null}
        {perspective === 'preferences' ? <PreferencesView /> : null}
      </div>

      <StatusBar />
    </div>
  );
}
