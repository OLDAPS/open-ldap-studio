/**
 * The attribute palette (screen 1c, sidebar bottom).
 *
 * Attributes are dragged from here into the filter, and the list comes from
 * the connected server's schema rather than a built-in table — a filter built
 * against attributes this directory does not have is a filter that returns
 * nothing for a reason nobody can see.
 */
export function AttributePalette() {
  return (
    <>
      <div className="sidebar__section">Attribute palette</div>
      <div className="placeholder" style={{ margin: '0 var(--space-3)', height: 110 }}>
        drag attributes from the schema
        <br />
        into the filter
      </div>
    </>
  );
}
