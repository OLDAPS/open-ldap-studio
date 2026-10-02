import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
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
      <Card className="card">
        <span className="card__label">Behaviour</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Default tab</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              attribute table
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              LDIF view
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              last used
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Save mode</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              explicit (⌘S)
            </Button>
          </span>
          <span className="dim">every commit passes through a preview</span>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Confirm before</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              deleting a value
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              deleting an attribute
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              leaving unsaved
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-schema-check">
            Schema check
          </FieldLabel>
          <NativeSelect id="pref-schema-check" className="field" defaultValue="typing">
            <option value="typing">validate while typing</option>
            <option value="save">on save</option>
            <option value="off">off</option>
          </NativeSelect>
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Table</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Attribute names as</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              short name
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              OID
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              description
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-group">
            Group rows by
          </FieldLabel>
          <NativeSelect id="pref-group" className="field" defaultValue="kind">
            <option value="kind">must / may / operational</option>
            <option value="alpha">alphabetical</option>
            <option value="class">objectClass</option>
          </NativeSelect>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-fold">
            Multi-values
          </FieldLabel>
          <Input
            id="pref-fold"
            className="field"
            type="number"
            defaultValue={3}
            style={{ width: 90, flex: 'none' }}
          />
          <span className="dim">shown before &ldquo;+N more&rdquo;</span>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Copy as</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              LDIF
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              CSV
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              DN only
            </Button>
          </span>
        </Field>
      </Card>
    </>
  );
}
