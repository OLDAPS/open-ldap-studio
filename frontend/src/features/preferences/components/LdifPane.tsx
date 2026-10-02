import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
/**
 * Screen 5d — LDIF & text editors.
 *
 * Everything here is output layout: wrap width, folding, encoding. None of it
 * touches value fidelity — a value is never altered to make a line fit.
 */
export function LdifPane() {
  return (
    <>
      <Card className="card">
        <span className="card__label">Formatting</span>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-wrap">
            Wrap lines at
          </FieldLabel>
          <Input
            id="pref-wrap"
            className="field"
            type="number"
            defaultValue={78}
            style={{ width: 90, flex: 'none' }}
          />
          <FieldLabel className="field-row__label field-row__label--auto" htmlFor="pref-b64">
            Fold base64 at
          </FieldLabel>
          <Input
            id="pref-b64"
            className="field"
            type="number"
            defaultValue={76}
            style={{ width: 90, flex: 'none' }}
          />
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Show</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              line numbers
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              whitespace
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              record folding
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-encoding">
            Encoding · EOL
          </FieldLabel>
          <NativeSelect id="pref-encoding" className="field" defaultValue="utf8">
            <option value="utf8">UTF-8</option>
          </NativeSelect>
          <NativeSelect className="field" defaultValue="lf" aria-label="Line ending">
            <option value="lf">LF</option>
            <option value="crlf">CRLF</option>
          </NativeSelect>
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Validation &amp; syntax</span>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-validate">
            Validate
          </FieldLabel>
          <NativeSelect id="pref-validate" className="field" defaultValue="typing">
            <option value="typing">while typing</option>
            <option value="save">on save</option>
            <option value="manual">manual</option>
          </NativeSelect>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Flag</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              unknown attributes
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              schema violations
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              missing DNs
            </Button>
          </span>
        </Field>
        <div className="placeholder" style={{ height: 56 }}>
          preview strip of a coloured LDIF record
          <br />
          with a warning gutter marker
        </div>
      </Card>
    </>
  );
}
