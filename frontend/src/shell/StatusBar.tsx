/**
 * The status bar, visible on every screen.
 *
 * FR-010 makes this non-optional: connection state, bind DN, target server,
 * entry counts, the current page and any active limit are always on screen, so
 * "which server am I on, and as whom" is never a question the user has to go
 * and answer.
 *
 * Two badges are safety features rather than decoration:
 *   - a read-only connection says so, because that is the state in which a
 *     write will be refused below the UI (design gap G3);
 *   - an unverified TLS connection says so for as long as it is open, because
 *     the opt-out was a decision and its consequence should not be invisible
 *     (FR-006).
 */
import { useSession } from '@/app/session';

export interface StatusCounts {
  /** Entries loaded in the active view, and the total where the server said. */
  loaded?: number;
  total?: number;
  page?: { current: number; of?: number };
  /** A limit the server reported enforcing, stated rather than swallowed. */
  limit?: string;
}

export function StatusBar({ counts }: { counts?: StatusCounts }) {
  const connection = useSession((s) => s.activeConnection());
  const allJobs = useSession((s) => s.jobs);
  const runningJobs = Object.values(allJobs).filter((j) => j.state === 'running');
  const togglePanel = useSession((s) => s.togglePanel);

  const state = connection?.state ?? 'disconnected';
  const stateLabel: Record<string, string> = {
    disconnected: 'not connected',
    connecting: 'connecting…',
    connected: 'connected',
    lost: 'connection lost',
  };

  return (
    <footer className="statusbar" role="status">
      <span className="statusbar__state" data-state={state}>
        <span className="statusbar__dot" aria-hidden="true" />
        {stateLabel[state]}
      </span>

      {connection?.serverIdentity ? (
        <span className="statusbar__item mono" title={connection.serverIdentity}>
          {connection.serverIdentity}
        </span>
      ) : null}

      {connection?.state === 'connected' ? (
        <span className="statusbar__item mono" title="bind DN">
          {connection.boundDn ? `bound: ${connection.boundDn}` : 'bound: anonymous'}
        </span>
      ) : null}

      {connection?.readOnly ? (
        <span
          className="badge badge--info"
          title="No change can be committed through this connection"
        >
          read-only
        </span>
      ) : null}

      {connection?.production ? (
        <span className="badge badge--warning" title="This connection is tagged production">
          production
        </span>
      ) : null}

      {connection?.state === 'connected' && !connection.encrypted ? (
        <span className="badge badge--danger" title="This connection is not encrypted">
          cleartext
        </span>
      ) : null}

      {connection?.state === 'connected' && connection.encrypted && !connection.tlsVerified ? (
        <span
          className="badge badge--warning"
          title="Certificate verification is off for this connection"
        >
          TLS unverified
        </span>
      ) : null}

      {connection?.writesRequireConfirmation ? (
        <span
          className="badge badge--warning"
          title="The session was re-established; the next write is confirmed afresh"
        >
          reconnected
        </span>
      ) : null}

      <span className="statusbar__spacer" />

      {counts?.loaded !== undefined ? (
        <span className="statusbar__item mono">
          {counts.total !== undefined
            ? `${counts.loaded.toLocaleString()} of ${counts.total.toLocaleString()} entries`
            : `${counts.loaded.toLocaleString()} entries`}
        </span>
      ) : null}

      {counts?.page ? (
        <span className="statusbar__item mono">
          page {counts.page.current}
          {counts.page.of ? ` of ${counts.page.of}` : ''}
        </span>
      ) : null}

      {counts?.limit ? (
        <span className="badge badge--warning" title="A server limit is in force">
          {counts.limit}
        </span>
      ) : null}

      {runningJobs.length > 0 ? (
        <button type="button" className="statusbar__jobs" onClick={togglePanel}>
          {runningJobs.length} running
        </button>
      ) : null}
    </footer>
  );
}
