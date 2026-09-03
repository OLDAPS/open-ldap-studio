/**
 * Screen 5a — Browser & tree.
 *
 * Paging, limits and alias/referral handling are on the same pane as the
 * display options because they are the same decision: how much of a directory
 * this application will pull before it asks.
 */
export function BrowserPane() {
  return (
    <>
      <div className="card">
        <span className="card__label">Paging &amp; limits</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-page">
            Entries per page
          </label>
          <input
            id="pref-page"
            className="field"
            type="number"
            defaultValue={100}
            style={{ width: 90, flex: 'none' }}
          />
          <label className="field-row__label field-row__label--auto" htmlFor="pref-fetch">
            Fetch on scroll
          </label>
          <input id="pref-fetch" type="checkbox" />
          <span className="dim">off means an explicit &ldquo;fetch next&rdquo; row</span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-size">
            Size limit
          </label>
          <input
            id="pref-size"
            className="field"
            type="number"
            defaultValue={1000}
            style={{ width: 90, flex: 'none' }}
          />
          <label className="field-row__label field-row__label--auto" htmlFor="pref-time">
            Time limit
          </label>
          <input
            id="pref-time"
            className="field"
            defaultValue="10 s"
            style={{ width: 90, flex: 'none' }}
          />
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-aliases">
            Aliases · referrals
          </label>
          <select id="pref-aliases" className="field" defaultValue="find">
            <option value="never">dereference: never</option>
            <option value="find">dereference: finding</option>
            <option value="search">dereference: searching</option>
            <option value="always">dereference: always</option>
          </select>
          <select className="field" defaultValue="follow" aria-label="Referrals">
            <option value="follow">referrals: follow</option>
            <option value="ignore">referrals: ignore</option>
            <option value="ask">referrals: ask</option>
          </select>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Display</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Entry label</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              RDN
            </button>
            <button type="button" className="tag">
              full DN
            </button>
            <button type="button" className="tag">
              attribute…
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-sort">
            Sort children by
          </label>
          <select id="pref-sort" className="field" defaultValue="rdn">
            <option value="rdn">RDN</option>
            <option value="none">server order</option>
          </select>
        </div>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Show</span>
          <span className="tag-set">
            <button type="button" className="tag" data-selected>
              child count
            </button>
            <button type="button" className="tag">
              operational attrs
            </button>
            <button type="button" className="tag">
              subentries
            </button>
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-expand">
            Expand on connect
          </label>
          <select id="pref-expand" className="field" defaultValue="first">
            <option value="first">first naming context</option>
            <option value="none">none</option>
            <option value="last">last session</option>
          </select>
        </div>
      </div>
    </>
  );
}
