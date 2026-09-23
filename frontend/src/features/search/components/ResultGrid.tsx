/**
 * The result grid (screen 1c, lower half).
 *
 * Columns are the returning attributes, plus the row number and the DN. The
 * bar above carries the count and the elapsed time, because a search that took
 * four seconds and a search that took forty are different events even when
 * they return the same rows.
 *
 * A size limit reached is reported in the last row rather than swallowed:
 * "247 results" and "247 results, and the server stopped counting" must never
 * look the same.
 */
import { useState } from 'react';

const COLS = '60px 130px 200px minmax(0, 1fr) 190px';

export function ResultGrid() {
  const [selected, setSelected] = useState<number | undefined>();
  const rows: { uid: string; cn: string; dn: string; extra: string }[] = [];

  return (
    <>
      <div className="toolbar">
        <span>{rows.length === 0 ? 'no results yet' : `${rows.length} results`}</span>
        <span className="toolbar__spacer" />
        <div className="actions">
          <button type="button" className="button" disabled={rows.length === 0}>
            Export LDIF / CSV
          </button>
          <button type="button" className="button" disabled={rows.length === 0}>
            Bulk modify…
          </button>
          <button type="button" className="button">
            Columns ▾
          </button>
        </div>
      </div>

      <div className="dgrid dgrid--flush" style={{ flex: 1 }}>
        <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
          <div>#</div>
          <div>uid</div>
          <div>cn</div>
          <div>dn</div>
          <div>attributes</div>
        </div>

        {rows.map((row, index) => (
          <div
            key={row.dn}
            className="dgrid__row"
            role="button"
            tabIndex={0}
            data-selected={selected === index || undefined}
            style={{ '--cols': COLS } as React.CSSProperties}
            onClick={() => setSelected(index)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') setSelected(index);
            }}
          >
            <div className="dim">{index + 1}</div>
            <div className="mono">{row.uid}</div>
            <div>{row.cn}</div>
            <div className="mono dim">{row.dn}</div>
            <div className="mono">{row.extra}</div>
          </div>
        ))}

        {rows.length === 0 ? (
          <p className="dim" style={{ padding: 'var(--space-4)', fontSize: 'var(--text-caption)' }}>
            Describe a filter above and run it. Results are paged; the grid says when the server
            stopped rather than showing a short list as if it were complete.
          </p>
        ) : null}
      </div>

      <div className="pane" style={{ flex: 'none' }}>
        <div className="placeholder" style={{ height: 66 }}>
          selected-row preview: the attribute table for the highlighted result
          <br />
          (double-click opens it as an entry-editor tab)
        </div>
      </div>
    </>
  );
}
