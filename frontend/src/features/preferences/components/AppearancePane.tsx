/**
 * Screen 3a — Appearance: colours, fonts, sizes.
 *
 * Theme and density are wired to the session store, which is what stamps
 * data-theme and data-density on the root, so the choice applies as it is made
 * — the screen promises a live preview and this is the cheapest honest way to
 * keep that promise.
 *
 * High contrast and the colourblind-safe diff pair are here as first-class
 * choices rather than an accessibility afterthought (design gap G2).
 */
import type { Density, Theme } from '@/app/session';
import { useSession } from '@/app/session';

const THEMES: [Theme, string][] = [
  ['dark', 'Dark'],
  ['light', 'Light'],
  ['high-contrast', 'High contrast'],
  ['system', 'Follow system'],
];

const DENSITIES: [Density, string][] = [
  ['compact', 'Compact'],
  ['comfortable', 'Comfortable'],
];

export function AppearancePane() {
  const theme = useSession((s) => s.theme);
  const setTheme = useSession((s) => s.setTheme);
  const density = useSession((s) => s.density);
  const setDensity = useSession((s) => s.setDensity);

  return (
    <>
      <div className="card">
        <span className="card__label">Colours</span>

        <div className="field-row field-row--wrap">
          <span className="field-row__label">Theme</span>
          <span className="tag-set">
            {THEMES.map(([id, label]) => (
              <button
                key={id}
                type="button"
                className="tag"
                data-selected={theme === id || undefined}
                onClick={() => setTheme(id)}
              >
                {label}
              </button>
            ))}
          </span>
        </div>

        <div className="field-row">
          <span className="field-row__label">Accent</span>
          <span className="field-group">
            <span
              aria-hidden="true"
              style={{
                width: 20,
                height: 20,
                background: 'var(--accent)',
                border: '1px solid var(--border-strong)',
                borderRadius: 'var(--radius-chrome)',
              }}
            />
            <span
              aria-hidden="true"
              style={{
                width: 20,
                height: 20,
                background: 'var(--accent-quiet)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-chrome)',
              }}
            />
            <span className="dim">accent and its quiet pair, used for selection and focus</span>
          </span>
        </div>

        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-syntax">
            LDIF syntax colours
          </label>
          <span id="pref-syntax" className="field field--static">
            dn · attribute · value · changetype · comment · error
          </span>
          <button type="button" className="button">
            Edit tokens…
          </button>
        </div>

        <div className="field-row field-row--wrap">
          <span className="field-row__label">Diff highlight</span>
          <span className="tag-set">
            <span
              className="tag"
              style={{ borderColor: 'var(--diff-added)', color: 'var(--diff-added)' }}
            >
              + added
            </span>
            <span
              className="tag"
              style={{ borderColor: 'var(--diff-removed)', color: 'var(--diff-removed)' }}
            >
              − removed
            </span>
            <span
              className="tag"
              style={{ borderColor: 'var(--diff-changed)', color: 'var(--diff-changed)' }}
            >
              ± changed
            </span>
            <span className="dim">paired shape with colour, never colour alone</span>
          </span>
        </div>
      </div>

      <div className="card">
        <span className="card__label">Fonts</span>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-ui-font">
            Interface font
          </label>
          <select id="pref-ui-font" className="field" defaultValue="system">
            <option value="system">System UI</option>
          </select>
          <input
            className="field"
            defaultValue="14 px"
            aria-label="Interface font size"
            style={{ width: 76, flex: 'none' }}
          />
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-mono-font">
            Monospace font
          </label>
          <select id="pref-mono-font" className="field" defaultValue="system">
            <option value="system">System monospace</option>
          </select>
          <input
            className="field"
            defaultValue="12 px"
            aria-label="Monospace font size"
            style={{ width: 76, flex: 'none' }}
          />
        </div>
      </div>

      <div className="card">
        <span className="card__label">Sizes &amp; density</span>
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Row density</span>
          <span className="tag-set">
            {DENSITIES.map(([id, label]) => (
              <button
                key={id}
                type="button"
                className="tag"
                data-selected={density === id || undefined}
                onClick={() => setDensity(id)}
              >
                {label}
              </button>
            ))}
          </span>
        </div>
        <div className="field-row">
          <label className="field-row__label" htmlFor="pref-truncate">
            Truncate values at
          </label>
          <input
            id="pref-truncate"
            className="field"
            type="number"
            defaultValue={120}
            style={{ width: 90, flex: 'none' }}
          />
          <span className="dim">characters · the full value is always in the editor</span>
        </div>
      </div>
    </>
  );
}
