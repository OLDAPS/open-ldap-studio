/**
 * The search editor (screen 1c, document area).
 *
 * Base, filter, returning attributes, scope and limits are one card, because
 * they are one request: a filter read without its scope and size limit is a
 * filter whose cost you cannot judge.
 *
 * The raw RFC 4515 field is the primary input and the builder is a way of
 * writing into it, not a parallel model — so what is run is always what is on
 * screen.
 */
import { useState } from 'react';

import { ResultGrid } from './ResultGrid';

export function SearchEditor() {
  const [filter, setFilter] = useState('(objectClass=*)');

  return (
    <>
      <div className="pane" style={{ flex: 'none' }}>
        <div className="card card--tight">
          <div className="field-row">
            <label className="field-row__label" htmlFor="search-base">
              Search base
            </label>
            <input id="search-base" className="field field--mono" placeholder="dc=example,dc=com" />
            <button type="button" className="button">
              Browse…
            </button>
          </div>

          <div className="field-row">
            <label className="field-row__label" htmlFor="search-filter">
              Filter
            </label>
            <input
              id="search-filter"
              className="field field--mono"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              spellCheck={false}
            />
            <button type="button" className="button">
              Builder
            </button>
            <button type="button" className="button">
              Validate
            </button>
          </div>

          <div className="field-row">
            <label className="field-row__label" htmlFor="search-attrs">
              Returning attrs
            </label>
            <input id="search-attrs" className="field field--mono" placeholder="uid, cn, mail, +" />
          </div>

          <div className="field-row field-row--wrap">
            <span className="field-row__label">Scope / limits</span>
            <select className="field" defaultValue="sub" aria-label="Scope" style={{ width: 120 }}>
              <option value="base">base</option>
              <option value="one">one level</option>
              <option value="sub">subtree</option>
            </select>
            <input
              className="field"
              aria-label="Size limit"
              defaultValue="count 1000"
              style={{ width: 110 }}
            />
            <input
              className="field"
              aria-label="Time limit"
              defaultValue="time 10 s"
              style={{ width: 100 }}
            />
            <select
              className="field"
              defaultValue="find"
              aria-label="Aliases"
              style={{ width: 150 }}
            >
              <option value="never">aliases: never</option>
              <option value="find">aliases: finding</option>
              <option value="search">aliases: searching</option>
              <option value="always">aliases: always</option>
            </select>
            <select
              className="field"
              defaultValue="follow"
              aria-label="Referrals"
              style={{ width: 150 }}
            >
              <option value="follow">referrals: follow</option>
              <option value="ignore">referrals: ignore</option>
              <option value="ask">referrals: ask</option>
            </select>

            <span className="actions actions--end">
              <button type="button" className="button">
                Save search
              </button>
              <button type="button" className="button button--primary">
                Run
              </button>
            </span>
          </div>
        </div>
      </div>

      <ResultGrid />
    </>
  );
}
