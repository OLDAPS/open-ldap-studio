/**
 * Screen 5b — Entry editor.
 *
 * "Auto on focus loss" is not among the save modes. A directory write is an
 * act, not a consequence of moving the cursor (deviation D3), so the setting
 * that would make it one does not exist rather than defaulting to off.
 */
export function EntryEditorPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Behaviour</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Default tab</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              attribute table
            </button>
            <button type="button" className="tag">
              LDIF view
            </button>
            <button type="button" className="tag">
              last used
            </button>
          </span>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Save mode</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              explicit (⌘S)
            </button>
          </span>
          <span className="dim">every commit passes through a preview</span>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Confirm before</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              deleting a value
            </button>
            <button type="button" className="tag" data-selected>
              deleting an attribute
            </button>
            <button type="button" className="tag" data-selected>
              leaving unsaved
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-schema-check">
            Schema check
          </label>
          <select id="pref-schema-check" className="field" defaultValue="typing">
            <option value="typing">validate while typing</option>
            <option value="save">on save</option>
            <option value="off">off</option>
          </select>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Table</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Attribute names as</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              short name
            </button>
            <button type="button" className="tag">
              OID
            </button>
            <button type="button" className="tag">
              description
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-group">
            Group rows by
          </label>
          <select id="pref-group" className="field" defaultValue="kind">
            <option value="kind">must / may / operational</option>
            <option value="alpha">alphabetical</option>
            <option value="class">objectClass</option>
          </select>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-fold">
            Multi-values
          </label>
          <input
            id="pref-fold"
            className="field"
            type="number"
            defaultValue={3}
            style={{ width: 90, flex: 'none' }}
          />
          <span className="dim">shown before &ldquo;+N more&rdquo;</span>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Copy as</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              LDIF
            </button>
            <button type="button" className="tag">
              CSV
            </button>
            <button type="button" className="tag">
              DN only
            </button>
          </span>
        </div>
      </div>
    </>
  );
}
