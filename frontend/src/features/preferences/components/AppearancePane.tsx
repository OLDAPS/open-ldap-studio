import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
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
      <Card className="card">
        <span className="card__label">Colours</span>

        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Theme</span>
          <ToggleGroup
            className="tag-set"
            value={[theme]}
            onValueChange={(value) => value[0] && setTheme(value[0] as Theme)}
          >
            {THEMES.map(([id, label]) => (
              <ToggleGroupItem key={id} value={id} className="tag">
                {label}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </Field>

        <Field orientation="horizontal" className="field-row">
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
        </Field>

        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-syntax">
            LDIF syntax colours
          </FieldLabel>
          <span id="pref-syntax" className="field field--static">
            dn · attribute · value · changetype · comment · error
          </span>
          <Button type="button" variant="outline">
            Edit tokens…
          </Button>
        </Field>

        <Field orientation="horizontal" className="field-row field-row--wrap">
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
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Fonts</span>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-ui-font">
            Interface font
          </FieldLabel>
          <NativeSelect id="pref-ui-font" className="field" defaultValue="system">
            <option value="system">System UI</option>
          </NativeSelect>
          <Input
            className="field"
            defaultValue="14 px"
            aria-label="Interface font size"
            style={{ width: 76, flex: 'none' }}
          />
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-mono-font">
            Monospace font
          </FieldLabel>
          <NativeSelect id="pref-mono-font" className="field" defaultValue="system">
            <option value="system">System monospace</option>
          </NativeSelect>
          <Input
            className="field"
            defaultValue="12 px"
            aria-label="Monospace font size"
            style={{ width: 76, flex: 'none' }}
          />
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Sizes &amp; density</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Row density</span>
          <ToggleGroup
            className="tag-set"
            value={[density]}
            onValueChange={(value) => value[0] && setDensity(value[0] as Density)}
          >
            {DENSITIES.map(([id, label]) => (
              <ToggleGroupItem key={id} value={id} className="tag">
                {label}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-truncate">
            Truncate values at
          </FieldLabel>
          <Input
            id="pref-truncate"
            className="field"
            type="number"
            defaultValue={120}
            style={{ width: 90, flex: 'none' }}
          />
          <span className="dim">characters · the full value is always in the editor</span>
        </Field>
      </Card>
    </>
  );
}
