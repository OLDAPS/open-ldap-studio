/**
 * Saved searches and filter history (screen 1c, sidebar top).
 *
 * History is separate from saved searches on purpose: one is a thing the user
 * decided to keep, the other is a thing they happened to type. Collapsing them
 * makes the kept ones impossible to find.
 */
export function SavedSearches() {
  const saved: { id: string; name: string }[] = [];
  const history: string[] = [];

  return (
    <>
      {saved.length === 0 ? (
        <p className="sidebar__note">no saved searches yet</p>
      ) : (
        saved.map((search) => (
          <button key={search.id} type="button" className="tree-row">
            <span className="tree-row__glyph" aria-hidden="true" />
            <span>{search.name}</span>
          </button>
        ))
      )}

      <div className="sidebar__section">Filter history</div>
      {history.length === 0 ? (
        <p className="sidebar__note">filters you run appear here</p>
      ) : (
        history.map((filter) => (
          <button key={filter} type="button" className="tree-row">
            <span className="mono dim">{filter}</span>
          </button>
        ))
      )}
    </>
  );
}
