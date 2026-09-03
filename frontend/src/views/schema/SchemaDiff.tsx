/**
 * Schema diff between two connections (screen 1d, "dev ↔ prod" tab).
 *
 * Difference is stated in words in its own column as well as by colour: a diff
 * read only through red and green is unreadable for a substantial minority,
 * and this is the screen where a misread precedes a schema change (gap G2).
 */
const COLS = 'minmax(0, 1fr) 140px 140px 110px';

export function SchemaDiff() {
  const rows: { element: string; left: string; right: string; state: string }[] = [];

  return (
    <div className="pane" style={{ flex: 1, minHeight: 0 }}>
      <div className="toolbar">
        <span>Schema diff</span>
        <span className="toolbar__spacer" />
        <div className="actions">
          <button type="button" className="button">
            Export as LDIF change
          </button>
        </div>
      </div>

      <div className="dgrid" style={{ flex: 1 }}>
        <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
          <div>Element</div>
          <div>Left</div>
          <div>Right</div>
          <div>State</div>
        </div>
        {rows.map((row) => (
          <div
            key={row.element}
            className="dgrid__row"
            style={{ '--cols': COLS } as React.CSSProperties}
          >
            <div className="mono">{row.element}</div>
            <div className="mono">{row.left}</div>
            <div className="mono">{row.right}</div>
            <div>{row.state}</div>
          </div>
        ))}
        {rows.length === 0 ? (
          <p className="dim" style={{ padding: 'var(--space-4)', fontSize: 'var(--text-caption)' }}>
            Pick two connections to compare their published schemas.
          </p>
        ) : null}
      </div>
    </div>
  );
}
