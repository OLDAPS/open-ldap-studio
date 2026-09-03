/**
 * "Used by N entries · run as search" (screen 1d).
 *
 * The count is a search the user could have written by hand, so the button
 * hands the search editor a definition rather than running anything here: the
 * cost of the query stays visible and cancellable where every other search is.
 */
export function UsedBySearch({ count }: { count?: number }) {
  return (
    <div className="field-group">
      <span className="mono">
        {count === undefined ? '—' : `${count.toLocaleString()} entries`}
      </span>
      <button type="button" className="button button--quiet" disabled={count === undefined}>
        run as search
      </button>
    </div>
  );
}
