/**
 * Screen 5f — Credentials & security.
 *
 * The wireframe's in-app vault with its own master password is not built
 * (deviation D2): this application defers secret storage to the platform
 * agent, so what this pane offers is the policy around that store, not a
 * second store of its own. Where no platform store exists, the pane says so
 * rather than silently falling back to a file.
 */
export function SecurityPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Secret storage</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Storage</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              platform credential store
            </button>
            <button type="button" className="tag">
              session only
            </button>
            <button type="button" className="tag">
              never store
            </button>
          </span>
        </div>
        <div className="field-row">
          <span className="field-row__label">Unlock</span>
          <span className="dim">
            deferred to the first bind and raised by the platform agent — this application never
            prompts for a master password of its own
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-prompt">
            Password prompt
          </label>
          <select id="pref-prompt" className="field" defaultValue="session">
            <option value="session">once per session</option>
            <option value="bind">every bind</option>
          </select>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Policy</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Plaintext bind</span>
          <span className="tag-set">
            <button type="button" className="tag">
              warn
            </button>
            <button type="button" className="tag" data-selected>
              block unless StartTLS
            </button>
            <button type="button" className="tag">
              allow
            </button>
          </span>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">On exit</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              clear session secrets
            </button>
            <button type="button" className="tag">
              clear DN history
            </button>
          </span>
        </div>
        <div className="field-row">
          <span className="field-row__label">Trusted certificates</span>
          <button type="button" className="button">
            Manage store…
          </button>
          <span className="dim">trust-once decisions are dropped when the session ends</span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-retention">
            Audit retention
          </label>
          <input
            id="pref-retention"
            className="field"
            defaultValue="30 days"
            style={{ width: 110, flex: 'none' }}
          />
          <span className="dim">modification and search logs rotate at 10 MB × 3</span>
        </div>
      </div>
    </>
  );
}
