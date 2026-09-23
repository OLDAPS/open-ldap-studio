/**
 * Screen 5c — Value editors: which editor opens for which syntax.
 *
 * Two tables, because the two rules resolve in order: an attribute-type
 * override beats the syntax default. Showing them as one table would hide the
 * precedence that decides what actually opens.
 */
const COLS = 'minmax(0, 1fr) 150px 96px';

export function ValueEditorsPane() {
  const byAttribute = [
    { key: 'userPassword', editor: 'Password', scope: 'all' },
    { key: 'jpegPhoto, photo', editor: 'Image', scope: 'all' },
    { key: 'userCertificate;binary', editor: 'Certificate', scope: 'all' },
    { key: 'manager, member, owner', editor: 'DN picker', scope: 'all' },
  ];

  const bySyntax = [
    { key: '…121.1.24 generalizedTime', editor: 'Date / time', scope: 'text' },
    { key: '…121.1.40 octetString', editor: 'Hex / base64', scope: 'text' },
  ];

  return (
    <>
      <div className="card">
        <span className="card__label">By attribute type</span>
        <div className="dgrid">
          <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
            <div>Attribute</div>
            <div>Editor</div>
            <div>Scope</div>
          </div>
          {byAttribute.map((row) => (
            <div
              key={row.key}
              className="dgrid__row"
              style={{ '--cols': COLS } as React.CSSProperties}
            >
              <div className="mono">{row.key}</div>
              <div>{row.editor}</div>
              <div className="dim">{row.scope}</div>
            </div>
          ))}
        </div>
        <div className="actions">
          <button type="button" className="button">
            + Add mapping
          </button>
          <button type="button" className="button">
            Remove
          </button>
        </div>
      </div>

      <div className="card">
        <span className="card__label">By syntax OID</span>
        <div className="dgrid">
          <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
            <div>Syntax</div>
            <div>Editor</div>
            <div>Fallback</div>
          </div>
          {bySyntax.map((row) => (
            <div
              key={row.key}
              className="dgrid__row"
              style={{ '--cols': COLS } as React.CSSProperties}
            >
              <div className="mono">{row.key}</div>
              <div>{row.editor}</div>
              <div className="dim">{row.scope}</div>
            </div>
          ))}
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-unknown">
            Unknown syntax
          </label>
          <select id="pref-unknown" className="field" defaultValue="text">
            <option value="text">multi-line text</option>
            <option value="hex">hex / base64</option>
          </select>
          <label className="field-row__label field-row__label--auto" htmlFor="pref-inline">
            Max inline length
          </label>
          <input
            id="pref-inline"
            className="field"
            type="number"
            defaultValue={120}
            style={{ width: 90, flex: 'none' }}
          />
        </div>
      </div>
    </>
  );
}
