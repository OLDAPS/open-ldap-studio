import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
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
      <Card className="card">
        <span className="card__label">Timeouts &amp; retries</span>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-connect">
            Connect timeout
          </FieldLabel>
          <Input
            id="pref-connect"
            className="field"
            defaultValue="10 s"
            style={{ width: 90, flex: 'none' }}
          />
          <FieldLabel className="field-row__label field-row__label--auto" htmlFor="pref-response">
            Response timeout
          </FieldLabel>
          <Input
            id="pref-response"
            className="field"
            defaultValue="30 s"
            style={{ width: 90, flex: 'none' }}
          />
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-keepalive">
            Idle keep-alive
          </FieldLabel>
          <Input
            id="pref-keepalive"
            className="field"
            defaultValue="5 min"
            style={{ width: 90, flex: 'none' }}
          />
          <FieldLabel className="field-row__label field-row__label--auto" htmlFor="pref-retry">
            Auto-reconnect
          </FieldLabel>
          <Input
            id="pref-retry"
            className="field"
            defaultValue="3 tries"
            style={{ width: 90, flex: 'none' }}
          />
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-loss">
            On connection loss
          </FieldLabel>
          <NativeSelect id="pref-loss" className="field" defaultValue="ask">
            <option value="silent">reconnect silently</option>
            <option value="ask">ask</option>
            <option value="offline">keep offline</option>
          </NativeSelect>
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Protocol defaults</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Default controls</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              paged results
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              ManageDsaIT
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              subentries
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-modify">
            Modify requests
          </FieldLabel>
          <NativeSelect id="pref-modify" className="field" defaultValue="changed">
            <option value="changed">send changed attributes only</option>
            <option value="replace">replace the whole entry</option>
          </NativeSelect>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">TLS</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              verify hostname
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              verify chain
            </Button>
          </span>
          <NativeSelect
            className="field"
            defaultValue="system"
            aria-label="Trust store"
            style={{ width: 180 }}
          >
            <option value="system">trust store: system</option>
          </NativeSelect>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">New connections</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              open read-only
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              warn on production tag
            </Button>
          </span>
        </Field>
      </Card>
    </>
  );
}
