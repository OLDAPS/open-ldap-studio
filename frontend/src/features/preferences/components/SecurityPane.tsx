import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
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
      <Card className="card">
        <span className="card__label">Secret storage</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Storage</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              platform credential store
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              session only
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              never store
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <span className="field-row__label">Unlock</span>
          <span className="dim">
            deferred to the first bind and raised by the platform agent — this application never
            prompts for a master password of its own
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-prompt">
            Password prompt
          </FieldLabel>
          <NativeSelect id="pref-prompt" className="field" defaultValue="session">
            <option value="session">once per session</option>
            <option value="bind">every bind</option>
          </NativeSelect>
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Policy</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Plaintext bind</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag">
              warn
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              block unless StartTLS
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              allow
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">On exit</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              clear session secrets
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              clear DN history
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <span className="field-row__label">Trusted certificates</span>
          <Button type="button" variant="outline">
            Manage store…
          </Button>
          <span className="dim">trust-once decisions are dropped when the session ends</span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-retention">
            Audit retention
          </FieldLabel>
          <Input
            id="pref-retention"
            className="field"
            defaultValue="30 days"
            style={{ width: 110, flex: 'none' }}
          />
          <span className="dim">modification and search logs rotate at 10 MB × 3</span>
        </Field>
      </Card>
    </>
  );
}
