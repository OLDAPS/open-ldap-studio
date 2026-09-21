/**
 * Back / forward through visited schema elements (screen 1d).
 *
 * The hierarchy diagram makes it easy to wander three classes deep; without a
 * way back, the only route home is the sidebar search.
 */
export function ElementNavigator({ current }: { current?: string }) {
  return (
    <div className="actions">
      <button type="button" className="button button--quiet" aria-label="Back">
        ←
      </button>
      <span className="mono">{current ?? '—'}</span>
      <button type="button" className="button button--quiet" aria-label="Forward">
        →
      </button>
    </div>
  );
}
