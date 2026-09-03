/**
 * Screen 5d — LDIF & text editors.
 *
 * Everything here is output layout: wrap width, folding, encoding. None of it
 * touches value fidelity — a value is never altered to make a line fit.
 */
export function LdifPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Formatting</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-wrap">
            Wrap lines at
          </label>
          <input
            id="pref-wrap"
            className="field"
            type="number"
            defaultValue={78}
            style={{ width: 90, flex: 'none' }}
          />
          <label className="field-row__label field-row__label--auto" htmlFor="pref-b64">
            Fold base64 at
          </label>
          <input
            id="pref-b64"
            className="field"
            type="number"
            defaultValue={76}
            style={{ width: 90, flex: 'none' }}
          />
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Show</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              line numbers
            </button>
            <button type="button" className="tag">
              whitespace
            </button>
            <button type="button" className="tag">
              record folding
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-encoding">
            Encoding · EOL
          </label>
          <select id="pref-encoding" className="field" defaultValue="utf8">
            <option value="utf8">UTF-8</option>
          </select>
          <select className="field" defaultValue="lf" aria-label="Line ending">
            <option value="lf">LF</option>
            <option value="crlf">CRLF</option>
          </select>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Validation &amp; syntax</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-validate">
            Validate
          </label>
          <select id="pref-validate" className="field" defaultValue="typing">
            <option value="typing">while typing</option>
            <option value="save">on save</option>
            <option value="manual">manual</option>
          </select>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Flag</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              unknown attributes
            </button>
            <button type="button" className="tag" data-selected>
              schema violations
            </button>
            <button type="button" className="tag" data-selected>
              missing DNs
            </button>
          </span>
        </div>
        <div className="placeholder" style={{ height: 56 }}>
          preview strip of a coloured LDIF record
          <br />
          with a warning gutter marker
        </div>
      </div>
    </>
  );
}
