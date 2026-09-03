/**
 * The bottom panel: Progress, Modification Logs, Search Logs, Errors, Console.
 *
 * Progress lives here rather than in a modal because operations are
 * cancellable and concurrent: a modal would make one job the only thing the
 * user can look at, and cancelling it the only thing they can do (FR-089,
 * FR-099).
 */
import { useSession } from '@/store/session';
import type { PanelTab } from '@/store/session';
import { bridge } from '@/bridge/client';

const TABS: { id: PanelTab; label: string }[] = [
  { id: 'progress', label: 'Progress' },
  { id: 'modifications', label: 'Modification Logs' },
  { id: 'searches', label: 'Search Logs' },
  { id: 'errors', label: 'Errors' },
  { id: 'console', label: 'Console' },
];

export function BottomPanel() {
  const panelTab = useSession((s) => s.panelTab);
  const panelOpen = useSession((s) => s.panelOpen);
  const setPanelTab = useSession((s) => s.setPanelTab);
  const togglePanel = useSession((s) => s.togglePanel);
  const allJobs = useSession((s) => s.jobs);
  const notices = useSession((s) => s.notices);
  const capabilities = useSession((s) => s.capabilityNotices);

  const jobs = Object.values(allJobs);
  const running = jobs.filter((j) => j.state === 'running');
  const errors = notices.filter((n) => n.level === 'error');

  const count = (tab: PanelTab) => {
    if (tab === 'progress' && running.length) return running.length;
    if (tab === 'errors' && errors.length) return errors.length;
    return undefined;
  };

  return (
    <section className="panel" data-open={panelOpen || undefined} aria-label="Progress and logs">
      <div className="panel__tabs" role="tablist">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={panelTab === tab.id}
            className="panel__tab"
            data-active={panelTab === tab.id || undefined}
            onClick={() => setPanelTab(tab.id)}
          >
            {tab.label}
            {count(tab.id) ? <span className="panel__count">{count(tab.id)}</span> : null}
          </button>
        ))}
        <span className="panel__spacer" />
        <button
          type="button"
          className="panel__collapse"
          onClick={togglePanel}
          aria-label={panelOpen ? 'Collapse panel' : 'Expand panel'}
        >
          {panelOpen ? '▾' : '▴'}
        </button>
      </div>

      {panelOpen ? (
        <div className="panel__body">
          {panelTab === 'progress' ? <ProgressList /> : null}
          {panelTab === 'errors' ? <ErrorList /> : null}
          {panelTab === 'console' ? <ConsoleList /> : null}
          {panelTab === 'modifications' ? (
            <EmptyPanel text="Committed changes are recorded here as LDIF, redacted before they are written." />
          ) : null}
          {panelTab === 'searches' ? (
            <EmptyPanel text="Every search is logged with its filter, scope and result code." />
          ) : null}

          {capabilities.length > 0 && panelTab === 'console' ? null : null}
        </div>
      ) : null}
    </section>
  );
}

function ProgressList() {
  const allJobs = useSession((s) => s.jobs);
  const jobs = Object.values(allJobs);

  if (jobs.length === 0) {
    return <EmptyPanel text="No operations have run in this session." />;
  }

  return (
    <table className="grid">
      <thead>
        <tr>
          <th style={{ width: '30%' }}>Operation</th>
          <th style={{ width: '40%' }}>Progress</th>
          <th style={{ width: '20%' }}>State</th>
          <th style={{ width: '10%' }} />
        </tr>
      </thead>
      <tbody>
        {jobs.map((job) => {
          const percent = job.total > 0 ? Math.round((job.done / job.total) * 100) : undefined;
          return (
            <tr key={job.id}>
              <td>
                {job.kind}
                {job.mode === 'dryRun' ? <span className="badge badge--info">dry run</span> : null}
              </td>
              <td>
                <div className="progress">
                  <div
                    className="progress__bar"
                    style={{ width: percent !== undefined ? `${percent}%` : '100%' }}
                    data-indeterminate={percent === undefined || undefined}
                  />
                </div>
                <span className="dim mono">{job.message || job.summary || ''}</span>
              </td>
              <td>
                <span className="badge" data-job-state={job.state}>
                  {job.state}
                </span>
              </td>
              <td>
                {job.state === 'running' ? (
                  <button
                    type="button"
                    className="button button--quiet"
                    onClick={() => void bridge.cancel(job.id)}
                  >
                    Cancel
                  </button>
                ) : null}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function ErrorList() {
  const allNotices = useSession((s) => s.notices);
  const notices = allNotices.filter((n) => n.level === 'error');
  if (notices.length === 0) {
    return <EmptyPanel text="No errors in this session." />;
  }
  return (
    <ul className="notice-list">
      {notices.map((n) => (
        <li key={n.id}>
          <span className="notice-list__message">{n.message}</span>
          {n.detail ? <span className="notice-list__detail mono">{n.detail}</span> : null}
        </li>
      ))}
    </ul>
  );
}

function ConsoleList() {
  const capabilities = useSession((s) => s.capabilityNotices);
  const notices = useSession((s) => s.notices);

  if (capabilities.length === 0 && notices.length === 0) {
    return <EmptyPanel text="Nothing logged yet." />;
  }

  return (
    <ul className="notice-list">
      {capabilities.map((c) => (
        <li key={`${c.profileId}-${c.capability}`}>
          <span className="badge badge--warning">{c.capability}</span>
          <span className="notice-list__message">{c.reason}</span>
        </li>
      ))}
      {notices.map((n) => (
        <li key={n.id}>
          <span className="notice-list__message">{n.message}</span>
          {n.detail ? <span className="notice-list__detail mono">{n.detail}</span> : null}
        </li>
      ))}
    </ul>
  );
}

function EmptyPanel({ text }: { text: string }) {
  return <p className="panel__empty dim">{text}</p>;
}
