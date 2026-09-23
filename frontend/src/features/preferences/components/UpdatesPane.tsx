/**
 * Screen 5h — Updates & about.
 *
 * Update checks default to off (FR-016, deviation D9): this application does
 * not reach the network unless the user pointed it somewhere. The extension
 * list the wireframe drew is dropped with the plugin runtime.
 */
export function UpdatesPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Updates</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-check">
            Check for updates
          </label>
          <select id="pref-check" className="field" defaultValue="never">
            <option value="never">never (default)</option>
            <option value="startup">on start-up</option>
            <option value="weekly">weekly</option>
          </select>
          <button type="button" className="button">
            Check now
          </button>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-proxy">
            Proxy
          </label>
          <select id="pref-proxy" className="field" defaultValue="system">
            <option value="system">system proxy</option>
            <option value="direct">direct</option>
            <option value="manual">manual…</option>
          </select>
        </div>
        <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
          Off by default is deliberate: an update check is an outbound connection, and this
          application makes none the user did not ask for.
        </p>
      </div>

      <div className="card">
        <span className="card__label">About</span>
        <div className="field-row">
          <span className="field-row__label">Version</span>
          <span className="mono">development build</span>
        </div>
        <div className="field-row">
          <span className="field-row__label">Licence</span>
          <span className="dim">open source</span>
        </div>
      </div>
    </>
  );
}
