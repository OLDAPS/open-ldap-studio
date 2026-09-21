/**
 * Screen 5e — Connections & timeouts.
 *
 * The wireframe's provider selector is absent (deviation D5). The read-only
 * and production defaults are present and are the cheapest protection this
 * application offers against a wrong-window mistake (design gap G3).
 */
export function ConnectionsPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Timeouts &amp; retries</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-connect">
            Connect timeout
          </label>
          <input
            id="pref-connect"
            className="field"
            defaultValue="10 s"
            style={{ width: 90, flex: 'none' }}
          />
          <label className="field-row__label field-row__label--auto" htmlFor="pref-response">
            Response timeout
          </label>
          <input
            id="pref-response"
            className="field"
            defaultValue="30 s"
            style={{ width: 90, flex: 'none' }}
          />
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-keepalive">
            Idle keep-alive
          </label>
          <input
            id="pref-keepalive"
            className="field"
            defaultValue="5 min"
            style={{ width: 90, flex: 'none' }}
          />
          <label className="field-row__label field-row__label--auto" htmlFor="pref-retry">
            Auto-reconnect
          </label>
          <input
            id="pref-retry"
            className="field"
            defaultValue="3 tries"
            style={{ width: 90, flex: 'none' }}
          />
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-loss">
            On connection loss
          </label>
          <select id="pref-loss" className="field" defaultValue="ask">
            <option value="silent">reconnect silently</option>
            <option value="ask">ask</option>
            <option value="offline">keep offline</option>
          </select>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Protocol defaults</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Default controls</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              paged results
            </button>
            <button type="button" className="tag">
              ManageDsaIT
            </button>
            <button type="button" className="tag">
              subentries
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-modify">
            Modify requests
          </label>
          <select id="pref-modify" className="field" defaultValue="changed">
            <option value="changed">send changed attributes only</option>
            <option value="replace">replace the whole entry</option>
          </select>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">TLS</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              verify hostname
            </button>
            <button type="button" className="tag" data-selected>
              verify chain
            </button>
          </span>
          <select
            className="field"
            defaultValue="system"
            aria-label="Trust store"
            style={{ width: 180 }}
          >
            <option value="system">trust store: system</option>
          </select>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">New connections</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              open read-only
            </button>
            <button type="button" className="tag" data-selected>
              warn on production tag
            </button>
          </span>
        </div>
      </div>
    </>
  );
}
